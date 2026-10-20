package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/auth"
)

type requestCodeRequest struct {
	Email string `json:"email"`
}

type requestCodeResponse struct {
	// ChallengeID goes back with the code to /auth/verify.
	ChallengeID uuid.UUID `json:"challengeId"`
}

func (s *Server) handleRequestCode(w http.ResponseWriter, r *http.Request) {
	var req requestCodeRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	challengeID, err := s.auth.RequestCode(r.Context(), req.Email, rateLimitKey(r))
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, requestCodeResponse{ChallengeID: challengeID})
	case errors.Is(err, auth.ErrInvalidEmail):
		writeError(w, http.StatusBadRequest, "invalid_email", "Please enter a valid email address.")
	case writeRateLimited(w, err):
	default:
		slog.ErrorContext(r.Context(), "request sign-in code", "err", err)
		writeError(w, http.StatusServiceUnavailable, "email_failed", "We could not send the email. Please try again.")
	}
}

type verifyRequest struct {
	ChallengeID string        `json:"challengeId"`
	Email       string        `json:"email"`
	Code        string        `json:"code"`
	Platform    auth.Platform `json:"platform"`
	DeviceName  string        `json:"deviceName"`
}

type verifyResponse struct {
	User userResponse `json:"user"`
	// Token is only returned to native apps, which keep it in secure storage.
	// Web clients get an HttpOnly cookie instead, which scripts cannot read.
	Token string `json:"token,omitempty"`
}

func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	var req verifyRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	// A missing or malformed ID matches no code, so it reads as code_expired.
	challengeID, _ := uuid.Parse(req.ChallengeID)
	res, err := s.auth.Verify(r.Context(), auth.VerifyInput{
		ChallengeID: challengeID,
		Email:       req.Email,
		Code:        req.Code,
		Platform:    req.Platform,
		DeviceName:  req.DeviceName,
		UserAgent:   r.UserAgent(),
		Client:      rateLimitKey(r),
	})
	switch {
	case err == nil:
	case errors.Is(err, auth.ErrInvalidEmail):
		writeError(w, http.StatusBadRequest, "invalid_email", "Please enter a valid email address.")
		return
	case errors.Is(err, auth.ErrInvalidPlatform):
		writeError(w, http.StatusBadRequest, "invalid_platform", `platform must be "web", "ios" or "android".`)
		return
	case errors.Is(err, auth.ErrInvalidCode):
		writeError(w, http.StatusBadRequest, "invalid_code", "That code is not right. Check the email and try again.")
		return
	case errors.Is(err, auth.ErrCodeExpired):
		writeError(w, http.StatusBadRequest, "code_expired", "This code has expired. Request a new one.")
		return
	case writeRateLimited(w, err):
		return
	default:
		writeInternalError(w, r, err)
		return
	}

	resp := verifyResponse{User: s.userResponse(res.User)}
	if req.Platform == auth.PlatformWeb {
		s.setSessionCookie(w, res.Token, res.ExpiresAt)
	} else {
		resp.Token = res.Token
	}
	writeJSON(w, http.StatusOK, resp)
}

// rateLimitKey identifies the client for the sign-in rate limits. An IPv6
// client usually controls a whole /64, so the /64 counts as one client;
// otherwise every address in it would get a budget of its own.
func rateLimitKey(r *http.Request) string {
	ip := middleware.GetClientIPAddr(r.Context()).Unmap()
	if ip.Is6() {
		if prefix, err := ip.Prefix(64); err == nil {
			return prefix.String()
		}
	}
	return middleware.GetClientIP(r.Context())
}

// handleLogout ends the current session. It succeeds without one too, so a
// client can always use it to get back to a clean signed-out state.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if p, ok := principalFrom(r.Context()); ok {
		if _, err := s.auth.RevokeSession(r.Context(), p.User.ID, p.Session.ID); err != nil {
			writeInternalError(w, r, err)
			return
		}
	}
	s.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}
