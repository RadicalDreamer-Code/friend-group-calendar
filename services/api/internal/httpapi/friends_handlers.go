package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/friends"
)

// personResponse is another user as the signed-in user sees them: no email.
type personResponse struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	AvatarURL *string   `json:"avatarUrl"`
}

func (s *Server) person(id uuid.UUID, username string, avatarKey *string) personResponse {
	return personResponse{ID: id, Username: username, AvatarURL: s.avatarURL(avatarKey)}
}

func (s *Server) personResponse(p friends.Person) personResponse {
	return s.person(p.ID, p.Username, p.AvatarKey)
}

// requireUsername keeps users who have not picked a username yet out of the
// friends and calendar features: nobody could tell who someone is.
func requireUsername(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if mustPrincipal(r.Context()).User.Username == nil {
			writeError(w, http.StatusForbidden, "username_required", "Pick a username first.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

type searchResultResponse struct {
	User       personResponse `json:"user"`
	Friendship friends.Status `json:"friendship"`
}

type searchResponse struct {
	Results []searchResultResponse `json:"results"`
}

func (s *Server) handleSearchUsers(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	results, err := s.friends.Search(r.Context(), p.User.ID, r.URL.Query().Get("q"))
	switch {
	case errors.Is(err, friends.ErrInvalidQuery):
		writeError(w, http.StatusBadRequest, "invalid_query", "Search for 2–20 characters of a username.")
		return
	case writeRateLimited(w, err):
		return
	case err != nil:
		writeInternalError(w, r, err)
		return
	}
	resp := searchResponse{Results: make([]searchResultResponse, 0, len(results))}
	for _, res := range results {
		resp.Results = append(resp.Results, searchResultResponse{User: s.personResponse(res.Person), Friendship: res.Status})
	}
	writeJSON(w, http.StatusOK, resp)
}

type friendResponse struct {
	User  personResponse `json:"user"`
	Since time.Time      `json:"since"`
}

type friendRequestResponse struct {
	User      personResponse `json:"user"`
	CreatedAt time.Time      `json:"createdAt"`
}

type friendsResponse struct {
	Friends  []friendResponse        `json:"friends"`
	Incoming []friendRequestResponse `json:"incoming"`
	Outgoing []friendRequestResponse `json:"outgoing"`
}

func (s *Server) handleListFriends(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	o, err := s.friends.Overview(r.Context(), p.User.ID)
	if err != nil {
		writeInternalError(w, r, err)
		return
	}
	resp := friendsResponse{
		Friends:  make([]friendResponse, 0, len(o.Friends)),
		Incoming: s.friendRequests(o.Incoming),
		Outgoing: s.friendRequests(o.Outgoing),
	}
	for _, f := range o.Friends {
		resp.Friends = append(resp.Friends, friendResponse{User: s.personResponse(f.Person), Since: f.Since})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) friendRequests(reqs []friends.Request) []friendRequestResponse {
	out := make([]friendRequestResponse, 0, len(reqs))
	for _, req := range reqs {
		out = append(out, friendRequestResponse{User: s.personResponse(req.Person), CreatedAt: req.CreatedAt})
	}
	return out
}

type sendFriendRequestRequest struct {
	UserID uuid.UUID `json:"userId"`
}

type sendFriendRequestResponse struct {
	Friendship friends.Status `json:"friendship"`
}

func (s *Server) handleSendFriendRequest(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	var req sendFriendRequestRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	status, err := s.friends.SendRequest(r.Context(), p.User, req.UserID)
	if !writeFriendsError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, sendFriendRequestResponse{Friendship: status})
}

func (s *Server) handleAcceptFriendRequest(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	other, ok := userIDParam(w, r)
	if !ok {
		return
	}
	if writeFriendsError(w, r, s.friends.AcceptRequest(r.Context(), p.User.ID, other)) {
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) handleDeleteFriendRequest(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	other, ok := userIDParam(w, r)
	if !ok {
		return
	}
	if writeFriendsError(w, r, s.friends.DeleteRequest(r.Context(), p.User.ID, other)) {
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) handleRemoveFriend(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	other, ok := userIDParam(w, r)
	if !ok {
		return
	}
	if writeFriendsError(w, r, s.friends.RemoveFriend(r.Context(), p.User.ID, other)) {
		w.WriteHeader(http.StatusNoContent)
	}
}

// userIDParam reads the {userID} path parameter and answers 404 when it is
// not a UUID, since no user could have that ID.
func userIDParam(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "user_not_found", "User not found.")
		return uuid.Nil, false
	}
	return id, true
}

// writeFriendsError answers for a failed friends operation and reports
// whether the handler may continue.
func writeFriendsError(w http.ResponseWriter, r *http.Request, err error) bool {
	switch {
	case err == nil:
		return true
	case writeRateLimited(w, err):
	case errors.Is(err, friends.ErrUserNotFound):
		writeError(w, http.StatusNotFound, "user_not_found", "User not found.")
	case errors.Is(err, friends.ErrSelf):
		writeError(w, http.StatusBadRequest, "cannot_add_self", "You cannot add yourself.")
	case errors.Is(err, friends.ErrUsernameRequired):
		writeError(w, http.StatusForbidden, "username_required", "Pick a username first.")
	case errors.Is(err, friends.ErrRequestNotFound):
		writeError(w, http.StatusNotFound, "request_not_found", "This friend request no longer exists.")
	case errors.Is(err, friends.ErrNotFriends):
		writeError(w, http.StatusNotFound, "not_friends", "You are not friends.")
	default:
		writeInternalError(w, r, err)
	}
	return false
}
