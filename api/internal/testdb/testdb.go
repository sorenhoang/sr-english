// Package testdb gives each test its own freshly migrated Postgres schema.
package testdb

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sorenhoang/sr-english/api/internal/db"
)

// New creates schema t_<random>, runs all migrations into it and drops it when the test ends.
func New(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		base = db.DefaultTestURL
	}

	admin, err := db.Connect(ctx, base)
	if err != nil {
		t.Fatalf("testdb: connect: %v", err)
	}
	defer admin.Close()

	schema := "t_" + randomHex(t)
	if _, err := admin.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS "uuid-ossp" SCHEMA public`); err != nil {
		t.Fatalf("testdb: extension: %v", err)
	}
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("testdb: create schema: %v", err)
	}

	scoped := withSearchPath(t, base, schema)
	if err := db.Migrate(scoped); err != nil {
		t.Fatalf("testdb: migrate: %v", err)
	}
	pool, err := db.Connect(ctx, scoped)
	if err != nil {
		t.Fatalf("testdb: connect scoped: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		cleanup, err := db.Connect(context.Background(), base)
		if err != nil {
			t.Logf("testdb: cleanup connect: %v", err)
			return
		}
		defer cleanup.Close()
		if _, err := cleanup.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Logf("testdb: drop schema: %v", err)
		}
	})
	return pool
}

func withSearchPath(t *testing.T, base, schema string) string {
	t.Helper()
	u, err := url.Parse(base)
	if err != nil {
		t.Fatalf("testdb: parse url: %v", err)
	}
	q := u.Query()
	q.Set("search_path", schema+",public")
	u.RawQuery = q.Encode()
	return u.String()
}

func randomHex(t *testing.T) string {
	t.Helper()
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("testdb: random: %v", err)
	}
	return hex.EncodeToString(b)
}
