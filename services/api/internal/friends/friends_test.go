package friends

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/ratelimit"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/store"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/testutil"
)

type fixture struct {
	svc *Service
	q   *store.Queries
}

func newFixture(t *testing.T, limits ...Limits) *fixture {
	t.Helper()
	l := DefaultLimits
	if len(limits) > 0 {
		l = limits[0]
	}
	pool := testutil.DB(t)
	return &fixture{svc: NewService(pool, l), q: store.New(pool)}
}

// user creates a user; an empty username leaves it unset.
func (f *fixture) user(t *testing.T, username string) store.User {
	t.Helper()
	ctx := context.Background()
	u, err := f.q.UpsertUserByEmail(ctx, uuid.NewString()+"@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if username == "" {
		return u
	}
	u, err = f.q.UpdateUsername(ctx, store.UpdateUsernameParams{ID: u.ID, Username: username})
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func (f *fixture) send(t *testing.T, from store.User, to store.User) Status {
	t.Helper()
	status, err := f.svc.SendRequest(context.Background(), from, to.ID)
	if err != nil {
		t.Fatalf("send %s → %s: %v", *from.Username, *to.Username, err)
	}
	return status
}

func (f *fixture) overview(t *testing.T, u store.User) Overview {
	t.Helper()
	o, err := f.svc.Overview(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func usernames[T interface{ person() Person }](items []T) []string {
	out := []string{}
	for _, it := range items {
		out = append(out, it.person().Username)
	}
	return out
}

func (r SearchResult) person() Person { return r.Person }
func (f Friend) person() Person       { return f.Person }
func (r Request) person() Person      { return r.Person }

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestNormalizeQuery(t *testing.T) {
	valid := map[string]string{"@Ada": "ada", " ad ": "ad", "a_b.c": "a_b.c"}
	for in, want := range valid {
		if got, err := NormalizeQuery(in); err != nil || got != want {
			t.Errorf("NormalizeQuery(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "a", "@", "a%", "ad min", "abcdefghijklmnopqrstu", "äd"} {
		if _, err := NormalizeQuery(in); !errors.Is(err, ErrInvalidQuery) {
			t.Errorf("NormalizeQuery(%q) should fail", in)
		}
	}
}

func TestSearch(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	me := f.user(t, "me.myself")
	f.user(t, "adam")
	f.user(t, "ada")
	f.user(t, "ada.lovelace")
	f.user(t, "bob")
	f.user(t, "") // no username yet: never found

	res, err := f.svc.Search(ctx, me.ID, "@ADA")
	if err != nil {
		t.Fatal(err)
	}
	// Exact match first, then shorter names.
	if got := usernames(res); !equal(got, []string{"ada", "adam", "ada.lovelace"}) {
		t.Fatalf("search ada = %v", got)
	}

	// "_" matches only itself, not any character.
	f.user(t, "a_x")
	f.user(t, "abx")
	res, _ = f.svc.Search(ctx, me.ID, "a_")
	if got := usernames(res); !equal(got, []string{"a_x"}) {
		t.Fatalf("search a_ = %v", got)
	}

	// The viewer is not in their own results.
	res, _ = f.svc.Search(ctx, me.ID, "me")
	if len(res) != 0 {
		t.Fatalf("search me = %v, want nothing", usernames(res))
	}

	if _, err := f.svc.Search(ctx, me.ID, "a"); !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("one-letter search: err = %v", err)
	}
}

func TestSearchShowsFriendship(t *testing.T) {
	f := newFixture(t)
	me := f.user(t, "myself")
	friend := f.user(t, "pal.friend")
	asked := f.user(t, "pal.asked")
	asking := f.user(t, "pal.asking")
	f.user(t, "pal.stranger")

	f.send(t, me, friend)
	f.send(t, friend, me)
	f.send(t, me, asked)
	f.send(t, asking, me)

	res, err := f.svc.Search(context.Background(), me.ID, "pal")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]Status{
		"pal.friend":   StatusFriends,
		"pal.asked":    StatusOutgoing,
		"pal.asking":   StatusIncoming,
		"pal.stranger": StatusNone,
	}
	if len(res) != len(want) {
		t.Fatalf("got %v", usernames(res))
	}
	for _, r := range res {
		if r.Status != want[r.Username] {
			t.Errorf("%s: status %q, want %q", r.Username, r.Status, want[r.Username])
		}
	}
}

func TestRequestAcceptAndRemove(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	ada := f.user(t, "ada")
	bob := f.user(t, "bob")

	if s := f.send(t, ada, bob); s != StatusOutgoing {
		t.Fatalf("send = %q", s)
	}
	// Asking again changes nothing.
	if s := f.send(t, ada, bob); s != StatusOutgoing {
		t.Fatalf("send again = %q", s)
	}
	if o := f.overview(t, ada); len(o.Outgoing) != 1 || o.Outgoing[0].Username != "bob" || len(o.Incoming) != 0 {
		t.Fatalf("ada's overview = %+v", o)
	}
	if o := f.overview(t, bob); len(o.Incoming) != 1 || o.Incoming[0].Username != "ada" || len(o.Outgoing) != 0 {
		t.Fatalf("bob's overview = %+v", o)
	}

	// Only the addressee can accept.
	if err := f.svc.AcceptRequest(ctx, ada.ID, bob.ID); !errors.Is(err, ErrRequestNotFound) {
		t.Fatalf("sender accepting their own request: err = %v", err)
	}
	if err := f.svc.AcceptRequest(ctx, bob.ID, ada.ID); err != nil {
		t.Fatal(err)
	}
	for _, u := range []store.User{ada, bob} {
		o := f.overview(t, u)
		if len(o.Friends) != 1 || len(o.Incoming) != 0 || len(o.Outgoing) != 0 || o.Friends[0].Since.IsZero() {
			t.Fatalf("%s's overview after accepting = %+v", *u.Username, o)
		}
	}
	if s := f.send(t, ada, bob); s != StatusFriends {
		t.Fatalf("send to a friend = %q", s)
	}
	if err := f.svc.DeleteRequest(ctx, ada.ID, bob.ID); !errors.Is(err, ErrRequestNotFound) {
		t.Fatalf("declining a friendship: err = %v", err)
	}

	// Either side can end it.
	if err := f.svc.RemoveFriend(ctx, bob.ID, ada.ID); err != nil {
		t.Fatal(err)
	}
	if o := f.overview(t, ada); len(o.Friends) != 0 {
		t.Fatalf("ada still has friends: %+v", o)
	}
	if err := f.svc.RemoveFriend(ctx, bob.ID, ada.ID); !errors.Is(err, ErrNotFriends) {
		t.Fatalf("removing twice: err = %v", err)
	}
}

func TestDeclineAndCancel(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	ada := f.user(t, "ada")
	bob := f.user(t, "bob")

	f.send(t, ada, bob)
	// A pending request is not a friendship.
	if err := f.svc.RemoveFriend(ctx, bob.ID, ada.ID); !errors.Is(err, ErrNotFriends) {
		t.Fatalf("removing a pending request as a friend: err = %v", err)
	}
	if err := f.svc.DeleteRequest(ctx, bob.ID, ada.ID); err != nil {
		t.Fatalf("decline: %v", err)
	}
	if o := f.overview(t, ada); len(o.Outgoing) != 0 {
		t.Fatalf("declined request still listed: %+v", o)
	}

	f.send(t, ada, bob)
	if err := f.svc.DeleteRequest(ctx, ada.ID, bob.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if o := f.overview(t, bob); len(o.Incoming) != 0 {
		t.Fatalf("cancelled request still listed: %+v", o)
	}
	if err := f.svc.DeleteRequest(ctx, ada.ID, bob.ID); !errors.Is(err, ErrRequestNotFound) {
		t.Fatalf("cancel twice: err = %v", err)
	}
}

func TestRequestingBackAccepts(t *testing.T) {
	f := newFixture(t)
	ada := f.user(t, "ada")
	bob := f.user(t, "bob")
	f.send(t, ada, bob)
	if s := f.send(t, bob, ada); s != StatusFriends {
		t.Fatalf("requesting back = %q, want friends", s)
	}
	if o := f.overview(t, ada); len(o.Friends) != 1 {
		t.Fatalf("ada's overview = %+v", o)
	}
}

// Two people adding each other at the same moment become friends.
func TestSimultaneousRequests(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	for i := range 20 {
		ada := f.user(t, "ada"+uuid.NewString()[:8])
		bob := f.user(t, "bob"+uuid.NewString()[:8])
		var wg sync.WaitGroup
		errs := make([]error, 2)
		wg.Go(func() { _, errs[0] = f.svc.SendRequest(ctx, ada, bob.ID) })
		wg.Go(func() { _, errs[1] = f.svc.SendRequest(ctx, bob, ada.ID) })
		wg.Wait()
		if errs[0] != nil || errs[1] != nil {
			t.Fatalf("round %d: %v, %v", i, errs[0], errs[1])
		}
		if o := f.overview(t, ada); len(o.Friends) != 1 {
			t.Fatalf("round %d: ada's overview = %+v, want one friend", i, o)
		}
	}
}

func TestSendErrors(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	ada := f.user(t, "ada")
	nameless := f.user(t, "")

	if _, err := f.svc.SendRequest(ctx, ada, ada.ID); !errors.Is(err, ErrSelf) {
		t.Fatalf("adding yourself: err = %v", err)
	}
	if _, err := f.svc.SendRequest(ctx, ada, uuid.New()); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("unknown user: err = %v", err)
	}
	if _, err := f.svc.SendRequest(ctx, ada, nameless.ID); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("user without a username: err = %v", err)
	}
	if _, err := f.svc.SendRequest(ctx, nameless, ada.ID); !errors.Is(err, ErrUsernameRequired) {
		t.Fatalf("sender without a username: err = %v", err)
	}
}

func TestSearchRateLimit(t *testing.T) {
	limits := DefaultLimits
	limits.Search = Limit{N: 2, Window: time.Hour}
	f := newFixture(t, limits)
	ctx := context.Background()
	ada := f.user(t, "ada")
	bob := f.user(t, "bob")

	// Invalid queries never reach the database and use up nothing.
	if _, err := f.svc.Search(ctx, ada.ID, "a"); !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("one-letter search: err = %v", err)
	}
	for range 2 {
		if _, err := f.svc.Search(ctx, ada.ID, "bo"); err != nil {
			t.Fatal(err)
		}
	}
	_, err := f.svc.Search(ctx, ada.ID, "bo")
	var rl *ratelimit.Error
	if !errors.As(err, &rl) || rl.RetryAfter <= 0 {
		t.Fatalf("third search: err = %v, want a rate limit", err)
	}

	// The limit is per user.
	if _, err := f.svc.Search(ctx, bob.ID, "ad"); err != nil {
		t.Fatalf("another user's search: %v", err)
	}
}

func TestSendRateLimits(t *testing.T) {
	f := newFixture(t, Limits{
		PerSender: Limit{N: 3, Window: time.Hour},
		PerPair:   Limit{N: 2, Window: time.Hour},
		Search:    DefaultLimits.Search,
	})
	ctx := context.Background()
	ada := f.user(t, "ada")
	bob := f.user(t, "bob")

	// Declining and asking again counts against the pair.
	for range 2 {
		f.send(t, ada, bob)
		if err := f.svc.DeleteRequest(ctx, bob.ID, ada.ID); err != nil {
			t.Fatal(err)
		}
	}
	_, err := f.svc.SendRequest(ctx, ada, bob.ID)
	var rl *ratelimit.Error
	if !errors.As(err, &rl) || rl.RetryAfter <= 0 {
		t.Fatalf("third request to bob: err = %v, want a rate limit", err)
	}

	// The sender limit counts requests to anyone: 2 so far, 1 left.
	f.send(t, ada, f.user(t, "cyd"))
	if _, err := f.svc.SendRequest(ctx, ada, f.user(t, "dee").ID); !errors.As(err, &rl) {
		t.Fatalf("request over the sender limit: err = %v", err)
	}

	// Accepting someone's request by asking back is never limited.
	eve := f.user(t, "eve")
	f.send(t, eve, ada)
	if s := f.send(t, ada, eve); s != StatusFriends {
		t.Fatalf("asking back while limited = %q", s)
	}
}
