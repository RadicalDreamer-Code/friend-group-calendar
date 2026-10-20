// Package httpapi wires HTTP routes, middleware and handlers.
package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
)

// healthTimeout bounds the database ping so /healthz answers even when the database hangs.
const healthTimeout = 2 * time.Second

type Server struct {
	db *pgxpool.Pool
}

func NewRouter(db *pgxpool.Pool, allowedOrigins []string) http.Handler {
	s := &Server{db: db}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	// Client IP from the TCP peer only; headers like X-Forwarded-For are spoofable.
	// Behind a reverse proxy, switch to middleware.ClientIPFromXFFTrustedProxies.
	r.Use(middleware.ClientIPFromRemoteAddr)
	r.Use(requestLogger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           600,
	}))

	r.Get("/healthz", s.handleHealth)

	return r
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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("write json response", "err", err)
	}
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
