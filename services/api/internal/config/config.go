// Package config loads the API configuration from environment variables.
package config

import (
	"fmt"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	// Addr is the address the HTTP server listens on, e.g. ":8080".
	Addr string
	// PublicURL is the address clients reach the API at, used to build avatar URLs.
	PublicURL string
	// AppURL is the web app's address, for links to it (from MCP tools).
	AppURL string
	// DatabaseURL is a Postgres connection string.
	DatabaseURL string
	// AllowedOrigins lists the web origins that may call the API with credentials.
	AllowedOrigins []string
	// TrustedProxies are the CIDR prefixes of reverse proxies in front of the
	// API. X-Forwarded-For is read only on requests from them; without any,
	// the client IP is the TCP peer.
	TrustedProxies []string
	// ShutdownTimeout bounds how long in-flight requests get to finish on shutdown.
	ShutdownTimeout time.Duration
	// LogFormat is "text" (readable, for development) or "json".
	LogFormat string

	// AuthSecret keys the HMAC that sign-in codes are stored under.
	AuthSecret []byte
	// CookieSecure sets the Secure flag on the session cookie. Browsers accept
	// Secure cookies on http://localhost, so this stays on in development too;
	// turn it off only to test the web app over plain http on a LAN address.
	CookieSecure bool

	Mail MailConfig
	S3   S3Config
}

type MailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	// From is the sender, e.g. "Friend Calendar <hello@example.com>".
	From string
}

type S3Config struct {
	// Endpoint is host[:port] without scheme, e.g. "rustfs:9000" or "<account>.r2.cloudflarestorage.com".
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	UseSSL    bool
	// CreateBucket creates the bucket on startup if it is missing (local development).
	CreateBucket bool
}

// minAuthSecretLen matches the 256-bit output of HMAC-SHA256, the recommended minimum key size.
const minAuthSecretLen = 32

func Load() (Config, error) {
	var errs []string
	fail := func(format string, args ...any) { errs = append(errs, fmt.Sprintf(format, args...)) }

	cfg := Config{
		Addr:            env("API_ADDR", ":8080"),
		PublicURL:       strings.TrimRight(env("API_PUBLIC_URL", "http://localhost:8080"), "/"),
		AppURL:          strings.TrimRight(env("APP_PUBLIC_URL", "http://localhost:8082"), "/"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		AllowedOrigins:  splitList(env("API_ALLOWED_ORIGINS", "http://localhost:8082")),
		TrustedProxies:  splitList(os.Getenv("TRUSTED_PROXIES")),
		ShutdownTimeout: 10 * time.Second,
		LogFormat:       env("LOG_FORMAT", "text"),
		AuthSecret:      []byte(os.Getenv("AUTH_SECRET")),
		CookieSecure:    envBool("COOKIE_SECURE", true, fail),
		Mail: MailConfig{
			Host:     os.Getenv("SMTP_HOST"),
			Port:     envInt("SMTP_PORT", 587, fail),
			Username: os.Getenv("SMTP_USERNAME"),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     env("MAIL_FROM", "Friend Calendar <no-reply@localhost>"),
		},
		S3: S3Config{
			Endpoint:     os.Getenv("S3_ENDPOINT"),
			Region:       env("S3_REGION", "us-east-1"),
			Bucket:       env("S3_BUCKET", "avatars"),
			AccessKey:    os.Getenv("S3_ACCESS_KEY"),
			SecretKey:    os.Getenv("S3_SECRET_KEY"),
			UseSSL:       envBool("S3_USE_SSL", true, fail),
			CreateBucket: envBool("S3_CREATE_BUCKET", false, fail),
		},
	}

	if cfg.DatabaseURL == "" {
		fail("DATABASE_URL is required")
	}
	if len(cfg.AuthSecret) < minAuthSecretLen {
		fail("AUTH_SECRET must be at least %d characters (generate one with `openssl rand -hex 32`)", minAuthSecretLen)
	}
	if cfg.Mail.Host == "" {
		fail("SMTP_HOST is required")
	}
	if cfg.S3.Endpoint == "" || cfg.S3.AccessKey == "" || cfg.S3.SecretKey == "" {
		fail("S3_ENDPOINT, S3_ACCESS_KEY and S3_SECRET_KEY are required")
	}
	for _, p := range cfg.TrustedProxies {
		if _, err := netip.ParsePrefix(p); err != nil {
			fail("TRUSTED_PROXIES must be CIDR prefixes like 172.28.80.0/24, got %q", p)
		}
	}
	if u, err := url.Parse(cfg.PublicURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		fail("API_PUBLIC_URL must be an absolute http(s) URL")
	}
	if u, err := url.Parse(cfg.AppURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		fail("APP_PUBLIC_URL must be an absolute http(s) URL")
	}

	if len(errs) > 0 {
		return Config{}, fmt.Errorf("invalid configuration: %s", strings.Join(errs, "; "))
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool, fail func(string, ...any)) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		fail("%s must be true or false", key)
	}
	return b
}

func envInt(key string, fallback int, fail func(string, ...any)) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		fail("%s must be a number", key)
	}
	return n
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
