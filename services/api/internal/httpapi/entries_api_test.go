package httpapi

import (
	"net/http"
	"slices"
	"strings"
	"testing"
)

type entryJSON struct {
	ID        string      `json:"id"`
	Title     string      `json:"title"`
	Date      string      `json:"date"`
	Color     string      `json:"color"`
	CreatedBy *personJSON `json:"createdBy"`
	CreatedAt string      `json:"createdAt"`
	UpdatedAt string      `json:"updatedAt"`
	Accepted  []string    `json:"accepted"`
	Declined  []string    `json:"declined"`
}

func (c *client) entry(method, path string, body any, want int) entryJSON {
	c.t.Helper()
	resp, data := c.do(method, path, body)
	if resp.StatusCode != want {
		c.t.Fatalf("%s %s: %d %s", method, path, resp.StatusCode, data)
	}
	var out entryJSON
	decode(c.t, data, &out)
	return out
}

func TestEntriesFlow(t *testing.T) {
	api := newTestAPI(t)
	ada, adaMe := api.signUp(t, "ada@example.com", "ada")
	bob, bobMe := api.signUp(t, "bob@example.com", "bob")
	carla, carlaMe := api.signUp(t, "carla@example.com", "carla")
	befriend(t, ada, bob, bobMe.User.ID, adaMe.User.ID)
	befriend(t, ada, carla, carlaMe.User.ID, adaMe.User.ID)

	cal := ada.calendar("POST", "/calendars", map[string]string{"name": "Trip"}, http.StatusCreated)
	path := "/calendars/" + cal.ID
	ada.calendar("POST", path+"/members", map[string]string{"userId": bobMe.User.ID, "role": "editor"}, http.StatusOK)
	ada.calendar("POST", path+"/members", map[string]string{"userId": carlaMe.User.ID}, http.StatusOK)

	resp, body := carla.do("GET", path+"/entries?month=2026-10", nil)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"entries":[]`) {
		t.Fatalf("empty month: %d %s", resp.StatusCode, body)
	}

	e := bob.entry("POST", path+"/entries", map[string]string{"title": " Ski day ", "date": "2026-10-31", "color": "rose"}, http.StatusCreated)
	if e.Title != "Ski day" || e.Date != "2026-10-31" || e.Color != "rose" || e.CreatedBy == nil ||
		e.CreatedBy.Username != "bob" || e.Accepted == nil || len(e.Accepted) != 0 || e.Declined == nil {
		t.Fatalf("created = %+v", e)
	}
	if strings.Contains(string(body), "@example.com") {
		t.Fatalf("entries must not reveal email addresses: %s", body)
	}
	entryPath := path + "/entries/" + e.ID
	ada.entry("POST", path+"/entries", map[string]string{"title": "Halloween eve", "date": "2026-10-30", "color": "plum"}, http.StatusCreated)
	ada.entry("POST", path+"/entries", map[string]string{"title": "Next month", "date": "2026-11-01", "color": "sage"}, http.StatusCreated)

	// Viewers cannot add, edit or delete.
	resp, body = carla.do("POST", path+"/entries", map[string]string{"title": "x", "date": "2026-10-01", "color": "sage"})
	expectError(t, resp, body, http.StatusForbidden, "forbidden")
	resp, body = carla.do("PATCH", entryPath, map[string]string{"title": "x"})
	expectError(t, resp, body, http.StatusForbidden, "forbidden")
	resp, body = carla.do("DELETE", entryPath, nil)
	expectError(t, resp, body, http.StatusForbidden, "forbidden")

	// Everyone answers.
	carla.entry("PUT", entryPath+"/response", map[string]string{"status": "accepted"}, http.StatusOK)
	e = ada.entry("PUT", entryPath+"/response", map[string]string{"status": "declined"}, http.StatusOK)
	if !slices.Equal(e.Accepted, []string{carlaMe.User.ID}) || !slices.Equal(e.Declined, []string{adaMe.User.ID}) {
		t.Fatalf("answers = %+v", e)
	}
	e = ada.entry("DELETE", entryPath+"/response", nil, http.StatusOK)
	if len(e.Declined) != 0 {
		t.Fatalf("after taking it back = %+v", e)
	}

	resp, body = carla.do("GET", path+"/entries?month=2026-10", nil)
	var list struct {
		Entries []entryJSON `json:"entries"`
	}
	decode(t, body, &list)
	if len(list.Entries) != 2 || list.Entries[0].Title != "Halloween eve" || list.Entries[1].ID != e.ID ||
		!slices.Equal(list.Entries[1].Accepted, []string{carlaMe.User.ID}) {
		t.Fatalf("October: %d %s", resp.StatusCode, body)
	}

	// The next entry from a day on; null once nothing is left.
	var next struct {
		Entry *entryJSON `json:"entry"`
	}
	resp, body = carla.do("GET", path+"/entries/next?from=2026-10-31", nil)
	decode(t, body, &next)
	if resp.StatusCode != http.StatusOK || next.Entry == nil || next.Entry.ID != e.ID ||
		!slices.Equal(next.Entry.Accepted, []string{carlaMe.User.ID}) {
		t.Fatalf("next: %d %s", resp.StatusCode, body)
	}
	resp, body = carla.do("GET", path+"/entries/next?from=2026-11-02", nil)
	if resp.StatusCode != http.StatusOK || string(body) != `{"entry":null}`+"\n" {
		t.Fatalf("nothing next: %d %q", resp.StatusCode, body)
	}

	e = bob.entry("PATCH", entryPath, map[string]string{"title": "Ski weekend", "date": "2026-11-07", "color": "fjord"}, http.StatusOK)
	if e.Title != "Ski weekend" || e.Date != "2026-11-07" || e.Color != "fjord" || len(e.Accepted) != 0 {
		t.Fatalf("after moving = %+v", e)
	}
	if got := carla.entry("GET", entryPath, nil, http.StatusOK); got.Date != "2026-11-07" {
		t.Fatalf("get = %+v", got)
	}

	if resp, _ := bob.do("DELETE", entryPath, nil); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: %d", resp.StatusCode)
	}
	resp, body = carla.do("GET", entryPath, nil)
	expectError(t, resp, body, http.StatusNotFound, "entry_not_found")
	resp, body = carla.do("PUT", entryPath+"/response", map[string]string{"status": "accepted"})
	expectError(t, resp, body, http.StatusNotFound, "entry_not_found")
}

func TestEntriesErrors(t *testing.T) {
	api := newTestAPI(t)
	ada, _ := api.signUp(t, "ada@example.com", "ada")
	dora, _ := api.signUp(t, "dora@example.com", "dora")
	cal := ada.calendar("POST", "/calendars", map[string]string{"name": "Trip"}, http.StatusCreated)
	path := "/calendars/" + cal.ID
	e := ada.entry("POST", path+"/entries", map[string]string{"title": "Dinner", "date": "2026-10-01", "color": "sage"}, http.StatusCreated)
	entryPath := path + "/entries/" + e.ID

	for _, tt := range []struct {
		method, path string
		body         any
		status       int
		code         string
	}{
		{"GET", path + "/entries", nil, http.StatusBadRequest, "invalid_month"},
		{"GET", path + "/entries?month=2026-13", nil, http.StatusBadRequest, "invalid_month"},
		{"POST", path + "/entries", map[string]string{"title": " ", "date": "2026-10-01", "color": "sage"}, http.StatusBadRequest, "invalid_title"},
		{"POST", path + "/entries", map[string]string{"title": "x", "date": "01.10.2026", "color": "sage"}, http.StatusBadRequest, "invalid_date"},
		{"POST", path + "/entries", map[string]string{"title": "x", "date": "2026-10-01", "color": "red"}, http.StatusBadRequest, "invalid_color"},
		{"PATCH", entryPath, map[string]string{"date": "2100-01-01"}, http.StatusBadRequest, "invalid_date"},
		{"PUT", entryPath + "/response", map[string]string{"status": "maybe"}, http.StatusBadRequest, "invalid_status"},
		{"PUT", entryPath + "/response", map[string]string{}, http.StatusBadRequest, "invalid_status"},
		{"GET", path + "/entries/not-a-uuid", nil, http.StatusNotFound, "entry_not_found"},
		{"PATCH", path + "/entries/00000000-0000-0000-0000-000000000000", map[string]string{"title": "x"}, http.StatusNotFound, "entry_not_found"},
		{"GET", path + "/entries/next", nil, http.StatusBadRequest, "invalid_date"},
		{"GET", path + "/entries/next?from=2026-02-30", nil, http.StatusBadRequest, "invalid_date"},
		{"GET", "/calendars/not-a-uuid/entries?month=2026-10", nil, http.StatusNotFound, "calendar_not_found"},
	} {
		resp, body := ada.do(tt.method, tt.path, tt.body)
		expectError(t, resp, body, tt.status, tt.code)
	}

	// Outsiders get the same answer as for a calendar that does not exist.
	for _, tt := range []struct {
		method, path string
		body         any
	}{
		{"GET", path + "/entries?month=2026-10", nil},
		{"GET", entryPath, nil},
		{"POST", path + "/entries", map[string]string{"title": "x", "date": "2026-10-01", "color": "sage"}},
		{"PUT", entryPath + "/response", map[string]string{"status": "accepted"}},
		{"DELETE", entryPath, nil},
	} {
		resp, body := dora.do(tt.method, tt.path, tt.body)
		expectError(t, resp, body, http.StatusNotFound, "calendar_not_found")
	}
}
