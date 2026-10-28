package auth

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestAPITokenLifecycle(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	user := f.signIn(t, "ada@example.com").User

	res, err := f.svc.CreateAPIToken(ctx, user.ID, "  Claude Code  ", 30)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(res.Token, APITokenPrefix) || len(res.Token) != len(APITokenPrefix)+43 {
		t.Fatalf("token %q: want fgc_ and 43 characters", res.Token)
	}
	if res.APIToken.Name != "Claude Code" || !res.APIToken.ExpiresAt.Equal(f.now.AddDate(0, 0, 30)) || res.APIToken.LastUsedAt != nil {
		t.Fatalf("unexpected token %+v", res.APIToken)
	}

	p, err := f.svc.AuthenticateToken(ctx, res.Token)
	if err != nil {
		t.Fatal(err)
	}
	if p.User.ID != user.ID || p.TokenID != res.APIToken.ID {
		t.Fatalf("principal %+v: want the user and the token's ID", p)
	}
	tokens, err := f.svc.ListAPITokens(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || tokens[0].LastUsedAt == nil || !tokens[0].LastUsedAt.Equal(f.now) {
		t.Fatalf("tokens = %+v, want one used just now", tokens)
	}

	// last_used_at moves at most once a minute.
	first := f.now
	f.now = f.now.Add(30 * time.Second)
	if _, err := f.svc.AuthenticateToken(ctx, res.Token); err != nil {
		t.Fatal(err)
	}
	tokens, _ = f.svc.ListAPITokens(ctx, user.ID)
	if !tokens[0].LastUsedAt.Equal(first) {
		t.Fatalf("last used %v, want it unchanged at %v", tokens[0].LastUsedAt, first)
	}
	f.now = f.now.Add(time.Minute)
	if _, err := f.svc.AuthenticateToken(ctx, res.Token); err != nil {
		t.Fatal(err)
	}
	tokens, _ = f.svc.ListAPITokens(ctx, user.ID)
	if !tokens[0].LastUsedAt.Equal(f.now) {
		t.Fatalf("last used %v, want %v", tokens[0].LastUsedAt, f.now)
	}

	ok, err := f.svc.RevokeAPIToken(ctx, user.ID, res.APIToken.ID)
	if err != nil || !ok {
		t.Fatalf("revoke: %v %v", ok, err)
	}
	if _, err := f.svc.AuthenticateToken(ctx, res.Token); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("revoked token: err = %v, want ErrUnauthenticated", err)
	}
	if ok, _ := f.svc.RevokeAPIToken(ctx, user.ID, res.APIToken.ID); ok {
		t.Fatal("revoking twice should report false")
	}
	if tokens, _ := f.svc.ListAPITokens(ctx, user.ID); len(tokens) != 0 {
		t.Fatalf("revoked tokens are not listed, got %+v", tokens)
	}
}

func TestAPITokenExpires(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	user := f.signIn(t, "ada@example.com").User
	res, err := f.svc.CreateAPIToken(ctx, user.ID, "laptop", 30)
	if err != nil {
		t.Fatal(err)
	}
	f.now = f.now.AddDate(0, 0, 30)
	if _, err := f.svc.AuthenticateToken(ctx, res.Token); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expired token: err = %v, want ErrUnauthenticated", err)
	}
	if tokens, _ := f.svc.ListAPITokens(ctx, user.ID); len(tokens) != 0 {
		t.Fatalf("expired tokens are not listed, got %+v", tokens)
	}
	if ok, _ := f.svc.RevokeAPIToken(ctx, user.ID, res.APIToken.ID); ok {
		t.Fatal("an expired token cannot be revoked")
	}
}

func TestAPITokenValidationAndLimits(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	ada := f.signIn(t, "ada@example.com").User
	bob := f.signIn(t, "bob@example.com").User

	for _, name := range []string{"", "   ", strings.Repeat("x", 41), "tab\there"} {
		if _, err := f.svc.CreateAPIToken(ctx, ada.ID, name, 90); !errors.Is(err, ErrInvalidTokenName) {
			t.Errorf("name %q: err = %v, want ErrInvalidTokenName", name, err)
		}
	}
	if _, err := f.svc.CreateAPIToken(ctx, ada.ID, strings.Repeat("ü", 40), 90); err != nil {
		t.Errorf("40 characters are allowed: %v", err)
	}
	for _, days := range []int{0, 7, 89, 366} {
		if _, err := f.svc.CreateAPIToken(ctx, ada.ID, "x", days); !errors.Is(err, ErrInvalidTokenExpiry) {
			t.Errorf("%d days: err = %v, want ErrInvalidTokenExpiry", days, err)
		}
	}

	// Ada has one token; nine more reach the limit.
	var last CreateAPITokenResult
	for i := range MaxAPITokens - 1 {
		res, err := f.svc.CreateAPIToken(ctx, ada.ID, "token "+strconv.Itoa(i), 365)
		if err != nil {
			t.Fatalf("token %d: %v", i, err)
		}
		last = res
	}
	if _, err := f.svc.CreateAPIToken(ctx, ada.ID, "one too many", 90); !errors.Is(err, ErrTooManyTokens) {
		t.Fatalf("11th token: err = %v, want ErrTooManyTokens", err)
	}
	// Someone else's token cannot be revoked; revoking one's own makes room.
	if ok, _ := f.svc.RevokeAPIToken(ctx, bob.ID, last.APIToken.ID); ok {
		t.Fatal("bob revoked ada's token")
	}
	if _, err := f.svc.AuthenticateToken(ctx, last.Token); err != nil {
		t.Fatalf("ada's token still works: %v", err)
	}
	if ok, _ := f.svc.RevokeAPIToken(ctx, ada.ID, last.APIToken.ID); !ok {
		t.Fatal("ada could not revoke her token")
	}
	if _, err := f.svc.CreateAPIToken(ctx, ada.ID, "after revoking", 90); err != nil {
		t.Fatalf("a revoked token frees its place: %v", err)
	}
}

func TestAPITokenCreateRateLimited(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	user := f.signIn(t, "ada@example.com").User
	for i := range DefaultLimits.TokenCreates.N {
		res, err := f.svc.CreateAPIToken(ctx, user.ID, "t", 90)
		if err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
		// Revoke right away, so the token limit does not get in the way.
		if _, err := f.svc.RevokeAPIToken(ctx, user.ID, res.APIToken.ID); err != nil {
			t.Fatal(err)
		}
	}
	var rl *RateLimitError
	if _, err := f.svc.CreateAPIToken(ctx, user.ID, "t", 90); !errors.As(err, &rl) {
		t.Fatalf("err = %v, want a rate limit", err)
	}
}

func TestAuthenticateTokenRejectsOtherTokens(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	session := f.signIn(t, "ada@example.com")
	for _, token := range []string{
		"",
		session.Token,
		APITokenPrefix + session.Token[:42],
		APITokenPrefix + session.Token, // the right shape, but no such token
	} {
		if _, err := f.svc.AuthenticateToken(ctx, token); !errors.Is(err, ErrUnauthenticated) {
			t.Errorf("token %q: err = %v, want ErrUnauthenticated", token, err)
		}
	}
}

func TestCleanupRemovesStaleAPITokens(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	user := f.signIn(t, "ada@example.com").User
	if _, err := f.svc.CreateAPIToken(ctx, user.ID, "expiring", 30); err != nil {
		t.Fatal(err)
	}
	revoked, err := f.svc.CreateAPIToken(ctx, user.ID, "revoked", 365)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.RevokeAPIToken(ctx, user.ID, revoked.APIToken.ID); err != nil {
		t.Fatal(err)
	}
	kept, err := f.svc.CreateAPIToken(ctx, user.ID, "kept", 365)
	if err != nil {
		t.Fatal(err)
	}

	count := func() int {
		var n int
		if err := f.svc.db.QueryRow(ctx, "SELECT count(*) FROM api_tokens").Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	// 30 days after expiring and revoking, both are still kept.
	f.now = f.now.AddDate(0, 0, 30)
	if err := f.svc.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	if n := count(); n != 3 {
		t.Fatalf("%d tokens after the first cleanup, want 3", n)
	}
	// Past expiry plus 30 days, both go.
	f.now = f.now.AddDate(0, 0, 31)
	if err := f.svc.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	if n := count(); n != 1 {
		t.Fatalf("%d tokens after the second cleanup, want 1", n)
	}
	if _, err := f.svc.AuthenticateToken(ctx, kept.Token); err != nil {
		t.Fatalf("the live token was removed: %v", err)
	}
}
