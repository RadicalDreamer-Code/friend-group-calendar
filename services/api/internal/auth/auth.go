// Package auth implements passwordless sign-in with emailed codes and the
// long-lived sessions that follow it.
//
// Flow: RequestCode emails a 6-digit code and returns a challenge ID to the
// client that asked for it; Verify checks the code against that challenge,
// creates the user on their first sign-in and returns a new session token.
// The token goes into an HttpOnly cookie on web and into secure storage on
// native; Authenticate turns it back into a Principal on every request.
package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	mailer "github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/mail"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/ratelimit"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/store"
)

const (
	CodeTTL         = 10 * time.Minute
	MaxCodeAttempts = 5
	SessionTTL      = 90 * 24 * time.Hour
	// Sessions are renewed at most this often, so normal use does not write
	// to the database on every request.
	renewInterval = time.Hour
	// Revoked and expired sessions are deleted after this long.
	staleSessionRetention = 30 * 24 * time.Hour

	codeDigits       = 6
	tokenBytes       = 32
	maxEmailLen      = 254
	maxDeviceNameLen = 64
)

var (
	ErrInvalidEmail    = errors.New("invalid email address")
	ErrInvalidPlatform = errors.New("invalid platform")
	// ErrInvalidCode: a code exists but this one does not match it.
	ErrInvalidCode = errors.New("invalid code")
	// ErrCodeExpired: no usable code (never requested, expired, used, or out of attempts).
	ErrCodeExpired     = errors.New("code expired")
	ErrUnauthenticated = errors.New("unauthenticated")
)

// RateLimitError is returned when a caller has to wait before trying again.
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limited, retry after %s", e.RetryAfter.Round(time.Second))
}

type Platform string

const (
	PlatformWeb     Platform = "web"
	PlatformIOS     Platform = "ios"
	PlatformAndroid Platform = "android"
)

func (p Platform) Valid() bool {
	return p == PlatformWeb || p == PlatformIOS || p == PlatformAndroid
}

// Principal is the signed-in user and the session they are using.
type Principal struct {
	User    store.User
	Session store.Session
}

type Service struct {
	db     *pgxpool.Pool
	q      *store.Queries
	mail   mailer.Sender
	secret []byte
	now    func() time.Time

	emailCooldown     *ratelimit.Limiter
	clientEmailHourly *ratelimit.Limiter
	emailHourly       *ratelimit.Limiter
	ipCodes           *ratelimit.Limiter
	ipVerify          *ratelimit.Limiter
}

// Limit allows N events per Window.
type Limit struct {
	N      int
	Window time.Duration
}

// The per-address limits that a sign-in could trip over are counted per
// client, so someone else asking for codes to the same address cannot lock
// its owner out. Clients are IP addresses (IPv6: their /64).
type Limits struct {
	// EmailCooldown spaces out codes from one client to one address, so a
	// double tap does not send a second code while the first is on its way.
	EmailCooldown Limit
	// ClientEmailHourly caps the codes one client requests for one address.
	ClientEmailHourly Limit
	// EmailHourly caps the codes sent to one address by all clients together,
	// against flooding a mailbox. It is a multiple of ClientEmailHourly, so
	// it takes several clients to use it up. It also bounds guessing: at most
	// EmailHourly.N × MaxCodeAttempts guesses per address and hour, each
	// costing the attacker a code that lands in the owner's inbox.
	EmailHourly Limit
	// IPCodes caps code requests per client IP against mass sending.
	IPCodes Limit
	// IPVerify caps verify attempts per client IP, on top of MaxCodeAttempts
	// per code, so one client cannot guess across many addresses.
	IPVerify Limit
}

var DefaultLimits = Limits{
	EmailCooldown:     Limit{1, 30 * time.Second},
	ClientEmailHourly: Limit{5, time.Hour},
	EmailHourly:       Limit{10, time.Hour},
	IPCodes:           Limit{20, time.Hour},
	IPVerify:          Limit{30, 15 * time.Minute},
}

func NewService(db *pgxpool.Pool, sender mailer.Sender, secret []byte, limits Limits) *Service {
	return &Service{
		db:                db,
		q:                 store.New(db),
		mail:              sender,
		secret:            secret,
		now:               time.Now,
		emailCooldown:     ratelimit.New(limits.EmailCooldown.N, limits.EmailCooldown.Window),
		clientEmailHourly: ratelimit.New(limits.ClientEmailHourly.N, limits.ClientEmailHourly.Window),
		emailHourly:       ratelimit.New(limits.EmailHourly.N, limits.EmailHourly.Window),
		ipCodes:           ratelimit.New(limits.IPCodes.N, limits.IPCodes.Window),
		ipVerify:          ratelimit.New(limits.IPVerify.N, limits.IPVerify.Window),
	}
}

// NormalizeEmail trims and lowercases an address and rejects anything that
// is not a bare address with a dotted domain.
func NormalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" || len(email) > maxEmailLen {
		return "", ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return "", ErrInvalidEmail
	}
	at := strings.LastIndexByte(email, '@')
	if domain := email[at+1:]; !strings.Contains(domain, ".") || strings.HasSuffix(domain, ".") {
		return "", ErrInvalidEmail
	}
	return email, nil
}

// RequestCode emails a fresh code to the address and returns the challenge
// ID that Verify needs along with it. Only the caller learns the ID, so codes
// requested by anyone else for the same address do not affect this one.
// It behaves the same whether or not an account exists for the address.
func (s *Service) RequestCode(ctx context.Context, rawEmail, client string) (challengeID uuid.UUID, err error) {
	email, err := NormalizeEmail(rawEmail)
	if err != nil {
		return uuid.Nil, err
	}
	clientEmail := client + "\x00" + email
	for _, check := range []struct {
		l   *ratelimit.Limiter
		key string
	}{
		{s.ipCodes, client},
		{s.emailCooldown, clientEmail},
		{s.clientEmailHourly, clientEmail},
		{s.emailHourly, email},
	} {
		if ok, wait := check.l.Allow(check.key); !ok {
			return uuid.Nil, &RateLimitError{RetryAfter: wait}
		}
	}

	code, err := generateCode()
	if err != nil {
		return uuid.Nil, err
	}
	now := s.now()
	challengeID, err = s.q.CreateLoginCode(ctx, store.CreateLoginCodeParams{
		Email:     email,
		CodeHash:  s.hashCode(email, code),
		ExpiresAt: now.Add(CodeTTL),
		CreatedAt: now,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("store login code: %w", err)
	}
	if err := s.mail.Send(ctx, codeEmail(email, code)); err != nil {
		return uuid.Nil, err
	}
	return challengeID, nil
}

type VerifyInput struct {
	// ChallengeID is what RequestCode returned for this code.
	ChallengeID uuid.UUID
	Email       string
	Code        string
	Platform    Platform
	// DeviceName is shown in the device list. Optional; derived from the
	// user agent when empty.
	DeviceName string
	UserAgent  string
	// Client identifies the caller for rate limits, like RequestCode's client.
	Client string
}

type VerifyResult struct {
	User store.User
	// Token is only ever shown to the client this once.
	Token     string
	ExpiresAt time.Time
}

// Verify checks the code, creates the user on first sign-in and starts a session.
func (s *Service) Verify(ctx context.Context, in VerifyInput) (VerifyResult, error) {
	email, err := NormalizeEmail(in.Email)
	if err != nil {
		return VerifyResult{}, err
	}
	if !in.Platform.Valid() {
		return VerifyResult{}, ErrInvalidPlatform
	}
	code := strings.Join(strings.Fields(in.Code), "")
	if !isCode(code) {
		return VerifyResult{}, ErrInvalidCode
	}
	if ok, wait := s.ipVerify.Allow(in.Client); !ok {
		return VerifyResult{}, &RateLimitError{RetryAfter: wait}
	}

	now := s.now()
	// Counting the attempt commits on its own, so wrong guesses are counted
	// even though the rest of the sign-in never happens.
	storedHash, err := s.q.AttemptLoginCode(ctx, store.AttemptLoginCodeParams{
		ID:          in.ChallengeID,
		Email:       email,
		Now:         now,
		MaxAttempts: MaxCodeAttempts,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return VerifyResult{}, ErrCodeExpired
	}
	if err != nil {
		return VerifyResult{}, fmt.Errorf("check login code: %w", err)
	}
	if !hmac.Equal(storedHash, s.hashCode(email, code)) {
		return VerifyResult{}, ErrInvalidCode
	}

	token, err := generateToken()
	if err != nil {
		return VerifyResult{}, err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return VerifyResult{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after Commit
	q := s.q.WithTx(tx)

	consumed, err := q.ConsumeLoginCode(ctx, in.ChallengeID)
	if err != nil {
		return VerifyResult{}, fmt.Errorf("consume login code: %w", err)
	}
	if consumed == 0 {
		// A parallel request used the same code first.
		return VerifyResult{}, ErrCodeExpired
	}
	user, err := q.UpsertUserByEmail(ctx, email)
	if err != nil {
		return VerifyResult{}, fmt.Errorf("upsert user: %w", err)
	}
	session, err := q.CreateSession(ctx, store.CreateSessionParams{
		UserID:     user.ID,
		TokenHash:  hashToken(token),
		DeviceName: deviceName(in.DeviceName, in.UserAgent, in.Platform),
		Platform:   string(in.Platform),
		UserAgent:  truncate(in.UserAgent, 512),
		CreatedAt:  now,
		ExpiresAt:  now.Add(SessionTTL),
	})
	if err != nil {
		return VerifyResult{}, fmt.Errorf("create session: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return VerifyResult{}, err
	}
	return VerifyResult{User: user, Token: token, ExpiresAt: session.ExpiresAt}, nil
}

// Authenticate resolves a session token. renewed reports that the session's
// expiry moved forward, so a web client should get a fresh cookie.
func (s *Service) Authenticate(ctx context.Context, token string) (p Principal, renewed bool, err error) {
	if !looksLikeToken(token) {
		return Principal{}, false, ErrUnauthenticated
	}
	now := s.now()
	row, err := s.q.GetActiveSessionByTokenHash(ctx, store.GetActiveSessionByTokenHashParams{
		TokenHash: hashToken(token),
		Now:       now,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Principal{}, false, ErrUnauthenticated
	}
	if err != nil {
		return Principal{}, false, fmt.Errorf("load session: %w", err)
	}
	p = Principal{User: row.User, Session: row.Session}

	if now.Sub(p.Session.LastUsedAt) >= renewInterval {
		expires := now.Add(SessionTTL)
		if err := s.q.RenewSession(ctx, store.RenewSessionParams{ID: p.Session.ID, LastUsedAt: now, ExpiresAt: expires}); err != nil {
			return Principal{}, false, fmt.Errorf("renew session: %w", err)
		}
		if err := s.q.TouchUserLastSeen(ctx, store.TouchUserLastSeenParams{ID: p.User.ID, LastSeenAt: now}); err != nil {
			return Principal{}, false, fmt.Errorf("touch user: %w", err)
		}
		p.Session.LastUsedAt, p.Session.ExpiresAt, p.User.LastSeenAt = now, expires, now
		renewed = true
	}
	return p, renewed, nil
}

func (s *Service) ListSessions(ctx context.Context, userID uuid.UUID) ([]store.Session, error) {
	return s.q.ListActiveSessions(ctx, store.ListActiveSessionsParams{UserID: userID, Now: s.now()})
}

// RevokeSession signs one of the user's sessions out. It reports false if
// the session does not exist, belongs to someone else or is already revoked.
func (s *Service) RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) (bool, error) {
	n, err := s.q.RevokeSession(ctx, store.RevokeSessionParams{Now: s.now(), ID: sessionID, UserID: userID})
	return n > 0, err
}

// RevokeOtherSessions signs out every session of the user except keep.
func (s *Service) RevokeOtherSessions(ctx context.Context, userID, keep uuid.UUID) (int64, error) {
	return s.q.RevokeOtherSessions(ctx, store.RevokeOtherSessionsParams{Now: s.now(), UserID: userID, KeepID: keep})
}

// Cleanup removes expired codes and long-dead sessions.
func (s *Service) Cleanup(ctx context.Context) error {
	now := s.now()
	if _, err := s.q.DeleteExpiredLoginCodes(ctx, now); err != nil {
		return fmt.Errorf("delete expired login codes: %w", err)
	}
	if _, err := s.q.DeleteStaleSessions(ctx, now.Add(-staleSessionRetention)); err != nil {
		return fmt.Errorf("delete stale sessions: %w", err)
	}
	return nil
}

// hashCode binds the code to the address, so a stored hash says nothing
// about codes sent to anyone else.
func (s *Service) hashCode(email, code string) []byte {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte("login-code\x00" + email + "\x00" + code))
	return mac.Sum(nil)
}

// Session tokens carry 256 random bits, so a plain SHA-256 is enough to
// make a leaked database useless for signing in.
func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func generateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", fmt.Errorf("generate code: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func generateToken() (string, error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func isCode(s string) bool {
	if len(s) != codeDigits {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func looksLikeToken(s string) bool {
	return len(s) == base64.RawURLEncoding.EncodedLen(tokenBytes)
}

func deviceName(given, userAgent string, platform Platform) string {
	if name := strings.TrimSpace(given); name != "" {
		return truncate(name, maxDeviceNameLen)
	}
	if name := describeUserAgent(userAgent); name != "" {
		return name
	}
	switch platform {
	case PlatformIOS:
		return "iOS device"
	case PlatformAndroid:
		return "Android device"
	default:
		return "Web browser"
	}
}

// truncate cuts s to at most n bytes without splitting a UTF-8 character.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}
