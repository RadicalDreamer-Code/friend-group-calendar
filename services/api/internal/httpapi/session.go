package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/auth"
)

const sessionCookie = "session"

type principalKey struct{}

// principalFrom returns the signed-in user, if any.
func principalFrom(ctx context.Context) (auth.Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(auth.Principal)
	return p, ok
}

// mustPrincipal is for handlers behind requireAuth.
func mustPrincipal(ctx context.Context) auth.Principal {
	p, ok := principalFrom(ctx)
	if !ok {
		panic("httpapi: handler needs requireAuth")
	}
	return p
}

// sessionToken reads the token from "Authorization: Bearer" (native apps)
// or from the session cookie (web).
func sessionToken(r *http.Request) (token string, fromCookie bool) {
	if h := r.Header.Get("Authorization"); h != "" {
		scheme, value, _ := strings.Cut(h, " ")
		if strings.EqualFold(scheme, "Bearer") {
			return strings.TrimSpace(value), false
		}
		return "", false
	}
	if c, err := r.Cookie(sessionCookie); err == nil {
		return c.Value, true
	}
	return "", false
}

// loadSession attaches the principal when the request carries a valid
// session, and renews the cookie when the session's expiry moved forward.
// Requests without one continue anonymously; requireAuth rejects them.
func (s *Server) loadSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, fromCookie := sessionToken(r)
		if token == "" {
			next.ServeHTTP(w, r)
			return
		}
		p, renewed, err := s.auth.Authenticate(r.Context(), token)
		switch {
		case errors.Is(err, auth.ErrUnauthenticated):
			if fromCookie {
				s.clearSessionCookie(w)
			}
			next.ServeHTTP(w, r)
			return
		case err != nil:
			writeInternalError(w, r, err)
			return
		}
		if renewed && fromCookie {
			s.setSessionCookie(w, token, p.Session.ExpiresAt)
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey{}, p)))
	})
}

func requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := principalFrom(r.Context()); !ok {
			writeError(w, http.StatusUnauthorized, "unauthenticated", "Please sign in.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) setSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}
