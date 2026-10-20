package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/auth"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/blob"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/config"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/database"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/httpapi"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/mail"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/users"
)

// cleanupInterval is how often expired sign-in codes and sessions are deleted.
const cleanupInterval = time.Hour

func main() {
	if err := run(); err != nil {
		slog.Error("api stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	setupLogger(cfg.LogFormat)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool); err != nil {
		return err
	}

	sender, err := mail.NewSMTP(cfg.Mail.Host, cfg.Mail.Port, cfg.Mail.Username, cfg.Mail.Password, cfg.Mail.From)
	if err != nil {
		return err
	}
	avatars, err := blob.NewS3(ctx, cfg.S3)
	if err != nil {
		return err
	}
	authService := auth.NewService(pool, sender, cfg.AuthSecret, auth.DefaultLimits)
	go runCleanup(ctx, authService)

	handler, err := httpapi.NewRouter(httpapi.Options{
		DB:             pool,
		AllowedOrigins: cfg.AllowedOrigins,
		Auth:           authService,
		Users:          users.NewService(pool, avatars),
		Avatars:        avatars,
		PublicURL:      cfg.PublicURL,
		CookieSecure:   cfg.CookieSecure,
	})
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("api listening", "addr", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func runCleanup(ctx context.Context, a *auth.Service) {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for {
		if err := a.Cleanup(ctx); err != nil && ctx.Err() == nil {
			slog.Error("cleanup", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func setupLogger(format string) {
	var h slog.Handler
	if format == "json" {
		h = slog.NewJSONHandler(os.Stdout, nil)
	} else {
		h = slog.NewTextHandler(os.Stdout, nil)
	}
	slog.SetDefault(slog.New(h))
}
