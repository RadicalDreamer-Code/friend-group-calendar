package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/ratelimit"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/testutil"
)

type fixture struct {
	svc  *Service
	mail *testutil.Mailbox
	now  time.Time
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{mail: &testutil.Mailbox{}, now: time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)}
	// Most tests request several codes in a row; the cooldown has its own test.
	limits := DefaultLimits
	limits.EmailCooldown = Limit{1000, time.Second}
	f.svc = NewService(testutil.DB(t), f.mail, []byte(strings.Repeat("k", 32)), limits)
	f.svc.now = func() time.Time { return f.now }
	return f
}

func (f *fixture) signIn(t *testing.T, email string) VerifyResult {
	t.Helper()
	ctx := context.Background()
	if err := f.svc.RequestCode(ctx, email, "203.0.113.1"); err != nil {
		t.Fatal(err)
	}
	// Mail goes to the normalized address.
	res, err := f.svc.Verify(ctx, VerifyInput{Email: email, Code: f.mail.LastCode(t, strings.ToLower(email)), Platform: PlatformWeb})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestVerifyCreatesUserOnFirstSignInOnly(t *testing.T) {
	f := newFixture(t)
	first := f.signIn(t, "ada@example.com")
	second := f.signIn(t, "ADA@example.com")
	if first.User.ID != second.User.ID {
		t.Fatal("signing in again with the same email (any case) must not create a second user")
	}
	if first.User.Username != nil {
		t.Fatal("new users start without a username")
	}
	if first.Token == second.Token {
		t.Fatal("each sign-in gets its own session")
	}
}

func TestCodeIsSingleUse(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	if err := f.svc.RequestCode(ctx, "ada@example.com", "ip"); err != nil {
		t.Fatal(err)
	}
	code := f.mail.LastCode(t, "ada@example.com")
	in := VerifyInput{Email: "ada@example.com", Code: code, Platform: PlatformIOS}
	if _, err := f.svc.Verify(ctx, in); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Verify(ctx, in); !errors.Is(err, ErrCodeExpired) {
		t.Fatalf("reusing a code: err = %v, want ErrCodeExpired", err)
	}
}

func TestNewCodeReplacesOldOne(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	_ = f.svc.RequestCode(ctx, "ada@example.com", "ip")
	old := f.mail.LastCode(t, "ada@example.com")
	_ = f.svc.RequestCode(ctx, "ada@example.com", "ip")
	fresh := f.mail.LastCode(t, "ada@example.com")
	if old == fresh {
		t.Skip("both random codes were equal; nothing to compare")
	}

	_, err := f.svc.Verify(ctx, VerifyInput{Email: "ada@example.com", Code: old, Platform: PlatformWeb})
	if !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("old code: err = %v, want ErrInvalidCode", err)
	}
	if _, err := f.svc.Verify(ctx, VerifyInput{Email: "ada@example.com", Code: fresh, Platform: PlatformWeb}); err != nil {
		t.Fatalf("new code: %v", err)
	}
}

func TestCodeExpires(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	_ = f.svc.RequestCode(ctx, "ada@example.com", "ip")
	f.now = f.now.Add(CodeTTL + time.Second)
	_, err := f.svc.Verify(ctx, VerifyInput{Email: "ada@example.com", Code: f.mail.LastCode(t, "ada@example.com"), Platform: PlatformWeb})
	if !errors.Is(err, ErrCodeExpired) {
		t.Fatalf("err = %v, want ErrCodeExpired", err)
	}
}

func TestWrongAttemptsUseUpTheCode(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	_ = f.svc.RequestCode(ctx, "ada@example.com", "ip")
	code := f.mail.LastCode(t, "ada@example.com")
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	for i := range MaxCodeAttempts {
		_, err := f.svc.Verify(ctx, VerifyInput{Email: "ada@example.com", Code: wrong, Platform: PlatformWeb})
		if !errors.Is(err, ErrInvalidCode) {
			t.Fatalf("attempt %d: err = %v, want ErrInvalidCode", i+1, err)
		}
	}
	_, err := f.svc.Verify(ctx, VerifyInput{Email: "ada@example.com", Code: code, Platform: PlatformWeb})
	if !errors.Is(err, ErrCodeExpired) {
		t.Fatalf("right code after %d wrong ones: err = %v, want ErrCodeExpired", MaxCodeAttempts, err)
	}
}

func TestRequestCodeCooldown(t *testing.T) {
	f := newFixture(t)
	f.svc.emailCooldown = ratelimit.New(1, 30*time.Second)
	ctx := context.Background()
	if err := f.svc.RequestCode(ctx, "ada@example.com", "ip"); err != nil {
		t.Fatal(err)
	}
	var rl *RateLimitError
	if err := f.svc.RequestCode(ctx, "ada@example.com", "ip"); !errors.As(err, &rl) {
		t.Fatalf("second request: err = %v, want RateLimitError", err)
	}
	if len(f.mail.Sent()) != 1 {
		t.Fatal("a rate-limited request must not send an email")
	}
}

func TestSessionsRollForwardAndExpire(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	res := f.signIn(t, "ada@example.com")

	f.now = f.now.Add(30 * time.Minute)
	p, renewed, err := f.svc.Authenticate(ctx, res.Token)
	if err != nil || renewed {
		t.Fatalf("within the renew interval: renewed=%v err=%v, want no renewal", renewed, err)
	}

	f.now = f.now.Add(2 * time.Hour)
	p, renewed, err = f.svc.Authenticate(ctx, res.Token)
	if err != nil || !renewed {
		t.Fatalf("after the renew interval: renewed=%v err=%v, want renewal", renewed, err)
	}
	if want := f.now.Add(SessionTTL); !p.Session.ExpiresAt.Equal(want) {
		t.Fatalf("expires at %v, want %v", p.Session.ExpiresAt, want)
	}

	f.now = f.now.Add(SessionTTL + time.Minute)
	if _, _, err := f.svc.Authenticate(ctx, res.Token); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("after %v unused: err = %v, want ErrUnauthenticated", SessionTTL, err)
	}
}

func TestRevokeOtherSessions(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	phone := f.signIn(t, "ada@example.com")
	laptop := f.signIn(t, "ada@example.com")

	p, _, err := f.svc.Authenticate(ctx, laptop.Token)
	if err != nil {
		t.Fatal(err)
	}
	if n, err := f.svc.RevokeOtherSessions(ctx, p.User.ID, p.Session.ID); err != nil || n != 1 {
		t.Fatalf("revoked %d sessions (err %v), want 1", n, err)
	}
	if _, _, err := f.svc.Authenticate(ctx, phone.Token); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("revoked session: err = %v, want ErrUnauthenticated", err)
	}
	if _, _, err := f.svc.Authenticate(ctx, laptop.Token); err != nil {
		t.Fatalf("current session must survive: %v", err)
	}
}

func TestAuthenticateRejectsGarbage(t *testing.T) {
	f := newFixture(t)
	for _, token := range []string{"", "nope", strings.Repeat("a", 43)} {
		if _, _, err := f.svc.Authenticate(context.Background(), token); !errors.Is(err, ErrUnauthenticated) {
			t.Errorf("token %q: err = %v, want ErrUnauthenticated", token, err)
		}
	}
}
