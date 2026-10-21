package httpapi

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/auth"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/calendars"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/friends"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/testutil"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/users"
)

const webOrigin = "http://localhost:8082"

type testAPI struct {
	url   string
	mail  *testutil.Mailbox
	blobs *testutil.Blobs
}

// newTestAPI runs the API with the default limits except the per-address
// cooldown, so tests can sign the same user in on several devices. configure
// can adjust the HTTP server before it starts.
func newTestAPI(t *testing.T, configure ...func(*http.Server)) *testAPI {
	t.Helper()
	limits := auth.DefaultLimits
	limits.EmailCooldown = auth.Limit{N: 1000, Window: time.Second}
	pool := testutil.DB(t)
	a := &testAPI{mail: &testutil.Mailbox{}, blobs: testutil.NewBlobs()}
	h, err := NewRouter(Options{
		DB:             pool,
		AllowedOrigins: []string{webOrigin},
		Auth:           auth.NewService(pool, a.mail, []byte(strings.Repeat("k", 32)), limits),
		Users:          users.NewService(pool, a.blobs),
		Friends:        friends.NewService(pool, friends.DefaultLimits),
		Calendars:      calendars.NewService(pool),
		Avatars:        a.blobs,
		PublicURL:      "http://api.test",
		// Go's cookie jar only sends Secure cookies over https.
		CookieSecure: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewUnstartedServer(h)
	for _, f := range configure {
		f(srv.Config)
	}
	srv.Start()
	t.Cleanup(srv.Close)
	a.url = srv.URL
	return a
}

// client is one device: a cookie jar for web, or a bearer token for native.
type client struct {
	t     *testing.T
	api   *testAPI
	http  *http.Client
	token string
}

func (a *testAPI) newClient(t *testing.T) *client {
	jar, _ := cookiejar.New(nil)
	return &client{t: t, api: a, http: &http.Client{Jar: jar}}
}

func (c *client) do(method, path string, body any, headers ...string) (*http.Response, []byte) {
	c.t.Helper()
	var r io.Reader
	contentType := ""
	switch b := body.(type) {
	case nil:
	default:
		buf, err := json.Marshal(b)
		if err != nil {
			c.t.Fatal(err)
		}
		r, contentType = bytes.NewReader(buf), "application/json"
	}
	req, err := http.NewRequest(method, c.api.url+path, r)
	if err != nil {
		c.t.Fatal(err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return c.send(req, headers...)
}

func (c *client) send(req *http.Request, headers ...string) (*http.Response, []byte) {
	c.t.Helper()
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	resp, err := c.http.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		c.t.Fatal(err)
	}
	return resp, data
}

// requestCode asks for a code and returns its challenge ID.
func (c *client) requestCode(email string) string {
	c.t.Helper()
	resp, body := c.do("POST", "/auth/request-code", map[string]string{"email": email})
	if resp.StatusCode != http.StatusOK {
		c.t.Fatalf("request-code: %d %s", resp.StatusCode, body)
	}
	var out struct {
		ChallengeID string `json:"challengeId"`
	}
	decode(c.t, body, &out)
	if out.ChallengeID == "" {
		c.t.Fatalf("request-code returned no challengeId: %s", body)
	}
	return out.ChallengeID
}

func (c *client) signIn(email, platform string) verifyResponseJSON {
	c.t.Helper()
	challengeID := c.requestCode(email)
	resp, body := c.do("POST", "/auth/verify", map[string]string{
		"challengeId": challengeID, "email": email, "code": c.api.mail.LastCode(c.t, email), "platform": platform,
	})
	if resp.StatusCode != http.StatusOK {
		c.t.Fatalf("verify: %d %s", resp.StatusCode, body)
	}
	var out verifyResponseJSON
	decode(c.t, body, &out)
	if platform != "web" {
		c.token = out.Token
	}
	return out
}

type userJSON struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Username  *string `json:"username"`
	AvatarURL *string `json:"avatarUrl"`
}

type verifyResponseJSON struct {
	User  userJSON `json:"user"`
	Token string   `json:"token"`
}

type errorJSON struct {
	Error struct {
		Code string `json:"code"`
	} `json:"error"`
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func decode(t *testing.T, data []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("decode %s: %v", data, err)
	}
}

func expectError(t *testing.T, resp *http.Response, body []byte, status int, code string) {
	t.Helper()
	var e errorJSON
	decode(t, body, &e)
	if resp.StatusCode != status || e.Error.Code != code {
		t.Fatalf("got %d %q, want %d %q (body %s)", resp.StatusCode, e.Error.Code, status, code, body)
	}
}

func TestWebSignInUsesHttpOnlyCookie(t *testing.T) {
	api := newTestAPI(t)
	c := api.newClient(t)

	// Signed out.
	resp, body := c.do("GET", "/me", nil)
	expectError(t, resp, body, http.StatusUnauthorized, "unauthenticated")

	out := c.signIn("ada@example.com", "web")
	if out.Token != "" {
		t.Fatal("web clients must not receive the token in the body")
	}
	if out.User.Email != "ada@example.com" || out.User.Username != nil {
		t.Fatalf("unexpected user %+v", out.User)
	}

	resp, body = c.do("GET", "/me", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /me after sign-in: %d %s", resp.StatusCode, body)
	}

	resp, _ = c.do("POST", "/auth/logout", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("logout: %d", resp.StatusCode)
	}
	resp, body = c.do("GET", "/me", nil)
	expectError(t, resp, body, http.StatusUnauthorized, "unauthenticated")
}

func TestVerifySetsCookieAttributes(t *testing.T) {
	api := newTestAPI(t)
	c := api.newClient(t)
	challengeID := c.requestCode("ada@example.com")
	resp, _ := c.do("POST", "/auth/verify", map[string]string{
		"challengeId": challengeID, "email": "ada@example.com", "code": api.mail.LastCode(t, "ada@example.com"), "platform": "web",
	})
	cookies := resp.Cookies()
	if len(cookies) != 1 || cookies[0].Name != sessionCookie {
		t.Fatalf("cookies = %v, want one %q cookie", cookies, sessionCookie)
	}
	ck := cookies[0]
	if !ck.HttpOnly || ck.SameSite != http.SameSiteLaxMode || ck.MaxAge < int(auth.SessionTTL.Seconds())-60 {
		t.Fatalf("cookie %+v: want HttpOnly, SameSite=Lax and a 90-day Max-Age", ck)
	}
}

func TestNativeSignInUsesBearerToken(t *testing.T) {
	api := newTestAPI(t)
	c := api.newClient(t)
	out := c.signIn("ada@example.com", "ios")
	if out.Token == "" {
		t.Fatal("native clients receive the token in the body")
	}
	if len(c.http.Jar.Cookies(mustURL(t, api.url))) != 0 {
		t.Fatal("native sign-in must not set a cookie")
	}
	resp, body := c.do("GET", "/me", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /me with bearer: %d %s", resp.StatusCode, body)
	}
}

func TestCrossOriginWritesAreRejected(t *testing.T) {
	api := newTestAPI(t)
	c := api.newClient(t)
	body := map[string]string{"email": "ada@example.com"}

	resp, data := c.do("POST", "/auth/request-code", body, "Origin", "https://evil.example", "Sec-Fetch-Site", "cross-site")
	expectError(t, resp, data, http.StatusForbidden, "forbidden_origin")

	resp, data = c.do("POST", "/auth/request-code", body, "Origin", webOrigin, "Sec-Fetch-Site", "same-site")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("request from the web app: %d %s", resp.StatusCode, data)
	}
}

func TestVerifyErrors(t *testing.T) {
	api := newTestAPI(t)
	c := api.newClient(t)

	resp, body := c.do("POST", "/auth/request-code", map[string]string{"email": "not-an-email"})
	expectError(t, resp, body, http.StatusBadRequest, "invalid_email")

	resp, body = c.do("POST", "/auth/verify", map[string]string{"email": "ada@example.com", "code": "123456", "platform": "web"})
	expectError(t, resp, body, http.StatusBadRequest, "code_expired")

	challengeID := c.requestCode("ada@example.com")
	code := api.mail.LastCode(t, "ada@example.com")
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	resp, body = c.do("POST", "/auth/verify", map[string]string{"challengeId": challengeID, "email": "ada@example.com", "code": wrong, "platform": "web"})
	expectError(t, resp, body, http.StatusBadRequest, "invalid_code")

	// Without its challenge ID, even the right code matches nothing.
	resp, body = c.do("POST", "/auth/verify", map[string]string{"email": "ada@example.com", "code": code, "platform": "web"})
	expectError(t, resp, body, http.StatusBadRequest, "code_expired")

	resp, body = c.do("POST", "/auth/verify", map[string]string{"challengeId": challengeID, "email": "ada@example.com", "code": code, "platform": "desktop"})
	expectError(t, resp, body, http.StatusBadRequest, "invalid_platform")

	// One client's hourly cap per address (5) applies even without the cooldown.
	for range auth.DefaultLimits.ClientEmailHourly.N - 1 {
		c.do("POST", "/auth/request-code", map[string]string{"email": "ada@example.com"})
	}
	resp, body = c.do("POST", "/auth/request-code", map[string]string{"email": "ada@example.com"})
	expectError(t, resp, body, http.StatusTooManyRequests, "rate_limited")
	if resp.Header.Get("Retry-After") == "" {
		t.Fatal("429 responses carry Retry-After")
	}
}

func TestUsername(t *testing.T) {
	api := newTestAPI(t)
	ada := api.newClient(t)
	ada.signIn("ada@example.com", "ios")
	bob := api.newClient(t)
	bob.signIn("bob@example.com", "android")

	resp, body := ada.do("PATCH", "/me", map[string]string{"username": "Ada.Lovelace"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("set username: %d %s", resp.StatusCode, body)
	}
	var u userJSON
	decode(t, body, &u)
	if u.Username == nil || *u.Username != "ada.lovelace" {
		t.Fatalf("username = %v, want ada.lovelace", u.Username)
	}

	resp, body = bob.do("PATCH", "/me", map[string]string{"username": "ADA.LOVELACE"})
	expectError(t, resp, body, http.StatusConflict, "username_taken")

	resp, body = bob.do("PATCH", "/me", map[string]string{"username": "no spaces"})
	expectError(t, resp, body, http.StatusBadRequest, "invalid_username")

	var avail struct{ Available bool }
	_, body = bob.do("GET", "/username-availability?username=ada.lovelace", nil)
	decode(t, body, &avail)
	if avail.Available {
		t.Fatal("a taken username is not available")
	}
	_, body = ada.do("GET", "/username-availability?username=ada.lovelace", nil)
	decode(t, body, &avail)
	if !avail.Available {
		t.Fatal("your own username counts as available")
	}
}

func TestSessionsListAndRevoke(t *testing.T) {
	api := newTestAPI(t)
	phone := api.newClient(t)
	phone.signIn("ada@example.com", "ios")
	laptop := api.newClient(t)
	laptop.signIn("ada@example.com", "web")

	var list struct {
		Sessions []struct {
			ID      string `json:"id"`
			Current bool   `json:"current"`
		} `json:"sessions"`
	}
	_, body := laptop.do("GET", "/me/sessions", nil)
	decode(t, body, &list)
	if len(list.Sessions) != 2 {
		t.Fatalf("sessions = %s, want 2", body)
	}
	var phoneID string
	for _, s := range list.Sessions {
		if !s.Current {
			phoneID = s.ID
		}
	}

	resp, _ := laptop.do("DELETE", "/me/sessions/"+phoneID, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("revoke: %d", resp.StatusCode)
	}
	resp, body = phone.do("GET", "/me", nil)
	expectError(t, resp, body, http.StatusUnauthorized, "unauthenticated")

	resp, body = laptop.do("DELETE", "/me/sessions/"+phoneID, nil)
	expectError(t, resp, body, http.StatusNotFound, "not_found")
}

func TestAvatarUploadAndServe(t *testing.T) {
	api := newTestAPI(t)
	c := api.newClient(t)
	c.signIn("ada@example.com", "ios")

	var img bytes.Buffer
	if err := png.Encode(&img, image.NewRGBA(image.Rect(0, 0, 900, 600))); err != nil {
		t.Fatal(err)
	}
	resp, body := c.upload("/me/avatar", img.Bytes())
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upload: %d %s", resp.StatusCode, body)
	}
	var u userJSON
	decode(t, body, &u)
	if u.AvatarURL == nil || !strings.HasPrefix(*u.AvatarURL, "http://api.test/avatars/"+u.ID+"/") {
		t.Fatalf("avatarUrl = %v", u.AvatarURL)
	}

	path := strings.TrimPrefix(*u.AvatarURL, "http://api.test")
	anon := api.newClient(t)
	resp, data := anon.do("GET", path, nil)
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "image/webp" {
		t.Fatalf("GET avatar: %d %s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	if !bytes.HasPrefix(data, []byte("RIFF")) {
		t.Fatal("avatar is not WebP")
	}

	// Replacing the avatar deletes the old file.
	resp, _ = c.upload("/me/avatar", img.Bytes())
	if resp.StatusCode != http.StatusOK || len(api.blobs.Keys()) != 1 {
		t.Fatalf("after replacing: %d, %d stored files, want 1", resp.StatusCode, len(api.blobs.Keys()))
	}

	resp, body = c.do("DELETE", "/me/avatar", nil)
	decode(t, body, &u)
	if resp.StatusCode != http.StatusOK || u.AvatarURL != nil || len(api.blobs.Keys()) != 0 {
		t.Fatalf("after removing: %d avatarUrl=%v files=%d", resp.StatusCode, u.AvatarURL, len(api.blobs.Keys()))
	}

	resp, body = c.upload("/me/avatar", []byte("definitely not an image"))
	expectError(t, resp, body, http.StatusBadRequest, "unsupported_image")
}

// A photo arriving over a slow connection must not hit the server-wide read
// timeout, which is sized for small JSON requests.
func TestSlowAvatarUploadOutlivesServerTimeouts(t *testing.T) {
	api := newTestAPI(t, func(s *http.Server) {
		s.ReadTimeout = 300 * time.Millisecond
		s.WriteTimeout = 300 * time.Millisecond
	})
	c := api.newClient(t)
	c.signIn("ada@example.com", "ios")

	var img bytes.Buffer
	if err := png.Encode(&img, image.NewRGBA(image.Rect(0, 0, 64, 64))); err != nil {
		t.Fatal(err)
	}
	resp, body := c.uploadFrom("/me/avatar", img.Bytes(), func(r io.Reader) io.Reader {
		return &slowReader{r: r, chunk: 16, delay: 50 * time.Millisecond}
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("slow upload: %d %s", resp.StatusCode, body)
	}
}

// slowReader hands out at most chunk bytes per Read, delay apart.
type slowReader struct {
	r     io.Reader
	chunk int
	delay time.Duration
}

func (s *slowReader) Read(p []byte) (int, error) {
	time.Sleep(s.delay)
	return s.r.Read(p[:min(len(p), s.chunk)])
}

func (c *client) upload(path string, file []byte) (*http.Response, []byte) {
	c.t.Helper()
	return c.uploadFrom(path, file, func(r io.Reader) io.Reader { return r })
}

func (c *client) uploadFrom(path string, file []byte, wrap func(io.Reader) io.Reader) (*http.Response, []byte) {
	c.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "photo.png")
	if err != nil {
		c.t.Fatal(err)
	}
	fw.Write(file)
	mw.Close()
	req, err := http.NewRequest("PUT", c.api.url+path, wrap(&buf))
	if err != nil {
		c.t.Fatal(err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return c.send(req)
}
