package httpapi

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

// A database that accepts connections but never answers must not hang /healthz.
func TestHealthTimesOutOnUnresponsiveDatabase(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var (
		mu    sync.Mutex
		conns []net.Conn
	)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			conns = append(conns, conn)
			mu.Unlock()
		}
	}()

	pool, err := pgxpool.New(context.Background(), "postgres://u:p@"+ln.Addr().String()+"/db?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	// Cleanups run last-in first-out: drop the server side first so the pool's
	// pending connect fails and pool.Close does not wait on it.
	t.Cleanup(func() {
		ln.Close()
		mu.Lock()
		defer mu.Unlock()
		for _, c := range conns {
			c.Close()
		}
	})

	rec := httptest.NewRecorder()
	start := time.Now()
	h, err := NewRouter(Options{DB: pool})
	if err != nil {
		t.Fatal(err)
	}
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/healthz", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if elapsed := time.Since(start); elapsed > healthTimeout+time.Second {
		t.Fatalf("healthz took %v, want about %v", elapsed, healthTimeout)
	}
}

// Forwarding headers come from the client and must not change its recorded IP.
func TestClientIPIgnoresForwardedHeaders(t *testing.T) {
	var got string
	h, err := NewRouter(Options{})
	if err != nil {
		t.Fatal(err)
	}
	r := h.(chi.Router)
	r.Get("/ip", func(w http.ResponseWriter, r *http.Request) {
		got = middleware.GetClientIP(r.Context())
	})

	req := httptest.NewRequest("GET", "/ip", nil)
	req.RemoteAddr = "203.0.113.9:5555"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	req.Header.Set("X-Real-IP", "1.2.3.4")
	req.Header.Set("True-Client-IP", "1.2.3.4")
	r.ServeHTTP(httptest.NewRecorder(), req)

	if got != "203.0.113.9" {
		t.Fatalf("client IP = %q, want 203.0.113.9", got)
	}
}
