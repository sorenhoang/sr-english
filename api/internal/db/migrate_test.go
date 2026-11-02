package db_test

import (
	"context"
	"testing"

	"github.com/sorenhoang/sr-english/api/internal/testdb"
)

func TestMigrateShouldCreateCoreTablesWithStandardColumns(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()

	for _, table := range []string{"users", "user_settings", "senses", "cards", "reviews"} {
		rows, err := pool.Query(ctx, `
			SELECT column_name, data_type, COALESCE(column_default, '')
			FROM information_schema.columns
			WHERE table_schema = current_schema() AND table_name = $1`, table)
		if err != nil {
			t.Fatalf("%s: query columns: %v", table, err)
		}
		cols := map[string][2]string{}
		for rows.Next() {
			var name, typ, def string
			if err := rows.Scan(&name, &typ, &def); err != nil {
				t.Fatalf("%s: scan: %v", table, err)
			}
			cols[name] = [2]string{typ, def}
		}
		rows.Close()

		if len(cols) == 0 {
			t.Fatalf("table %s does not exist", table)
		}
		if cols["id"][0] != "uuid" || cols["id"][1] != "uuid_generate_v4()" {
			t.Errorf("%s.id = %v, want uuid default uuid_generate_v4()", table, cols["id"])
		}
		for _, ts := range []string{"created_at", "updated_at", "deleted_at"} {
			if cols[ts][0] != "timestamp with time zone" {
				t.Errorf("%s.%s type = %q, want timestamp with time zone", table, ts, cols[ts][0])
			}
		}
	}

	var fks int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM information_schema.table_constraints
		WHERE table_schema = current_schema() AND constraint_type = 'FOREIGN KEY'`).Scan(&fks); err != nil {
		t.Fatalf("count foreign keys: %v", err)
	}
	if fks != 0 {
		t.Errorf("foreign keys = %d, want 0", fks)
	}
}
