package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/auth"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/store"
)

// The token routes sit behind loadSession, which only knows sessions, so
// an access token cannot be used to create or revoke tokens.

type apiTokenResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	// LastUsedAt is null until the token is first used.
	LastUsedAt *time.Time `json:"lastUsedAt"`
	ExpiresAt  time.Time  `json:"expiresAt"`
}

func apiToken(t store.ApiToken) apiTokenResponse {
	return apiTokenResponse{ID: t.ID, Name: t.Name, CreatedAt: t.CreatedAt, LastUsedAt: t.LastUsedAt, ExpiresAt: t.ExpiresAt}
}

type apiTokensResponse struct {
	Tokens []apiTokenResponse `json:"tokens"`
}

func (s *Server) handleListAPITokens(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	tokens, err := s.auth.ListAPITokens(r.Context(), p.User.ID)
	if err != nil {
		writeInternalError(w, r, err)
		return
	}
	resp := apiTokensResponse{Tokens: make([]apiTokenResponse, 0, len(tokens))}
	for _, t := range tokens {
		resp.Tokens = append(resp.Tokens, apiToken(t))
	}
	writeJSON(w, http.StatusOK, resp)
}

type createAPITokenRequest struct {
	Name string `json:"name"`
	// ExpiresInDays is 30, 90 or 365; left out, it is 90.
	ExpiresInDays *int `json:"expiresInDays"`
}

type createdAPITokenResponse struct {
	apiTokenResponse
	// Token is shown this one time and never again.
	Token string `json:"token"`
}

func (s *Server) handleCreateAPIToken(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	var req createAPITokenRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	days := auth.DefaultAPITokenDays
	if req.ExpiresInDays != nil {
		days = *req.ExpiresInDays
	}
	res, err := s.auth.CreateAPIToken(r.Context(), p.User.ID, req.Name, days)
	switch {
	case err == nil:
		writeJSON(w, http.StatusCreated, createdAPITokenResponse{apiTokenResponse: apiToken(res.APIToken), Token: res.Token})
	case writeRateLimited(w, err):
	case errors.Is(err, auth.ErrInvalidTokenName):
		writeError(w, http.StatusBadRequest, "invalid_token_name", "Token names are 1–40 characters.")
	case errors.Is(err, auth.ErrInvalidTokenExpiry):
		writeError(w, http.StatusBadRequest, "invalid_expiry", "Tokens expire after 30, 90 or 365 days.")
	case errors.Is(err, auth.ErrTooManyTokens):
		writeError(w, http.StatusConflict, "too_many_tokens", "You can have at most 10 access tokens. Revoke one first.")
	default:
		writeInternalError(w, r, err)
	}
}

func (s *Server) handleRevokeAPIToken(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "tokenID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Token not found.")
		return
	}
	ok, err := s.auth.RevokeAPIToken(r.Context(), p.User.ID, id)
	switch {
	case err != nil:
		writeInternalError(w, r, err)
	case !ok:
		writeError(w, http.StatusNotFound, "not_found", "Token not found.")
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}
