package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/auth"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/mcpserver"
)

// requireAPIToken lets only requests with a personal access token through
// to the MCP endpoint. Cookies and session tokens are not accepted there,
// so a page in the browser cannot reach it with the user's session.
func (s *Server) requireAPIToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r)
		if !ok {
			writeTokenRequired(w)
			return
		}
		p, err := s.auth.AuthenticateToken(r.Context(), token)
		switch {
		case errors.Is(err, auth.ErrUnauthenticated):
			writeTokenRequired(w)
			return
		case err != nil:
			writeInternalError(w, r, err)
			return
		}
		if p.User.Username == nil {
			writeError(w, http.StatusForbidden, "username_required", "Pick a username in the app first.")
			return
		}
		next.ServeHTTP(w, r.WithContext(mcpserver.WithPrincipal(r.Context(), p)))
	})
}

func bearerToken(r *http.Request) (string, bool) {
	scheme, value, _ := strings.Cut(r.Header.Get("Authorization"), " ")
	if !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token := strings.TrimSpace(value)
	return token, token != ""
}

func writeTokenRequired(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="friend-group-calendar"`)
	writeError(w, http.StatusUnauthorized, "invalid_token", "Send a personal access token (fgc_…) as \"Authorization: Bearer <token>\". Create one under Profile in the app.")
}
