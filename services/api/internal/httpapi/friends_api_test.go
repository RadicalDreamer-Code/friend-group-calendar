package httpapi

import (
	"net/http"
	"strings"
	"testing"
)

type personJSON struct {
	ID        string  `json:"id"`
	Username  string  `json:"username"`
	AvatarURL *string `json:"avatarUrl"`
}

type friendsJSON struct {
	Friends []struct {
		User  personJSON `json:"user"`
		Since string     `json:"since"`
	} `json:"friends"`
	Incoming []struct {
		User      personJSON `json:"user"`
		CreatedAt string     `json:"createdAt"`
	} `json:"incoming"`
	Outgoing []struct {
		User personJSON `json:"user"`
	} `json:"outgoing"`
}

// signUp signs a new user in on a native client and picks their username.
func (a *testAPI) signUp(t *testing.T, email, username string) (*client, verifyResponseJSON) {
	t.Helper()
	c := a.newClient(t)
	out := c.signIn(email, "ios")
	resp, body := c.do("PATCH", "/me", map[string]string{"username": username})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("set username: %d %s", resp.StatusCode, body)
	}
	return c, out
}

func (c *client) friends() friendsJSON {
	c.t.Helper()
	resp, body := c.do("GET", "/friends", nil)
	if resp.StatusCode != http.StatusOK {
		c.t.Fatalf("GET /friends: %d %s", resp.StatusCode, body)
	}
	var out friendsJSON
	decode(c.t, body, &out)
	return out
}

func TestFriendsFlow(t *testing.T) {
	api := newTestAPI(t)
	ada, adaMe := api.signUp(t, "ada@example.com", "ada")
	bob, bobMe := api.signUp(t, "bob@example.com", "bob")

	// Nothing yet, and lists are empty arrays rather than null.
	resp, body := ada.do("GET", "/friends", nil)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"friends":[]`) {
		t.Fatalf("empty /friends: %d %s", resp.StatusCode, body)
	}

	resp, body = ada.do("GET", "/users/search?q=%40Bo", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("search: %d %s", resp.StatusCode, body)
	}
	if strings.Contains(string(body), "@example.com") {
		t.Fatalf("search must not reveal email addresses: %s", body)
	}
	var search struct {
		Results []struct {
			User       personJSON `json:"user"`
			Friendship string     `json:"friendship"`
		} `json:"results"`
	}
	decode(t, body, &search)
	if len(search.Results) != 1 || search.Results[0].User.ID != bobMe.User.ID || search.Results[0].Friendship != "none" {
		t.Fatalf("search results = %s", body)
	}

	resp, body = ada.do("POST", "/friend-requests", map[string]string{"userId": bobMe.User.ID})
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"friendship":"outgoing"`) {
		t.Fatalf("send request: %d %s", resp.StatusCode, body)
	}

	resp, body = bob.do("GET", "/friends", nil)
	if strings.Contains(string(body), "@example.com") {
		t.Fatalf("friend requests must not reveal email addresses: %s", body)
	}
	var bobs friendsJSON
	decode(t, body, &bobs)
	if len(bobs.Incoming) != 1 || bobs.Incoming[0].User.Username != "ada" || bobs.Incoming[0].CreatedAt == "" {
		t.Fatalf("bob's requests: %d %s", resp.StatusCode, body)
	}

	resp, body = ada.do("POST", "/friend-requests/"+bobMe.User.ID+"/accept", nil)
	expectError(t, resp, body, http.StatusNotFound, "request_not_found")

	resp, _ = bob.do("POST", "/friend-requests/"+adaMe.User.ID+"/accept", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("accept: %d", resp.StatusCode)
	}
	if f := ada.friends(); len(f.Friends) != 1 || f.Friends[0].User.Username != "bob" || f.Friends[0].Since == "" {
		t.Fatalf("ada's friends = %+v", f)
	}

	resp, _ = ada.do("DELETE", "/friends/"+bobMe.User.ID, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("remove friend: %d", resp.StatusCode)
	}
	if f := bob.friends(); len(f.Friends) != 0 {
		t.Fatalf("bob's friends after removal = %+v", f)
	}
	resp, body = ada.do("DELETE", "/friends/"+bobMe.User.ID, nil)
	expectError(t, resp, body, http.StatusNotFound, "not_friends")

	// Cancel a request that was sent.
	ada.do("POST", "/friend-requests", map[string]string{"userId": bobMe.User.ID})
	resp, _ = ada.do("DELETE", "/friend-requests/"+bobMe.User.ID, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("cancel request: %d", resp.StatusCode)
	}
	if f := bob.friends(); len(f.Incoming) != 0 {
		t.Fatalf("bob's requests after cancel = %+v", f)
	}
}

func TestFriendsErrors(t *testing.T) {
	api := newTestAPI(t)
	ada, adaMe := api.signUp(t, "ada@example.com", "ada")

	resp, body := ada.do("GET", "/users/search?q=a", nil)
	expectError(t, resp, body, http.StatusBadRequest, "invalid_query")

	resp, body = ada.do("POST", "/friend-requests", map[string]string{"userId": adaMe.User.ID})
	expectError(t, resp, body, http.StatusBadRequest, "cannot_add_self")

	resp, body = ada.do("POST", "/friend-requests", map[string]string{"userId": "0190f3a0-0000-7000-8000-000000000000"})
	expectError(t, resp, body, http.StatusNotFound, "user_not_found")

	resp, body = ada.do("DELETE", "/friends/not-a-uuid", nil)
	expectError(t, resp, body, http.StatusNotFound, "user_not_found")

	// Someone who has not picked a username yet cannot use friends at all,
	// and cannot be found or added.
	newbie := api.newClient(t)
	newbieMe := newbie.signIn("new@example.com", "ios")
	resp, body = newbie.do("GET", "/friends", nil)
	expectError(t, resp, body, http.StatusForbidden, "username_required")
	resp, body = ada.do("POST", "/friend-requests", map[string]string{"userId": newbieMe.User.ID})
	expectError(t, resp, body, http.StatusNotFound, "user_not_found")

	signedOut := api.newClient(t)
	resp, body = signedOut.do("GET", "/friends", nil)
	expectError(t, resp, body, http.StatusUnauthorized, "unauthenticated")
}
