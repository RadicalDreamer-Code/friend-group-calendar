// Package testutil provides a throwaway Postgres database and fakes for tests.
package testutil

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/url"
	"os"
	"regexp"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/blob"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/database"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/mail"
)

// DB creates a fresh, migrated database for one test and drops it afterwards.
// It needs TEST_DATABASE_URL (any database on the server, used to create the
// test one); without it the test is skipped. `make test` sets it.
func DB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	adminURL := os.Getenv("TEST_DATABASE_URL")
	if adminURL == "" {
		t.Skip("TEST_DATABASE_URL not set; run `make test` for database tests")
	}
	ctx := context.Background()

	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatalf("connect to test server: %v", err)
	}
	t.Cleanup(func() { admin.Close(context.Background()) })

	buf := make([]byte, 6)
	_, _ = rand.Read(buf)
	name := "test_" + hex.EncodeToString(buf)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("create test database: %v", err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(context.Background(), "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("drop test database: %v", err)
		}
	})

	u, err := url.Parse(adminURL)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = "/" + name
	pool, err := database.Connect(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	return pool
}

// Mailbox records sent mail instead of delivering it.
type Mailbox struct {
	mu   sync.Mutex
	sent []mail.Message
}

func (m *Mailbox) Send(_ context.Context, msg mail.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, msg)
	return nil
}

func (m *Mailbox) Sent() []mail.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]mail.Message(nil), m.sent...)
}

var codePattern = regexp.MustCompile(`\b\d{6}\b`)

// LastCode returns the sign-in code from the latest email to the address.
func (m *Mailbox) LastCode(t *testing.T, to string) string {
	t.Helper()
	sent := m.Sent()
	for i := len(sent) - 1; i >= 0; i-- {
		if sent[i].To == to {
			if code := codePattern.FindString(sent[i].Text); code != "" {
				return code
			}
		}
	}
	t.Fatalf("no sign-in code was emailed to %s", to)
	return ""
}

// Blobs is an in-memory blob.Store.
type Blobs struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func NewBlobs() *Blobs { return &Blobs{objects: map[string][]byte{}} }

func (b *Blobs) Put(_ context.Context, key string, data []byte, _ string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.objects[key] = append([]byte(nil), data...)
	return nil
}

func (b *Blobs) Get(_ context.Context, key string) (io.ReadCloser, int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	data, ok := b.objects[key]
	if !ok {
		return nil, 0, blob.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(data)), int64(len(data)), nil
}

func (b *Blobs) Delete(_ context.Context, key string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.objects, key)
	return nil
}

func (b *Blobs) Keys() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	keys := make([]string, 0, len(b.objects))
	for k := range b.objects {
		keys = append(keys, k)
	}
	return keys
}
