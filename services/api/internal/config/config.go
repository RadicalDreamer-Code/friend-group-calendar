// Package config loads the API configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	// Addr is the address the HTTP server listens on, e.g. ":8080".
	Addr string
	// DatabaseURL is a Postgres connection string.
	DatabaseURL string
	// AllowedOrigins lists the web origins that may call the API with credentials.
	AllowedOrigins []string
	// ShutdownTimeout bounds how long in-flight requests get to finish on shutdown.
	ShutdownTimeout time.Duration
	// LogFormat is "text" (readable, for development) or "json".
	LogFormat string
}

func Load() (Config, error) {
	cfg := Config{
		Addr:            env("API_ADDR", ":8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		AllowedOrigins:  splitList(env("API_ALLOWED_ORIGINS", "http://localhost:8082")),
		ShutdownTimeout: 10 * time.Second,
		LogFormat:       env("LOG_FORMAT", "text"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
