package httpapi

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/avatar"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/store"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/users"
)

type userResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Username  *string   `json:"username"`
	AvatarURL *string   `json:"avatarUrl"`
	CreatedAt time.Time `json:"createdAt"`
}

func (s *Server) userResponse(u store.User) userResponse {
	resp := userResponse{ID: u.ID, Email: u.Email, Username: u.Username, CreatedAt: u.CreatedAt}
	if u.AvatarKey != nil {
		url := s.publicURL + "/avatars/" + *u.AvatarKey
		resp.AvatarURL = &url
	}
	return resp
}

func (s *Server) handleGetMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.userResponse(mustPrincipal(r.Context()).User))
}

type updateMeRequest struct {
	Username *string `json:"username"`
}

func (s *Server) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	var req updateMeRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	user := p.User
	if req.Username != nil {
		var err error
		user, err = s.users.SetUsername(r.Context(), p.User.ID, *req.Username)
		if !s.writeUsernameError(w, r, err) {
			return
		}
	}
	writeJSON(w, http.StatusOK, s.userResponse(user))
}

type usernameAvailabilityResponse struct {
	Available bool `json:"available"`
}

func (s *Server) handleUsernameAvailability(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	available, err := s.users.UsernameAvailable(r.Context(), p.User.ID, r.URL.Query().Get("username"))
	if !s.writeUsernameError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, usernameAvailabilityResponse{Available: available})
}

// writeUsernameError answers for a failed username operation and reports
// whether the handler may continue.
func (s *Server) writeUsernameError(w http.ResponseWriter, r *http.Request, err error) bool {
	switch {
	case err == nil:
		return true
	case errors.Is(err, users.ErrInvalidUsername):
		writeError(w, http.StatusBadRequest, "invalid_username", "Use 3–20 characters: lowercase letters, numbers, _ and .")
	case errors.Is(err, users.ErrUsernameTaken):
		writeError(w, http.StatusConflict, "username_taken", "That username is taken.")
	default:
		writeInternalError(w, r, err)
	}
	return false
}

// handlePutAvatar takes a multipart/form-data upload with the image in the
// "file" field; that is what both browsers and React Native send natively.
func (s *Server) handlePutAvatar(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())

	// Leave room for the multipart framing around the file itself.
	r.Body = http.MaxBytesReader(w, r.Body, avatar.MaxUploadBytes+64<<10)
	mr, err := r.MultipartReader()
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Send the image as multipart/form-data in a field named \"file\".")
		return
	}
	var upload []byte
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			writeUploadReadError(w, err)
			return
		}
		if part.FormName() != "file" {
			continue
		}
		upload, err = io.ReadAll(io.LimitReader(part, avatar.MaxUploadBytes+1))
		if err != nil {
			writeUploadReadError(w, err)
			return
		}
		break
	}
	switch {
	case upload == nil:
		writeError(w, http.StatusBadRequest, "invalid_body", "Send the image as multipart/form-data in a field named \"file\".")
		return
	case len(upload) > avatar.MaxUploadBytes:
		writeError(w, http.StatusRequestEntityTooLarge, "image_too_large", "The image is too large. The limit is 10 MB.")
		return
	}

	user, err := s.users.SetAvatar(r.Context(), p.User.ID, upload)
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, s.userResponse(user))
	case errors.Is(err, avatar.ErrUnsupported):
		writeError(w, http.StatusBadRequest, "unsupported_image", "Use a JPEG, PNG, WebP or GIF image.")
	case errors.Is(err, avatar.ErrTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, "image_too_large", "The image dimensions are too large.")
	default:
		writeInternalError(w, r, err)
	}
}

func writeUploadReadError(w http.ResponseWriter, err error) {
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		writeError(w, http.StatusRequestEntityTooLarge, "image_too_large", "The image is too large. The limit is 10 MB.")
		return
	}
	writeError(w, http.StatusBadRequest, "invalid_body", "The upload could not be read.")
}

func (s *Server) handleDeleteAvatar(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	user, err := s.users.RemoveAvatar(r.Context(), p.User.ID)
	if err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, s.userResponse(user))
}

type sessionResponse struct {
	ID         uuid.UUID `json:"id"`
	DeviceName string    `json:"deviceName"`
	Platform   string    `json:"platform"`
	CreatedAt  time.Time `json:"createdAt"`
	LastUsedAt time.Time `json:"lastUsedAt"`
	// Current marks the session this request was made with.
	Current bool `json:"current"`
}

type sessionsResponse struct {
	Sessions []sessionResponse `json:"sessions"`
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	sessions, err := s.auth.ListSessions(r.Context(), p.User.ID)
	if err != nil {
		writeInternalError(w, r, err)
		return
	}
	resp := sessionsResponse{Sessions: make([]sessionResponse, 0, len(sessions))}
	for _, sess := range sessions {
		resp.Sessions = append(resp.Sessions, sessionResponse{
			ID:         sess.ID,
			DeviceName: sess.DeviceName,
			Platform:   sess.Platform,
			CreatedAt:  sess.CreatedAt,
			LastUsedAt: sess.LastUsedAt,
			Current:    sess.ID == p.Session.ID,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleRevokeSession(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "sessionID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Session not found.")
		return
	}
	ok, err := s.auth.RevokeSession(r.Context(), p.User.ID, id)
	switch {
	case err != nil:
		writeInternalError(w, r, err)
	case !ok:
		writeError(w, http.StatusNotFound, "not_found", "Session not found.")
	default:
		if id == p.Session.ID {
			s.clearSessionCookie(w)
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) handleRevokeOtherSessions(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	if _, err := s.auth.RevokeOtherSessions(r.Context(), p.User.ID, p.Session.ID); err != nil {
		writeInternalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
