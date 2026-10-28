package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// bearer adds an Authorization header to every request.
type bearer struct {
	token string
	base  http.RoundTripper
}

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return b.base.RoundTrip(r)
}

// connectMCP opens an MCP client session with an access token.
func (a *testAPI) connectMCP(t *testing.T, token string) (*mcp.ClientSession, error) {
	t.Helper()
	transport := &mcp.StreamableClientTransport{
		Endpoint:             a.url + "/mcp",
		HTTPClient:           &http.Client{Transport: bearer{token, http.DefaultTransport}},
		DisableStandaloneSSE: true,
		MaxRetries:           -1,
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1.0.0"}, nil)
	session, err := client.Connect(context.Background(), transport, nil)
	if err == nil {
		t.Cleanup(func() { session.Close() })
	}
	return session, err
}

func (a *testAPI) mustConnectMCP(t *testing.T, token string) *mcp.ClientSession {
	t.Helper()
	session, err := a.connectMCP(t, token)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	return session
}

// callTool calls a tool and decodes its structured content into out. It
// returns the result's text and whether it is a tool error.
func callTool(t *testing.T, s *mcp.ClientSession, name string, args map[string]any, out any) (text string, isError bool) {
	t.Helper()
	res, err := s.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: protocol error %v", name, err)
	}
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			text += tc.Text
		}
	}
	if !res.IsError && out != nil {
		data, err := json.Marshal(res.StructuredContent)
		if err != nil {
			t.Fatal(err)
		}
		decode(t, data, out)
	}
	return text, res.IsError
}

type mcpCalendarsJSON struct {
	Calendars []struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		Role          string `json:"role"`
		MemberCount   int    `json:"member_count"`
		CanAddEntries bool   `json:"can_add_entries"`
	} `json:"calendars"`
}

type mcpEntryJSON struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Date     string   `json:"date"`
	EndDate  string   `json:"end_date"`
	Color    string   `json:"color"`
	Cover    string   `json:"cover"`
	Creator  string   `json:"creator"`
	Going    []string `json:"going"`
	CantGo   []string `json:"cant_go"`
	MyAnswer string   `json:"my_answer"`
	URL      string   `json:"url"`
}

type mcpEntriesJSON struct {
	Calendar struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"calendar"`
	From    string         `json:"from"`
	To      string         `json:"to"`
	Entries []mcpEntryJSON `json:"entries"`
}

// mcpGroup is ada (owner), bob (editor) and carla (viewer) in one calendar,
// and dora on her own, each with an access token.
type mcpGroup struct {
	api                   *testAPI
	ada, bob, carla, dora *client
	adaID, bobID, carlaID string
	calID, path           string
	adaToken, carlaToken  string
	bobToken, doraToken   string
	doraCalID             string
}

func newMCPGroup(t *testing.T) mcpGroup {
	t.Helper()
	api := newTestAPI(t)
	g := mcpGroup{api: api}
	var adaMe, bobMe, carlaMe verifyResponseJSON
	g.ada, adaMe = api.signUp(t, "ada@example.com", "ada")
	g.bob, bobMe = api.signUp(t, "bob@example.com", "bob")
	g.carla, carlaMe = api.signUp(t, "carla@example.com", "carla")
	g.dora, _ = api.signUp(t, "dora@example.com", "dora")
	g.adaID, g.bobID, g.carlaID = adaMe.User.ID, bobMe.User.ID, carlaMe.User.ID
	befriend(t, g.ada, g.bob, g.bobID, g.adaID)
	befriend(t, g.ada, g.carla, g.carlaID, g.adaID)

	cal := g.ada.calendar("POST", "/calendars", map[string]string{"name": "Board game nights"}, http.StatusCreated)
	g.calID, g.path = cal.ID, "/calendars/"+cal.ID
	g.ada.calendar("POST", g.path+"/members", map[string]string{"userId": g.bobID, "role": "editor"}, http.StatusOK)
	g.ada.calendar("POST", g.path+"/members", map[string]string{"userId": g.carlaID, "role": "viewer"}, http.StatusOK)
	g.doraCalID = g.dora.calendar("POST", "/calendars", map[string]string{"name": "Dora's secret"}, http.StatusCreated).ID

	g.adaToken = g.ada.createToken("Claude Code").Token
	g.bobToken = g.bob.createToken("Claude Code").Token
	g.carlaToken = g.carla.createToken("Claude Code").Token
	g.doraToken = g.dora.createToken("Claude Code").Token
	return g
}

func TestMCPRejectsMissingAndBadTokens(t *testing.T) {
	g := newMCPGroup(t)
	api := g.api

	// Session tokens, from a native sign-in, are not access tokens.
	native := api.newClient(t)
	native.signIn("ada@example.com", "ios")

	revoked := g.ada.createToken("revoked")
	if resp, _ := g.ada.do("DELETE", "/me/tokens/"+revoked.ID, nil); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("revoke: %d", resp.StatusCode)
	}
	expired := g.ada.createToken("expired")
	if _, err := api.pool.Exec(context.Background(), "UPDATE api_tokens SET expires_at = now() - interval '1 second' WHERE id = $1", expired.ID); err != nil {
		t.Fatal(err)
	}

	for name, token := range map[string]string{
		"bad":     "fgc_" + strings.Repeat("A", 43),
		"session": native.token,
		"revoked": revoked.Token,
		"expired": expired.Token,
	} {
		if _, err := api.connectMCP(t, token); err == nil {
			t.Errorf("%s token: connected, want 401", name)
		}
		bot := api.newClient(t)
		bot.token = token
		resp, body := bot.do("POST", "/mcp", map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"},
			"Accept", "application/json, text/event-stream")
		expectError(t, resp, body, http.StatusUnauthorized, "invalid_token")
		if !strings.HasPrefix(resp.Header.Get("WWW-Authenticate"), "Bearer") {
			t.Errorf("%s token: WWW-Authenticate = %q, want Bearer", name, resp.Header.Get("WWW-Authenticate"))
		}
	}

	// No token at all, and the web session cookie does not count either.
	web := api.newClient(t)
	web.signIn("ada@example.com", "web")
	resp, body := web.do("POST", "/mcp", map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"},
		"Accept", "application/json, text/event-stream")
	expectError(t, resp, body, http.StatusUnauthorized, "invalid_token")
	if resp.Header.Get("WWW-Authenticate") == "" {
		t.Error("missing WWW-Authenticate")
	}

	// A good token works.
	session := api.mustConnectMCP(t, g.adaToken)
	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range tools.Tools {
		names = append(names, tool.Name)
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"create_entry", "list_calendars", "list_entries"}) {
		t.Fatalf("tools = %v", names)
	}
	if !strings.Contains(session.InitializeResult().Instructions, "list_calendars first") {
		t.Errorf("instructions = %q", session.InitializeResult().Instructions)
	}
}

func TestMCPNeedsAUsernameAndTheRightOrigin(t *testing.T) {
	api := newTestAPI(t)
	ada, _ := api.signUp(t, "ada@example.com", "ada")
	token := ada.createToken("Claude Code").Token

	bot := api.newClient(t)
	bot.token = token
	call := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}
	resp, body := bot.do("POST", "/mcp", call, "Accept", "application/json, text/event-stream",
		"Origin", "https://evil.example", "Sec-Fetch-Site", "cross-site")
	expectError(t, resp, body, http.StatusForbidden, "forbidden_origin")

	// Usernames cannot be removed, so take it away in the database.
	if _, err := api.pool.Exec(context.Background(), "UPDATE users SET username = NULL"); err != nil {
		t.Fatal(err)
	}
	resp, body = bot.do("POST", "/mcp", call, "Accept", "application/json, text/event-stream")
	expectError(t, resp, body, http.StatusForbidden, "username_required")
}

func TestMCPListCalendars(t *testing.T) {
	g := newMCPGroup(t)

	var out mcpCalendarsJSON
	text, isErr := callTool(t, g.api.mustConnectMCP(t, g.carlaToken), "list_calendars", nil, &out)
	if isErr || len(out.Calendars) != 1 {
		t.Fatalf("carla's calendars = %+v (%s)", out, text)
	}
	c := out.Calendars[0]
	if c.ID != g.calID || c.Name != "Board game nights" || c.Role != "viewer" || c.MemberCount != 3 || c.CanAddEntries {
		t.Fatalf("calendar = %+v", c)
	}
	if !strings.Contains(text, "Board game nights") || !strings.Contains(text, g.calID) {
		t.Errorf("summary = %q", text)
	}

	callTool(t, g.api.mustConnectMCP(t, g.bobToken), "list_calendars", nil, &out)
	if len(out.Calendars) != 1 || out.Calendars[0].Role != "editor" || !out.Calendars[0].CanAddEntries {
		t.Fatalf("bob's calendars = %+v", out)
	}
	callTool(t, g.api.mustConnectMCP(t, g.doraToken), "list_calendars", nil, &out)
	if len(out.Calendars) != 1 || out.Calendars[0].ID != g.doraCalID || out.Calendars[0].Role != "owner" {
		t.Fatalf("dora's calendars = %+v", out)
	}
}

func TestMCPListEntries(t *testing.T) {
	g := newMCPGroup(t)
	trip := g.bob.entry("POST", g.path+"/entries", map[string]string{
		"title": "Cabin trip", "date": "2026-10-30", "endDate": "2026-11-02", "color": "sage", "cover": "lake",
	}, http.StatusCreated)
	g.ada.entry("POST", g.path+"/entries", map[string]string{"title": "Catan night", "date": "2026-11-05", "color": "rose"}, http.StatusCreated)
	g.ada.entry("POST", g.path+"/entries", map[string]string{"title": "Way later", "date": "2027-01-15", "color": "rose"}, http.StatusCreated)
	g.ada.entry("PUT", g.path+"/entries/"+trip.ID+"/response", map[string]string{"status": "accepted"}, http.StatusOK)
	g.carla.entry("PUT", g.path+"/entries/"+trip.ID+"/response", map[string]string{"status": "declined"}, http.StatusOK)

	ada := g.api.mustConnectMCP(t, g.adaToken)

	// The trip started before the range and still shows.
	var out mcpEntriesJSON
	text, isErr := callTool(t, ada, "list_entries", map[string]any{"calendar_id": g.calID, "from": "2026-11-01", "to": "2026-11-30"}, &out)
	if isErr || out.Calendar.Name != "Board game nights" || out.From != "2026-11-01" || out.To != "2026-11-30" {
		t.Fatalf("list = %+v (%s)", out, text)
	}
	var titles []string
	for _, e := range out.Entries {
		titles = append(titles, e.Title)
	}
	if !slices.Equal(titles, []string{"Cabin trip", "Catan night"}) {
		t.Fatalf("titles = %v", titles)
	}
	e := out.Entries[0]
	if e.ID != trip.ID || e.Date != "2026-10-30" || e.EndDate != "2026-11-02" || e.Color != "sage" || e.Cover != "lake" ||
		e.Creator != "bob" || !slices.Equal(e.Going, []string{"ada"}) || !slices.Equal(e.CantGo, []string{"carla"}) ||
		e.MyAnswer != "going" || e.URL != "http://app.test/calendars/"+g.calID+"/entries/"+trip.ID {
		t.Fatalf("trip = %+v", e)
	}
	if catan := out.Entries[1]; catan.MyAnswer != "none" || catan.Going == nil || catan.CantGo == nil || catan.EndDate != catan.Date {
		t.Fatalf("catan = %+v", catan)
	}
	if !strings.Contains(text, "Cabin trip") || !strings.Contains(text, "2026-10-30 to 2026-11-02") {
		t.Errorf("summary = %q", text)
	}

	// And a range that ends while the trip is still going.
	callTool(t, ada, "list_entries", map[string]any{"calendar_id": g.calID, "from": "2026-10-01", "to": "2026-10-30"}, &out)
	if len(out.Entries) != 1 || out.Entries[0].Title != "Cabin trip" {
		t.Fatalf("october = %+v", out.Entries)
	}

	// Without dates: today, 2026-10-28, and the 30 days after it.
	callTool(t, ada, "list_entries", map[string]any{"calendar_id": g.calID}, &out)
	if out.From != "2026-10-28" || out.To != "2026-11-27" || len(out.Entries) != 2 {
		t.Fatalf("default range = %s to %s, %d entries", out.From, out.To, len(out.Entries))
	}
	callTool(t, ada, "list_entries", map[string]any{"calendar_id": g.calID, "from": "2026-12-20"}, &out)
	if out.To != "2027-01-19" || len(out.Entries) != 1 {
		t.Fatalf("from only = %s to %s, %d entries", out.From, out.To, len(out.Entries))
	}

	for name, args := range map[string]map[string]any{
		"93 days":       {"calendar_id": g.calID, "from": "2026-01-01", "to": "2026-04-03"},
		"backwards":     {"calendar_id": g.calID, "from": "2026-11-02", "to": "2026-11-01"},
		"bad date":      {"calendar_id": g.calID, "from": "2026-02-30"},
		"bad pattern":   {"calendar_id": g.calID, "from": "next tuesday"},
		"bad id":        {"calendar_id": "board game nights"},
		"no id":         {},
		"unknown field": {"calendar_id": g.calID, "month": "2026-11"},
	} {
		if text, isErr := callTool(t, ada, "list_entries", args, nil); !isErr {
			t.Errorf("%s: want a tool error, got %q", name, text)
		}
	}
}

func TestMCPCalendarOfSomeoneElse(t *testing.T) {
	g := newMCPGroup(t)
	ada := g.api.mustConnectMCP(t, g.adaToken)
	for _, id := range []string{g.doraCalID, "00000000-0000-7000-8000-000000000000"} {
		text, isErr := callTool(t, ada, "list_entries", map[string]any{"calendar_id": id}, nil)
		if !isErr || !strings.Contains(text, "Calendar not found") {
			t.Errorf("list_entries in %s = %q, want a not-found tool error", id, text)
		}
		text, isErr = callTool(t, ada, "create_entry", map[string]any{"calendar_id": id, "title": "Hi", "date": "2026-11-01"}, nil)
		if !isErr || !strings.Contains(text, "Calendar not found") {
			t.Errorf("create_entry in %s = %q, want a not-found tool error", id, text)
		}
	}
}

func TestMCPCreateEntry(t *testing.T) {
	g := newMCPGroup(t)
	bob := g.api.mustConnectMCP(t, g.bobToken)

	var e mcpEntryJSON
	text, isErr := callTool(t, bob, "create_entry", map[string]any{
		"calendar_id": g.calID, "title": " Catan night ", "date": "2026-11-07",
	}, &e)
	if isErr || e.Title != "Catan night" || e.Date != "2026-11-07" || e.EndDate != "2026-11-07" || e.Color != "fjord" ||
		e.Cover != "" || e.Creator != "bob" || e.MyAnswer != "none" || !strings.HasPrefix(e.URL, "http://app.test/calendars/") {
		t.Fatalf("created = %+v (%s)", e, text)
	}
	if !strings.Contains(text, "Board game nights") {
		t.Errorf("summary = %q", text)
	}
	callTool(t, bob, "create_entry", map[string]any{
		"calendar_id": g.calID, "title": "Lake weekend", "date": "2026-11-13", "end_date": "2026-11-15", "color": "moss", "cover": "lake",
	}, &e)
	if e.EndDate != "2026-11-15" || e.Color != "moss" || e.Cover != "lake" {
		t.Fatalf("trip = %+v", e)
	}
	// It is in the app like any other entry.
	got := g.ada.entry("GET", g.path+"/entries/"+e.ID, nil, http.StatusOK)
	if got.Title != "Lake weekend" || got.CreatedBy == nil || got.CreatedBy.Username != "bob" {
		t.Fatalf("in the app = %+v", got)
	}

	// A viewer gets a tool error, not a protocol error.
	text, isErr = callTool(t, g.api.mustConnectMCP(t, g.carlaToken), "create_entry", map[string]any{
		"calendar_id": g.calID, "title": "Sneaky", "date": "2026-11-07",
	}, nil)
	if !isErr || !strings.Contains(text, "can view this calendar but can't add to it") {
		t.Fatalf("viewer = %q", text)
	}

	for name, args := range map[string]map[string]any{
		"empty title":  {"calendar_id": g.calID, "title": "", "date": "2026-11-07"},
		"blank title":  {"calendar_id": g.calID, "title": "   ", "date": "2026-11-07"},
		"long title":   {"calendar_id": g.calID, "title": strings.Repeat("x", 81), "date": "2026-11-07"},
		"no date":      {"calendar_id": g.calID, "title": "x"},
		"1999":         {"calendar_id": g.calID, "title": "x", "date": "1999-12-31"},
		"not a day":    {"calendar_id": g.calID, "title": "x", "date": "2026-02-30"},
		"end before":   {"calendar_id": g.calID, "title": "x", "date": "2026-11-07", "end_date": "2026-11-06"},
		"32 days":      {"calendar_id": g.calID, "title": "x", "date": "2026-11-01", "end_date": "2026-12-02"},
		"bad color":    {"calendar_id": g.calID, "title": "x", "date": "2026-11-07", "color": "red"},
		"bad cover":    {"calendar_id": g.calID, "title": "x", "date": "2026-11-07", "cover": "volcano"},
		"time of day":  {"calendar_id": g.calID, "title": "x", "date": "2026-11-07T19:00:00Z"},
		"extra fields": {"calendar_id": g.calID, "title": "x", "date": "2026-11-07", "time": "19:00"},
	} {
		if text, isErr := callTool(t, bob, "create_entry", args, nil); !isErr {
			t.Errorf("%s: want a tool error, got %q", name, text)
		}
	}
	var list mcpEntriesJSON
	callTool(t, bob, "list_entries", map[string]any{"calendar_id": g.calID, "from": "2026-11-01", "to": "2026-12-31"}, &list)
	if len(list.Entries) != 2 {
		t.Fatalf("invalid calls added entries: %+v", list.Entries)
	}
}

func TestMCPCreateEntryRateLimited(t *testing.T) {
	g := newMCPGroup(t)
	bob := g.api.mustConnectMCP(t, g.bobToken)
	for i := range 30 {
		// Spread over two days; a day holds at most 20 entries.
		date := "2026-11-01"
		if i%2 == 1 {
			date = "2026-11-02"
		}
		if text, isErr := callTool(t, bob, "create_entry", map[string]any{"calendar_id": g.calID, "title": "x", "date": date}, nil); isErr {
			t.Fatalf("entry %d: %s", i, text)
		}
	}
	text, isErr := callTool(t, bob, "create_entry", map[string]any{"calendar_id": g.calID, "title": "x", "date": "2026-11-03"}, nil)
	if !isErr || !strings.Contains(text, "Too many entries") {
		t.Fatalf("31st entry = %q, want a rate limit", text)
	}
	// Reading is not limited, and ada has her own budget.
	if _, isErr := callTool(t, bob, "list_calendars", nil, nil); isErr {
		t.Fatal("list_calendars was limited")
	}
	if text, isErr := callTool(t, g.api.mustConnectMCP(t, g.adaToken), "create_entry", map[string]any{"calendar_id": g.calID, "title": "x", "date": "2026-11-03"}, nil); isErr {
		t.Fatalf("ada: %s", text)
	}
}
