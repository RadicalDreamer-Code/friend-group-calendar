package config

import (
	"slices"
	"strings"
	"testing"
)

// setValidEnv sets every required variable so each test can break one.
func setValidEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("AUTH_SECRET", strings.Repeat("s", minAuthSecretLen))
	t.Setenv("SMTP_HOST", "localhost")
	t.Setenv("S3_ENDPOINT", "localhost:9000")
	t.Setenv("S3_ACCESS_KEY", "key")
	t.Setenv("S3_SECRET_KEY", "secret")
}

func TestLoadValid(t *testing.T) {
	setValidEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.CookieSecure {
		t.Error("CookieSecure should default to true")
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	setValidEnv(t)
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected an error without DATABASE_URL")
	}
}

func TestLoadRejectsShortAuthSecret(t *testing.T) {
	setValidEnv(t)
	t.Setenv("AUTH_SECRET", "too-short")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "AUTH_SECRET") {
		t.Fatalf("err = %v, want an AUTH_SECRET error", err)
	}
}

func TestLoadRejectsInvalidBool(t *testing.T) {
	setValidEnv(t)
	t.Setenv("COOKIE_SECURE", "maybe")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "COOKIE_SECURE") {
		t.Fatalf("err = %v, want a COOKIE_SECURE error", err)
	}
}

func TestLoadParsesAllowedOrigins(t *testing.T) {
	setValidEnv(t)
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

func TestLoadTrustedProxies(t *testing.T) {
	setValidEnv(t)
	t.Setenv("TRUSTED_PROXIES", "172.28.80.0/24, fd00::/8")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"172.28.80.0/24", "fd00::/8"}
	if !slices.Equal(cfg.TrustedProxies, want) {
		t.Fatalf("TrustedProxies = %q, want %q", cfg.TrustedProxies, want)
	}

	// A bare address is a likely typo for a /32 and must not be guessed at.
	t.Setenv("TRUSTED_PROXIES", "172.28.80.2")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "TRUSTED_PROXIES") {
		t.Fatalf("err = %v, want a TRUSTED_PROXIES error", err)
	}
}
