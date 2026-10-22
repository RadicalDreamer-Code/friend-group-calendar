// Package httpapi wires HTTP routes, middleware and handlers.
package httpapi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/auth"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/blob"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/calendars"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/friends"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/users"
)

// healthTimeout bounds the database ping so /healthz answers even when the database hangs.
const healthTimeout = 2 * time.Second

type Options struct {
	DB             *pgxpool.Pool
	AllowedOrigins []string
	// TrustedProxies are CIDR prefixes of reverse proxies whose X-Forwarded-For is believed.
	TrustedProxies []string
	Auth           *auth.Service
	Users          *users.Service
	Friends        *friends.Service
	Calendars      *calendars.Service
	Avatars        blob.Store
	// PublicURL is the API's own address as clients see it, for avatar URLs.
	PublicURL    string
	CookieSecure bool
}

type Server struct {
	db           *pgxpool.Pool
	auth         *auth.Service
	users        *users.Service
	friends      *friends.Service
	calendars    *calendars.Service
	avatars      blob.Store
	publicURL    string
	cookieSecure bool
}

func NewRouter(o Options) (http.Handler, error) {
	s := &Server{
		db:           o.DB,
		auth:         o.Auth,
		users:        o.Users,
		friends:      o.Friends,
		calendars:    o.Calendars,
		avatars:      o.Avatars,
		publicURL:    o.PublicURL,
		cookieSecure: o.CookieSecure,
	}

	// Rejects cross-origin browser requests that change data (POST, PATCH,
	// DELETE, ...) unless they come from the web app. It reads Sec-Fetch-Site
	// and Origin, which browsers always send and native apps do not, so
	// requests from the iOS and Android app pass.
	csrf := http.NewCrossOriginProtection()
	for _, origin := range o.AllowedOrigins {
		if err := csrf.AddTrustedOrigin(origin); err != nil {
			return nil, fmt.Errorf("allowed origin %q: %w", origin, err)
		}
	}
	csrf.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusForbidden, "forbidden_origin", "Requests from this origin are not allowed.")
	}))

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	// The client IP (for rate limits and logs) is the TCP peer: headers like
	// X-Forwarded-For are spoofable. Behind a reverse proxy, the proxy's
	// X-Forwarded-For is read, but only on requests that come from it.
	if len(o.TrustedProxies) > 0 {
		r.Use(middleware.ClientIPFromXFF(o.TrustedProxies...))
	} else {
		r.Use(middleware.ClientIPFromRemoteAddr)
	}
	r.Use(requestLogger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   o.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           600,
	}))
	r.Use(csrf.Handler)

	r.Get("/healthz", s.handleHealth)
	r.Get("/avatars/{userID}/{file}", s.handleGetAvatar)

	r.Group(func(r chi.Router) {
		r.Use(s.loadSession)

		r.Post("/auth/request-code", s.handleRequestCode)
		r.Post("/auth/verify", s.handleVerify)
		r.Post("/auth/logout", s.handleLogout)

		r.Group(func(r chi.Router) {
			r.Use(requireAuth)
			r.Get("/me", s.handleGetMe)
			r.Patch("/me", s.handleUpdateMe)
			r.Put("/me/avatar", s.handlePutAvatar)
			r.Delete("/me/avatar", s.handleDeleteAvatar)
			r.Get("/me/sessions", s.handleListSessions)
			r.Post("/me/sessions/revoke-others", s.handleRevokeOtherSessions)
			r.Delete("/me/sessions/{sessionID}", s.handleRevokeSession)
			r.Get("/username-availability", s.handleUsernameAvailability)

			r.Group(func(r chi.Router) {
				r.Use(requireUsername)
				r.Get("/users/search", s.handleSearchUsers)
				r.Get("/friends", s.handleListFriends)
				r.Delete("/friends/{userID}", s.handleRemoveFriend)
				r.Post("/friend-requests", s.handleSendFriendRequest)
				r.Post("/friend-requests/{userID}/accept", s.handleAcceptFriendRequest)
				r.Delete("/friend-requests/{userID}", s.handleDeleteFriendRequest)

				r.Get("/calendars", s.handleListCalendars)
				r.Post("/calendars", s.handleCreateCalendar)
				r.Get("/calendars/{calendarID}", s.handleGetCalendar)
				r.Patch("/calendars/{calendarID}", s.handleUpdateCalendar)
				r.Delete("/calendars/{calendarID}", s.handleDeleteCalendar)
				r.Post("/calendars/{calendarID}/members", s.handleInviteMember)
				r.Patch("/calendars/{calendarID}/members/{userID}", s.handleUpdateMember)
				r.Delete("/calendars/{calendarID}/members/{userID}", s.handleRemoveMember)
				r.Post("/calendars/{calendarID}/transfer", s.handleTransferOwnership)
				r.Get("/calendars/{calendarID}/entries", s.handleListEntries)
				r.Post("/calendars/{calendarID}/entries", s.handleCreateEntry)
				r.Get("/calendars/{calendarID}/entries/{entryID}", s.handleGetEntry)
				r.Patch("/calendars/{calendarID}/entries/{entryID}", s.handleUpdateEntry)
				r.Delete("/calendars/{calendarID}/entries/{entryID}", s.handleDeleteEntry)
				r.Put("/calendars/{calendarID}/entries/{entryID}/response", s.handleRespond)
				r.Delete("/calendars/{calendarID}/entries/{entryID}/response", s.handleDeleteResponse)
			})
		})
	})

	return r, nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := http.StatusOK
	body := map[string]string{"status": "ok", "database": "ok"}
	ctx, cancel := context.WithTimeout(r.Context(), healthTimeout)
	defer cancel()
	if err := s.db.Ping(ctx); err != nil {
		slog.ErrorContext(r.Context(), "health check: database ping failed", "err", err)
		status = http.StatusServiceUnavailable
		body = map[string]string{"status": "degraded", "database": "unreachable"}
	}
	writeJSON(w, status, body)
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		slog.InfoContext(r.Context(), "request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"duration", time.Since(start),
			"request_id", middleware.GetReqID(r.Context()),
			"client_ip", middleware.GetClientIP(r.Context()),
		)
	})
}
