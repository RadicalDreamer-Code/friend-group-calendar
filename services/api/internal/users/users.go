// Package users manages profile data: usernames and avatars.
package users

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/avatar"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/blob"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/store"
)

var (
	ErrInvalidUsername = errors.New("usernames are 3–20 characters: lowercase letters, numbers, _ and .")
	ErrUsernameTaken   = errors.New("username is taken")
)

var usernamePattern = regexp.MustCompile(`^[a-z0-9_.]{3,20}$`)

// avatarKeyPattern matches the keys SetAvatar creates: <user id>/<random>.webp.
var avatarKeyPattern = regexp.MustCompile(`^[0-9a-f-]{36}/[0-9a-f]{32}\.webp$`)

// NormalizeUsername lowercases and validates a username.
func NormalizeUsername(raw string) (string, error) {
	name := strings.ToLower(strings.TrimSpace(raw))
	if !usernamePattern.MatchString(name) {
		return "", ErrInvalidUsername
	}
	return name, nil
}

// ValidAvatarKey reports whether key has the shape of a stored avatar.
func ValidAvatarKey(key string) bool {
	return avatarKeyPattern.MatchString(key)
}

type Service struct {
	db      *pgxpool.Pool
	q       *store.Queries
	avatars blob.Store
	images  *avatar.Processor
}

func NewService(db *pgxpool.Pool, avatars blob.Store) *Service {
	return &Service{db: db, q: store.New(db), avatars: avatars, images: avatar.NewProcessor()}
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (store.User, error) {
	return s.q.GetUser(ctx, id)
}

// UsernameAvailable reports whether the user could take this username.
// Their own current username counts as available.
func (s *Service) UsernameAvailable(ctx context.Context, userID uuid.UUID, raw string) (bool, error) {
	name, err := NormalizeUsername(raw)
	if err != nil {
		return false, err
	}
	taken, err := s.q.UsernameTaken(ctx, store.UsernameTakenParams{Username: name, ExceptUserID: userID})
	return !taken, err
}

func (s *Service) SetUsername(ctx context.Context, userID uuid.UUID, raw string) (store.User, error) {
	name, err := NormalizeUsername(raw)
	if err != nil {
		return store.User{}, err
	}
	user, err := s.q.UpdateUsername(ctx, store.UpdateUsernameParams{ID: userID, Username: name})
	if isUniqueViolation(err) {
		return store.User{}, ErrUsernameTaken
	}
	return user, err
}

// SetAvatar processes and stores a new avatar and deletes the previous one.
func (s *Service) SetAvatar(ctx context.Context, userID uuid.UUID, upload []byte) (store.User, error) {
	img, err := s.images.Process(ctx, upload)
	if err != nil {
		return store.User{}, err
	}

	// A fresh key per upload lets clients cache avatars forever.
	key := fmt.Sprintf("%s/%s.webp", userID, randomHex(16))
	if err := s.avatars.Put(ctx, key, img, "image/webp"); err != nil {
		return store.User{}, err
	}
	user, previous, err := s.replaceAvatarKey(ctx, userID, &key)
	if err != nil {
		s.deleteAvatar(ctx, &key)
		return store.User{}, err
	}
	s.deleteAvatar(ctx, previous)
	return user, nil
}

func (s *Service) RemoveAvatar(ctx context.Context, userID uuid.UUID) (store.User, error) {
	user, previous, err := s.replaceAvatarKey(ctx, userID, nil)
	if err != nil {
		return store.User{}, err
	}
	s.deleteAvatar(ctx, previous)
	return user, nil
}

// replaceAvatarKey sets the new key and returns the one it replaced. The row
// lock keeps two parallel uploads from both missing the other's old file.
func (s *Service) replaceAvatarKey(ctx context.Context, userID uuid.UUID, key *string) (store.User, *string, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return store.User{}, nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after Commit
	q := s.q.WithTx(tx)

	current, err := q.GetUserForUpdate(ctx, userID)
	if err != nil {
		return store.User{}, nil, err
	}
	user, err := q.SetAvatarKey(ctx, store.SetAvatarKeyParams{ID: userID, AvatarKey: key})
	if err != nil {
		return store.User{}, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return store.User{}, nil, err
	}
	return user, current.AvatarKey, nil
}

// deleteAvatar removes a stored avatar. Failing leaves an orphaned file,
// which is not worth failing the request for.
func (s *Service) deleteAvatar(ctx context.Context, key *string) {
	if key == nil {
		return
	}
	if err := s.avatars.Delete(context.WithoutCancel(ctx), *key); err != nil {
		slog.WarnContext(ctx, "delete old avatar", "key", *key, "err", err)
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func randomHex(n int) string {
	buf := make([]byte, n)
	_, _ = rand.Read(buf) // crypto/rand.Read never returns an error
	return hex.EncodeToString(buf)
}
