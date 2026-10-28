package auth

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

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

// request asks for a code and returns its challenge and the emailed code.
func (f *fixture) request(t *testing.T, email, client string) (uuid.UUID, string) {
	t.Helper()
	id, err := f.svc.RequestCode(context.Background(), email, client)
	if err != nil {
		t.Fatal(err)
	}
	// Mail goes to the normalized address.
	return id, f.mail.LastCode(t, strings.ToLower(email))
}

func (f *fixture) signIn(t *testing.T, email string) VerifyResult {
	t.Helper()
	id, code := f.request(t, email, "203.0.113.1")
	res, err := f.svc.Verify(context.Background(), VerifyInput{ChallengeID: id, Email: email, Code: code, Platform: PlatformWeb})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// otherCode returns a valid-looking code that differs from code.
func otherCode(code string) string {
	if code == "000000" {
		return "111111"
	}
	return "000000"
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
	id, code := f.request(t, "ada@example.com", "ip")
	in := VerifyInput{ChallengeID: id, Email: "ada@example.com", Code: code, Platform: PlatformIOS}
	if _, err := f.svc.Verify(ctx, in); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Verify(ctx, in); !errors.Is(err, ErrCodeExpired) {
		t.Fatalf("reusing a code: err = %v, want ErrCodeExpired", err)
	}
}

func TestCodeOnlyWorksWithItsChallenge(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	firstID, first := f.request(t, "ada@example.com", "ip")
	secondID, second := f.request(t, "ada@example.com", "ip")
	if first == second {
		t.Skip("both random codes were equal; nothing to compare")
	}

	_, err := f.svc.Verify(ctx, VerifyInput{ChallengeID: secondID, Email: "ada@example.com", Code: first, Platform: PlatformWeb})
	if !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("first code on the second challenge: err = %v, want ErrInvalidCode", err)
	}
	_, err = f.svc.Verify(ctx, VerifyInput{ChallengeID: firstID, Email: "bob@example.com", Code: first, Platform: PlatformWeb})
	if !errors.Is(err, ErrCodeExpired) {
		t.Fatalf("challenge with another address: err = %v, want ErrCodeExpired", err)
	}
	// Requesting a second code does not invalidate the first.
	if _, err := f.svc.Verify(ctx, VerifyInput{ChallengeID: firstID, Email: "ada@example.com", Code: first, Platform: PlatformWeb}); err != nil {
		t.Fatalf("first code on its own challenge: %v", err)
	}
}

func TestOthersCannotLockTheOwnerOut(t *testing.T) {
	f := newFixture(t)
	f.svc.emailCooldown = ratelimit.New(1, 30*time.Second)
	ctx := context.Background()
	ownerID, code := f.request(t, "ada@example.com", "owner")

	// Someone else asks for codes to the same address and guesses wrong
	// until their own challenge is used up.
	attackerID, attackerCode := f.request(t, "ada@example.com", "attacker")
	for range MaxCodeAttempts + 1 {
		_, _ = f.svc.Verify(ctx, VerifyInput{ChallengeID: attackerID, Email: "ada@example.com", Code: otherCode(attackerCode), Platform: PlatformWeb})
		_, _ = f.svc.Verify(ctx, VerifyInput{ChallengeID: uuid.New(), Email: "ada@example.com", Code: code, Platform: PlatformWeb})
	}

	if _, err := f.svc.Verify(ctx, VerifyInput{ChallengeID: ownerID, Email: "ada@example.com", Code: code, Platform: PlatformWeb}); err != nil {
		t.Fatalf("owner's code after someone else's attempts: %v", err)
	}
}

func TestCodeExpires(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	id, code := f.request(t, "ada@example.com", "ip")
	f.now = f.now.Add(CodeTTL + time.Second)
	_, err := f.svc.Verify(ctx, VerifyInput{ChallengeID: id, Email: "ada@example.com", Code: code, Platform: PlatformWeb})
	if !errors.Is(err, ErrCodeExpired) {
		t.Fatalf("err = %v, want ErrCodeExpired", err)
	}
}

func TestWrongAttemptsUseUpTheCode(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	id, code := f.request(t, "ada@example.com", "ip")
	for i := range MaxCodeAttempts {
		_, err := f.svc.Verify(ctx, VerifyInput{ChallengeID: id, Email: "ada@example.com", Code: otherCode(code), Platform: PlatformWeb})
		if !errors.Is(err, ErrInvalidCode) {
			t.Fatalf("attempt %d: err = %v, want ErrInvalidCode", i+1, err)
		}
	}
	_, err := f.svc.Verify(ctx, VerifyInput{ChallengeID: id, Email: "ada@example.com", Code: code, Platform: PlatformWeb})
	if !errors.Is(err, ErrCodeExpired) {
		t.Fatalf("right code after %d wrong ones: err = %v, want ErrCodeExpired", MaxCodeAttempts, err)
	}
}

func TestRequestCodeCooldown(t *testing.T) {
	f := newFixture(t)
	f.svc.emailCooldown = ratelimit.New(1, 30*time.Second)
	ctx := context.Background()
	f.request(t, "ada@example.com", "ip")
	var rl *RateLimitError
	if _, err := f.svc.RequestCode(ctx, "ada@example.com", "ip"); !errors.As(err, &rl) {
		t.Fatalf("second request: err = %v, want RateLimitError", err)
	}
	if len(f.mail.Sent()) != 1 {
		t.Fatal("a rate-limited request must not send an email")
	}
	// The cooldown is per client: another one is not held up by it.
	f.request(t, "ada@example.com", "other-ip")
}

func TestEmailHourlyLimitSpansClients(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	perClient, total := DefaultLimits.ClientEmailHourly.N, DefaultLimits.EmailHourly.N
	var rl *RateLimitError
	for range perClient {
		f.request(t, "ada@example.com", "a")
	}
	if _, err := f.svc.RequestCode(ctx, "ada@example.com", "a"); !errors.As(err, &rl) {
		t.Fatalf("request %d from one client: err = %v, want RateLimitError", perClient+1, err)
	}
	for i := perClient; i < total; i++ {
		f.request(t, "ada@example.com", "client-"+strconv.Itoa(i))
	}
	if _, err := f.svc.RequestCode(ctx, "ada@example.com", "yet-another"); !errors.As(err, &rl) {
		t.Fatalf("request %d to one address: err = %v, want RateLimitError", total+1, err)
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
	token, err := f.svc.CreateAPIToken(ctx, p.User.ID, "Claude Code", 365)
	if err != nil {
		t.Fatal(err)
	}
	if n, err := f.svc.RevokeOtherSessions(ctx, p.User.ID, p.Session.ID); err != nil || n != 1 {
		t.Fatalf("revoked %d sessions (err %v), want 1", n, err)
	}
	// A token made from a stolen session must not outlive it.
	if _, err := f.svc.AuthenticateToken(ctx, token.Token); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("access token after signing out others: err = %v, want ErrUnauthenticated", err)
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

func TestRejectedRequestsUseUpNoBudget(t *testing.T) {
	f := newFixture(t)
	f.svc.emailCooldown = ratelimit.New(1, 30*time.Second)
	f.svc.ipCodes = ratelimit.New(2, time.Hour)
	ctx := context.Background()
	f.request(t, "ada@example.com", "ip")
	var rl *RateLimitError
	if _, err := f.svc.RequestCode(ctx, "ada@example.com", "ip"); !errors.As(err, &rl) {
		t.Fatalf("resend during the cooldown: err = %v, want RateLimitError", err)
	}
	// The rejected resend did not take the client's second code.
	f.request(t, "bob@example.com", "ip")
}
