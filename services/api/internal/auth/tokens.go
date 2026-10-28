package auth

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/store"
)

// Personal access tokens let an AI assistant or another MCP client act for
// a user without a sign-in code. Only the MCP endpoint accepts them: the
// REST API looks bearer tokens up in sessions, where these never are.
const (
	// APITokenPrefix marks access tokens, so they are easy to tell apart from
	// session tokens and to find when they leak.
	APITokenPrefix = "fgc_"
	// MaxAPITokens is how many live tokens a user can have at once.
	MaxAPITokens       = 10
	MaxAPITokenNameLen = 40
	// DefaultAPITokenDays is the expiry when the user does not pick one.
	DefaultAPITokenDays = 90
	// last_used_at is updated at most this often.
	tokenTouchInterval = time.Minute
)

// APITokenDays are the expiries a user can pick, in days.
var APITokenDays = []int{30, 90, 365}

var (
	ErrInvalidTokenName   = fmt.Errorf("token names are 1–%d characters", MaxAPITokenNameLen)
	ErrInvalidTokenExpiry = errors.New("tokens expire after 30, 90 or 365 days")
	ErrTooManyTokens      = fmt.Errorf("you can have at most %d access tokens", MaxAPITokens)
)

// NormalizeTokenName trims a token name and checks its length.
func NormalizeTokenName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	n := utf8.RuneCountInString(name)
	if n == 0 || n > MaxAPITokenNameLen || strings.ContainsFunc(name, unicode.IsControl) {
		return "", ErrInvalidTokenName
	}
	return name, nil
}

type CreateAPITokenResult struct {
	APIToken store.ApiToken
	// Token is only ever shown to the user this once.
	Token string
}

// CreateAPIToken makes a new access token that expires after `days` days.
func (s *Service) CreateAPIToken(ctx context.Context, userID uuid.UUID, rawName string, days int) (CreateAPITokenResult, error) {
	name, err := NormalizeTokenName(rawName)
	if err != nil {
		return CreateAPITokenResult{}, err
	}
	if !slices.Contains(APITokenDays, days) {
		return CreateAPITokenResult{}, ErrInvalidTokenExpiry
	}
	if ok, wait := s.tokenCreates.Allow(userID.String()); !ok {
		return CreateAPITokenResult{}, &RateLimitError{RetryAfter: wait}
	}
	secret, err := generateToken()
	if err != nil {
		return CreateAPITokenResult{}, err
	}
	token := APITokenPrefix + secret

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return CreateAPITokenResult{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after Commit
	q := s.q.WithTx(tx)

	// Locking the user makes the count and the insert one step, so parallel
	// requests cannot go over the limit together.
	if _, err := q.GetUserForUpdate(ctx, userID); err != nil {
		return CreateAPITokenResult{}, fmt.Errorf("lock user: %w", err)
	}
	now := s.now()
	live, err := q.CountLiveAPITokens(ctx, store.CountLiveAPITokensParams{UserID: userID, Now: now})
	if err != nil {
		return CreateAPITokenResult{}, fmt.Errorf("count tokens: %w", err)
	}
	if live >= MaxAPITokens {
		return CreateAPITokenResult{}, ErrTooManyTokens
	}
	t, err := q.CreateAPIToken(ctx, store.CreateAPITokenParams{
		UserID:    userID,
		Name:      name,
		TokenHash: hashToken(token),
		CreatedAt: now,
		// Whole 24-hour days, whatever the server's time zone.
		ExpiresAt: now.Add(time.Duration(days) * 24 * time.Hour),
	})
	if err != nil {
		return CreateAPITokenResult{}, fmt.Errorf("create token: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return CreateAPITokenResult{}, err
	}
	return CreateAPITokenResult{APIToken: t, Token: token}, nil
}

// ListAPITokens returns the user's tokens that are neither expired nor
// revoked, newest first.
func (s *Service) ListAPITokens(ctx context.Context, userID uuid.UUID) ([]store.ApiToken, error) {
	return s.q.ListLiveAPITokens(ctx, store.ListLiveAPITokensParams{UserID: userID, Now: s.now()})
}

// RevokeAPIToken ends one of the user's tokens. It reports false if the
// token does not exist, belongs to someone else or has already ended.
func (s *Service) RevokeAPIToken(ctx context.Context, userID, tokenID uuid.UUID) (bool, error) {
	n, err := s.q.RevokeAPIToken(ctx, store.RevokeAPITokenParams{Now: s.now(), ID: tokenID, UserID: userID})
	return n > 0, err
}

// AuthenticateToken resolves an access token. The Principal carries the
// token's ID and no session.
func (s *Service) AuthenticateToken(ctx context.Context, token string) (Principal, error) {
	if !looksLikeAPIToken(token) {
		return Principal{}, ErrUnauthenticated
	}
	now := s.now()
	row, err := s.q.GetLiveAPITokenByHash(ctx, store.GetLiveAPITokenByHashParams{TokenHash: hashToken(token), Now: now})
	if errors.Is(err, pgx.ErrNoRows) {
		return Principal{}, ErrUnauthenticated
	}
	if err != nil {
		return Principal{}, fmt.Errorf("load token: %w", err)
	}
	t := row.ApiToken
	if t.LastUsedAt == nil || now.Sub(*t.LastUsedAt) >= tokenTouchInterval {
		if err := s.q.TouchAPIToken(ctx, store.TouchAPITokenParams{ID: t.ID, Now: now}); err != nil {
			return Principal{}, fmt.Errorf("touch token: %w", err)
		}
		t.LastUsedAt = &now
	}
	return Principal{User: row.User, TokenID: t.ID}, nil
}

func looksLikeAPIToken(s string) bool {
	secret, ok := strings.CutPrefix(s, APITokenPrefix)
	return ok && looksLikeToken(secret)
}
