package httpapi

import (
	"net/http"
	"strings"
	"testing"
)

type calendarJSON struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	MembersCanInvite bool   `json:"membersCanInvite"`
	Role             string `json:"role"`
	CanInvite        bool   `json:"canInvite"`
	Members          []struct {
		User      personJSON `json:"user"`
		Role      string     `json:"role"`
		CanInvite bool       `json:"canInvite"`
		JoinedAt  string     `json:"joinedAt"`
	} `json:"members"`
}

type calendarsJSON struct {
	Calendars []struct {
		ID          string       `json:"id"`
		Name        string       `json:"name"`
		Role        string       `json:"role"`
		CanInvite   bool         `json:"canInvite"`
		MemberCount int          `json:"memberCount"`
		Members     []personJSON `json:"members"`
	} `json:"calendars"`
}

// befriend makes two signed-up users friends.
func befriend(t *testing.T, a *client, b *client, bID string, aID string) {
	t.Helper()
	if resp, body := a.do("POST", "/friend-requests", map[string]string{"userId": bID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("send request: %d %s", resp.StatusCode, body)
	}
	if resp, body := b.do("POST", "/friend-requests/"+aID+"/accept", nil); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("accept: %d %s", resp.StatusCode, body)
	}
}

func (c *client) calendar(method, path string, body any, want int) calendarJSON {
	c.t.Helper()
	resp, data := c.do(method, path, body)
	if resp.StatusCode != want {
		c.t.Fatalf("%s %s: %d %s", method, path, resp.StatusCode, data)
	}
	var out calendarJSON
	decode(c.t, data, &out)
	return out
}

func TestCalendarsFlow(t *testing.T) {
	api := newTestAPI(t)
	ada, adaMe := api.signUp(t, "ada@example.com", "ada")
	bob, bobMe := api.signUp(t, "bob@example.com", "bob")
	carla, carlaMe := api.signUp(t, "carla@example.com", "carla")
	befriend(t, ada, bob, bobMe.User.ID, adaMe.User.ID)
	befriend(t, bob, carla, carlaMe.User.ID, bobMe.User.ID)

	resp, body := ada.do("GET", "/calendars", nil)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"calendars":[]`) {
		t.Fatalf("empty list: %d %s", resp.StatusCode, body)
	}

	cal := ada.calendar("POST", "/calendars", map[string]string{"name": " Ski trip "}, http.StatusCreated)
	if cal.Name != "Ski trip" || cal.Role != "owner" || !cal.CanInvite || len(cal.Members) != 1 {
		t.Fatalf("created = %+v", cal)
	}
	path := "/calendars/" + cal.ID

	// Bob is not a member yet: the calendar does not exist for him.
	resp, body = bob.do("GET", path, nil)
	expectError(t, resp, body, http.StatusNotFound, "calendar_not_found")

	cal = ada.calendar("POST", path+"/members", map[string]string{"userId": bobMe.User.ID}, http.StatusOK)
	if len(cal.Members) != 2 || cal.Members[1].User.Username != "bob" || cal.Members[1].Role != "viewer" {
		t.Fatalf("after invite = %+v", cal)
	}
	if strings.Contains(string(body), "@example.com") {
		t.Fatalf("members must not reveal email addresses: %s", body)
	}

	resp, body = bob.do("GET", "/calendars", nil)
	var list calendarsJSON
	decode(t, body, &list)
	if len(list.Calendars) != 1 || list.Calendars[0].Role != "viewer" || list.Calendars[0].MemberCount != 2 ||
		len(list.Calendars[0].Members) != 2 || list.Calendars[0].Members[0].Username != "ada" {
		t.Fatalf("bob's list: %d %s", resp.StatusCode, body)
	}

	// Bob cannot invite Carla until he is allowed to.
	resp, body = bob.do("POST", path+"/members", map[string]string{"userId": carlaMe.User.ID})
	expectError(t, resp, body, http.StatusForbidden, "forbidden")
	cal = ada.calendar("PATCH", path+"/members/"+bobMe.User.ID, map[string]any{"canInvite": true, "role": "editor"}, http.StatusOK)
	if cal.Members[1].Role != "editor" || !cal.Members[1].CanInvite {
		t.Fatalf("bob after update = %+v", cal.Members[1])
	}
	resp, body = ada.do("POST", path+"/members", map[string]string{"userId": carlaMe.User.ID})
	expectError(t, resp, body, http.StatusForbidden, "not_friends")
	cal = bob.calendar("POST", path+"/members", map[string]string{"userId": carlaMe.User.ID, "role": "editor"}, http.StatusOK)
	if len(cal.Members) != 3 || cal.Role != "editor" || !cal.CanInvite {
		t.Fatalf("after bob invited carla = %+v", cal)
	}

	// Settings and member management are the owner's.
	resp, body = bob.do("PATCH", path, map[string]string{"name": "Mine now"})
	expectError(t, resp, body, http.StatusForbidden, "forbidden")
	cal = ada.calendar("PATCH", path, map[string]any{"name": "Alps", "membersCanInvite": true}, http.StatusOK)
	if cal.Name != "Alps" || !cal.MembersCanInvite {
		t.Fatalf("after rename = %+v", cal)
	}
	resp, body = ada.do("DELETE", path+"/members/"+adaMe.User.ID, nil)
	expectError(t, resp, body, http.StatusConflict, "owner_must_transfer")

	cal = ada.calendar("POST", path+"/transfer", map[string]string{"userId": bobMe.User.ID}, http.StatusOK)
	if cal.Role != "editor" || cal.Members[0].User.Username != "bob" || cal.Members[0].Role != "owner" {
		t.Fatalf("after transfer = %+v", cal)
	}

	// Carla leaves, Bob removes Ada, then deletes the calendar.
	if resp, _ := carla.do("DELETE", path+"/members/"+carlaMe.User.ID, nil); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("leave: %d", resp.StatusCode)
	}
	if resp, _ := bob.do("DELETE", path+"/members/"+adaMe.User.ID, nil); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("remove: %d", resp.StatusCode)
	}
	resp, body = ada.do("DELETE", path, nil)
	expectError(t, resp, body, http.StatusNotFound, "calendar_not_found")
	if resp, _ := bob.do("DELETE", path, nil); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: %d", resp.StatusCode)
	}
	resp, body = bob.do("GET", path, nil)
	expectError(t, resp, body, http.StatusNotFound, "calendar_not_found")
}

func TestCalendarsErrors(t *testing.T) {
	api := newTestAPI(t)
	ada, adaMe := api.signUp(t, "ada@example.com", "ada")

	resp, body := ada.do("POST", "/calendars", map[string]string{"name": "  "})
	expectError(t, resp, body, http.StatusBadRequest, "invalid_name")
	resp, body = ada.do("GET", "/calendars/not-a-uuid", nil)
	expectError(t, resp, body, http.StatusNotFound, "calendar_not_found")

	cal := ada.calendar("POST", "/calendars", map[string]string{"name": "Trip"}, http.StatusCreated)
	path := "/calendars/" + cal.ID
	resp, body = ada.do("POST", path+"/members", map[string]string{"userId": adaMe.User.ID, "role": "boss"})
	expectError(t, resp, body, http.StatusBadRequest, "invalid_role")
	resp, body = ada.do("POST", path+"/members", map[string]string{"userId": adaMe.User.ID})
	expectError(t, resp, body, http.StatusConflict, "already_member")
	resp, body = ada.do("PATCH", path+"/members/"+adaMe.User.ID, map[string]string{"role": "viewer"})
	expectError(t, resp, body, http.StatusBadRequest, "member_is_owner")
	resp, body = ada.do("PATCH", path+"/members/not-a-uuid", map[string]string{"role": "viewer"})
	expectError(t, resp, body, http.StatusNotFound, "member_not_found")
	resp, body = ada.do("POST", path+"/transfer", map[string]string{"userId": "0190f3a0-0000-7000-8000-000000000000"})
	expectError(t, resp, body, http.StatusNotFound, "member_not_found")

	// Calendars need a username, like friends.
	newbie := api.newClient(t)
	newbie.signIn("new@example.com", "ios")
	resp, body = newbie.do("GET", "/calendars", nil)
	expectError(t, resp, body, http.StatusForbidden, "username_required")
}
