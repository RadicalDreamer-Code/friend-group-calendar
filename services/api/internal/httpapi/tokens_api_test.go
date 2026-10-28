package httpapi

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

type apiTokenJSON struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	Token      string     `json:"token"`
}

// createToken makes an access token and returns it with its secret.
func (c *client) createToken(name string, body ...map[string]any) apiTokenJSON {
	c.t.Helper()
	req := map[string]any{"name": name}
	if len(body) > 0 {
		req = body[0]
	}
	resp, data := c.do("POST", "/me/tokens", req)
	if resp.StatusCode != http.StatusCreated {
		c.t.Fatalf("create token: %d %s", resp.StatusCode, data)
	}
	var out apiTokenJSON
	decode(c.t, data, &out)
	return out
}

func (c *client) tokens() []apiTokenJSON {
	c.t.Helper()
	resp, data := c.do("GET", "/me/tokens", nil)
	if resp.StatusCode != http.StatusOK {
		c.t.Fatalf("list tokens: %d %s", resp.StatusCode, data)
	}
	var out struct {
		Tokens []apiTokenJSON `json:"tokens"`
	}
	decode(c.t, data, &out)
	return out.Tokens
}

func TestAPITokensFlow(t *testing.T) {
	api := newTestAPI(t)
	ada, _ := api.signUp(t, "ada@example.com", "ada")

	if got := ada.tokens(); len(got) != 0 {
		t.Fatalf("tokens = %+v, want none", got)
	}
	created := ada.createToken("Claude Code")
	if !strings.HasPrefix(created.Token, "fgc_") || created.Name != "Claude Code" || created.LastUsedAt != nil {
		t.Fatalf("unexpected token %+v", created)
	}
	if days := created.ExpiresAt.Sub(created.CreatedAt).Hours() / 24; days != 90 {
		t.Fatalf("expires after %v days, want the default of 90", days)
	}
	year := ada.createToken("", map[string]any{"name": "laptop", "expiresInDays": 365})
	if days := year.ExpiresAt.Sub(year.CreatedAt).Hours() / 24; days != 365 {
		t.Fatalf("expires after %v days, want 365", days)
	}

	resp, data := ada.do("GET", "/me/tokens", nil)
	if strings.Contains(string(data), "fgc_") || strings.Contains(string(data), "token_hash") {
		t.Fatalf("the list must never include tokens: %s", data)
	}
	list := ada.tokens()
	if resp.StatusCode != http.StatusOK || len(list) != 2 || list[0].ID != year.ID {
		t.Fatalf("tokens = %+v, want both, newest first", list)
	}

	resp, _ = ada.do("DELETE", "/me/tokens/"+created.ID, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("revoke: %d", resp.StatusCode)
	}
	if list := ada.tokens(); len(list) != 1 || list[0].ID != year.ID {
		t.Fatalf("tokens after revoking = %+v", list)
	}
	resp, data = ada.do("DELETE", "/me/tokens/"+created.ID, nil)
	expectError(t, resp, data, http.StatusNotFound, "not_found")
	resp, data = ada.do("DELETE", "/me/tokens/not-a-uuid", nil)
	expectError(t, resp, data, http.StatusNotFound, "not_found")
}

func TestAPITokensErrors(t *testing.T) {
	api := newTestAPI(t)
	ada, _ := api.signUp(t, "ada@example.com", "ada")
	bob, _ := api.signUp(t, "bob@example.com", "bob")

	resp, data := ada.do("POST", "/me/tokens", map[string]any{"name": " "})
	expectError(t, resp, data, http.StatusBadRequest, "invalid_token_name")
	resp, data = ada.do("POST", "/me/tokens", map[string]any{"name": strings.Repeat("x", 41)})
	expectError(t, resp, data, http.StatusBadRequest, "invalid_token_name")
	resp, data = ada.do("POST", "/me/tokens", map[string]any{"name": "x", "expiresInDays": 7})
	expectError(t, resp, data, http.StatusBadRequest, "invalid_expiry")

	var last apiTokenJSON
	for range 10 {
		last = ada.createToken("token")
	}
	resp, data = ada.do("POST", "/me/tokens", map[string]any{"name": "one too many"})
	expectError(t, resp, data, http.StatusConflict, "too_many_tokens")

	// Someone else's token looks like one that does not exist.
	resp, data = bob.do("DELETE", "/me/tokens/"+last.ID, nil)
	expectError(t, resp, data, http.StatusNotFound, "not_found")
	if len(ada.tokens()) != 10 {
		t.Fatal("bob revoked one of ada's tokens")
	}

	// A username comes first, like for the rest of the app.
	newbie := api.newClient(t)
	newbie.signIn("new@example.com", "ios")
	resp, data = newbie.do("GET", "/me/tokens", nil)
	expectError(t, resp, data, http.StatusForbidden, "username_required")
}

func TestAPITokenDoesNotSignInToTheRESTAPI(t *testing.T) {
	api := newTestAPI(t)
	ada, _ := api.signUp(t, "ada@example.com", "ada")
	created := ada.createToken("Claude Code")

	bot := api.newClient(t)
	bot.token = created.Token
	resp, data := bot.do("GET", "/me", nil)
	expectError(t, resp, data, http.StatusUnauthorized, "unauthenticated")
	resp, data = bot.do("GET", "/calendars", nil)
	expectError(t, resp, data, http.StatusUnauthorized, "unauthenticated")
	// So a token cannot make more tokens either.
	resp, data = bot.do("POST", "/me/tokens", map[string]any{"name": "sneaky"})
	expectError(t, resp, data, http.StatusUnauthorized, "unauthenticated")
	if len(ada.tokens()) != 1 {
		t.Fatal("a token created another token")
	}
}
