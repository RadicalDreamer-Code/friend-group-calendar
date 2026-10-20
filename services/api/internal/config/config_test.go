package config

import (
	"slices"
	"testing"
)

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected an error without DATABASE_URL")
	}
}

func TestLoadParsesAllowedOrigins(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("API_ALLOWED_ORIGINS", " http://localhost:8082, ,https://app.example.com ")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"http://localhost:8082", "https://app.example.com"}
	if !slices.Equal(cfg.AllowedOrigins, want) {
		t.Fatalf("AllowedOrigins = %q, want %q", cfg.AllowedOrigins, want)
	}
}
