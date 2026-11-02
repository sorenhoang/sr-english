package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/sorenhoang/sr-english/api/internal/db"
)

func TestConnectShouldReachPostgresWhenTestDatabaseURLIsSet(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = db.DefaultTestURL
	}
	ctx := context.Background()

	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer pool.Close()

	var one int
	if err := pool.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
		t.Fatalf("SELECT 1: %v", err)
	}
	if one != 1 {
		t.Fatalf("SELECT 1 = %d, want 1", one)
	}
}
