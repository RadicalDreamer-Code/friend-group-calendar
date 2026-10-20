// Package friends manages friend requests and friendships, and finds people
// by username.
//
// Each pair of users has at most one friendships row: a pending request from
// one to the other, or an accepted friendship. Declining a request,
// cancelling it and removing a friend all delete that row.
package friends

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/ratelimit"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/store"
)

// MaxSearchResults bounds one username search.
const MaxSearchResults = 20

var (
	ErrInvalidQuery = errors.New("search for 2–20 characters of a username")
	// ErrUserNotFound: no such user, or one who has not picked a username yet
	// and so cannot be found or added.
	ErrUserNotFound     = errors.New("user not found")
	ErrSelf             = errors.New("you cannot add yourself")
	ErrUsernameRequired = errors.New("pick a username first")
	ErrRequestNotFound  = errors.New("friend request not found")
	ErrNotFriends       = errors.New("not friends")
)

// queryPattern is a username prefix: the characters usernames are made of.
var queryPattern = regexp.MustCompile(`^[a-z0-9_.]{2,20}$`)

// Status is how the viewer relates to another user.
type Status string

const (
	StatusNone    Status = "none"
	StatusFriends Status = "friends"
	// StatusOutgoing: the viewer sent a request that is still pending.
	StatusOutgoing Status = "outgoing"
	// StatusIncoming: the other user sent the viewer a request.
	StatusIncoming Status = "incoming"
)

// Person is what other users see of someone: never their email.
type Person struct {
	ID        uuid.UUID
	Username  string
	AvatarKey *string
}

type SearchResult struct {
	Person
	Status Status
}

type Friend struct {
	Person
	Since time.Time
}

type Request struct {
	Person
	CreatedAt time.Time
}

// Overview is everything the friends screen shows.
type Overview struct {
	Friends []Friend
	// Incoming are requests to the user, Outgoing the ones they sent; newest first.
	Incoming []Request
	Outgoing []Request
}

// Limit allows N events per Window.
type Limit struct {
	N      int
	Window time.Duration
}

// Limits bound how many friend requests one user can send, so a declined
// request cannot be repeated over and over.
type Limits struct {
	// PerSender caps the requests one user sends to anyone.
	PerSender Limit
	// PerPair caps the requests one user sends to the same person.
	PerPair Limit
}

var DefaultLimits = Limits{
	PerSender: Limit{N: 30, Window: time.Hour},
	PerPair:   Limit{N: 3, Window: 24 * time.Hour},
}

type Service struct {
	q         *store.Queries
	perSender *ratelimit.Limiter
	perPair   *ratelimit.Limiter
}

func NewService(db *pgxpool.Pool, limits Limits) *Service {
	return &Service{
		q:         store.New(db),
		perSender: ratelimit.New(limits.PerSender.N, limits.PerSender.Window),
		perPair:   ratelimit.New(limits.PerPair.N, limits.PerPair.Window),
	}
}

// NormalizeQuery turns what someone typed ("@Ada.L") into a username prefix.
func NormalizeQuery(raw string) (string, error) {
	q := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(raw), "@"))
	if !queryPattern.MatchString(q) {
		return "", ErrInvalidQuery
	}
	return q, nil
}

// Search finds users whose username starts with the query, except the viewer.
func (s *Service) Search(ctx context.Context, viewerID uuid.UUID, raw string) ([]SearchResult, error) {
	q, err := NormalizeQuery(raw)
	if err != nil {
		return nil, err
	}
	rows, err := s.q.SearchUsers(ctx, store.SearchUsersParams{
		ViewerID: viewerID,
		// "_" is a LIKE wildcard and allowed in usernames; "%" is not allowed.
		Pattern:    strings.ReplaceAll(q, "_", `\_`) + "%",
		Exact:      q,
		MaxResults: MaxSearchResults,
	})
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	results := make([]SearchResult, 0, len(rows))
	for _, r := range rows {
		results = append(results, SearchResult{
			Person: Person{ID: r.ID, Username: r.Username, AvatarKey: r.AvatarKey},
			Status: Status(r.Friendship),
		})
	}
	return results, nil
}

// Overview lists the user's friends and pending requests.
func (s *Service) Overview(ctx context.Context, userID uuid.UUID) (Overview, error) {
	friendRows, err := s.q.ListFriends(ctx, userID)
	if err != nil {
		return Overview{}, fmt.Errorf("list friends: %w", err)
	}
	requestRows, err := s.q.ListFriendRequests(ctx, userID)
	if err != nil {
		return Overview{}, fmt.Errorf("list friend requests: %w", err)
	}

	o := Overview{
		Friends:  make([]Friend, 0, len(friendRows)),
		Incoming: []Request{},
		Outgoing: []Request{},
	}
	for _, r := range friendRows {
		f := Friend{Person: Person{ID: r.ID, Username: r.Username, AvatarKey: r.AvatarKey}}
		if r.AcceptedAt != nil {
			f.Since = *r.AcceptedAt
		}
		o.Friends = append(o.Friends, f)
	}
	for _, r := range requestRows {
		req := Request{Person: Person{ID: r.ID, Username: r.Username, AvatarKey: r.AvatarKey}, CreatedAt: r.CreatedAt}
		if r.Incoming {
			o.Incoming = append(o.Incoming, req)
		} else {
			o.Outgoing = append(o.Outgoing, req)
		}
	}
	return o, nil
}

// SendRequest asks another user to be friends. If they already asked the
// sender, this accepts their request instead. The result is StatusOutgoing
// or StatusFriends; asking again is harmless.
func (s *Service) SendRequest(ctx context.Context, from store.User, toID uuid.UUID) (Status, error) {
	if from.Username == nil {
		return "", ErrUsernameRequired
	}
	if toID == from.ID {
		return "", ErrSelf
	}
	to, err := s.q.GetUser(ctx, toID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return "", ErrUserNotFound
	case err != nil:
		return "", fmt.Errorf("load user: %w", err)
	case to.Username == nil:
		return "", ErrUserNotFound
	}

	// The other user may send, cancel or accept at the same moment; each
	// step below is a single statement, so retry when one lost a race.
	for range 3 {
		accepted, err := s.q.AcceptFriendRequest(ctx, store.AcceptFriendRequestParams{RequesterID: toID, AddresseeID: from.ID})
		if err != nil {
			return "", fmt.Errorf("accept their request: %w", err)
		}
		if accepted == 1 {
			return StatusFriends, nil
		}

		existing, err := s.q.GetFriendship(ctx, store.GetFriendshipParams{UserA: from.ID, UserB: toID})
		switch {
		case errors.Is(err, pgx.ErrNoRows):
		case err != nil:
			return "", fmt.Errorf("load friendship: %w", err)
		case existing.Status == "accepted":
			return StatusFriends, nil
		case existing.RequesterID == from.ID:
			return StatusOutgoing, nil
		default:
			continue // their request arrived just now; accept it
		}

		if ok, wait := ratelimit.AllowAll(
			ratelimit.Check{Limiter: s.perSender, Key: from.ID.String()},
			ratelimit.Check{Limiter: s.perPair, Key: from.ID.String() + ">" + toID.String()},
		); !ok {
			return "", &ratelimit.Error{RetryAfter: wait}
		}
		created, err := s.q.CreateFriendRequest(ctx, store.CreateFriendRequestParams{RequesterID: from.ID, AddresseeID: toID})
		if err != nil {
			return "", fmt.Errorf("create friend request: %w", err)
		}
		if created == 1 {
			return StatusOutgoing, nil
		}
	}
	return "", errors.New("friendship kept changing while sending a request")
}

// AcceptRequest accepts the request that fromID sent to userID.
func (s *Service) AcceptRequest(ctx context.Context, userID, fromID uuid.UUID) error {
	n, err := s.q.AcceptFriendRequest(ctx, store.AcceptFriendRequestParams{RequesterID: fromID, AddresseeID: userID})
	if err != nil {
		return fmt.Errorf("accept friend request: %w", err)
	}
	if n == 0 {
		return ErrRequestNotFound
	}
	return nil
}

// DeleteRequest declines a request from otherID or cancels one sent to them.
func (s *Service) DeleteRequest(ctx context.Context, userID, otherID uuid.UUID) error {
	n, err := s.q.DeleteFriendship(ctx, store.DeleteFriendshipParams{UserA: userID, UserB: otherID, Status: "pending"})
	if err != nil {
		return fmt.Errorf("delete friend request: %w", err)
	}
	if n == 0 {
		return ErrRequestNotFound
	}
	return nil
}

// RemoveFriend ends a friendship; either side can.
func (s *Service) RemoveFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	n, err := s.q.DeleteFriendship(ctx, store.DeleteFriendshipParams{UserA: userID, UserB: friendID, Status: "accepted"})
	if err != nil {
		return fmt.Errorf("remove friend: %w", err)
	}
	if n == 0 {
		return ErrNotFriends
	}
	return nil
}
