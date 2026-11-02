package seed_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sorenhoang/sr-english/api/internal/seed"
	"github.com/sorenhoang/sr-english/api/internal/testdb"
)

const validCSV = `headword,pos,CEFR,CoreInventory 1,CoreInventory 2,Threshold
abandon,verb,B1,,,
adviser/advisor,noun,B2,,,
run,verb,A1,,,
run,noun,A2,,,
`

func countSenses(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM senses WHERE deleted_at IS NULL`).Scan(&n); err != nil {
		t.Fatalf("count senses: %v", err)
	}
	return n
}

func TestLoadShouldCreateOneSensePerValidRowWhenCSVIsValid(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()

	res, err := seed.Load(ctx, pool, strings.NewReader(validCSV), "cefrj")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if res.Loaded != 4 || res.Skipped != 0 {
		t.Fatalf("result = %+v, want Loaded 4 Skipped 0", res)
	}
	if got := countSenses(t, pool); got != 4 {
		t.Fatalf("senses = %d, want 4", got)
	}

	var headword, level string
	var variants []string
	if err := pool.QueryRow(ctx, `SELECT headword, cefr_level, variants FROM senses WHERE pos = 'noun' AND headword = 'adviser'`).
		Scan(&headword, &level, &variants); err != nil {
		t.Fatalf("query adviser: %v", err)
	}
	if level != "B2" || strings.Join(variants, ",") != "adviser,advisor" {
		t.Fatalf("adviser = level %q variants %v, want B2 [adviser advisor]", level, variants)
	}

	var runs int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM senses WHERE headword = 'run'`).Scan(&runs); err != nil {
		t.Fatalf("count run: %v", err)
	}
	if runs != 2 {
		t.Fatalf("run senses = %d, want 2 (verb + noun)", runs)
	}
}

func TestLoadShouldBeIdempotentAndKeepCardsWhenRunTwice(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()

	if _, err := seed.Load(ctx, pool, strings.NewReader(validCSV), "cefrj"); err != nil {
		t.Fatalf("first Load: %v", err)
	}
	var senseID, cardID string
	if err := pool.QueryRow(ctx, `SELECT id FROM senses WHERE headword = 'abandon'`).Scan(&senseID); err != nil {
		t.Fatalf("query sense: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO cards (user_id, sense_id, due, stability, difficulty, reps)
		VALUES (uuid_generate_v4(), $1, now(), 3.5, 5.0, 2) RETURNING id`, senseID).Scan(&cardID); err != nil {
		t.Fatalf("insert card: %v", err)
	}

	if _, err := seed.Load(ctx, pool, strings.NewReader(validCSV), "cefrj"); err != nil {
		t.Fatalf("second Load: %v", err)
	}

	if got := countSenses(t, pool); got != 4 {
		t.Fatalf("senses after reseed = %d, want 4", got)
	}
	var cardSense string
	var stability float64
	var reps int
	var deleted *string
	if err := pool.QueryRow(ctx, `SELECT sense_id, stability, reps, deleted_at::text FROM cards WHERE id = $1`, cardID).
		Scan(&cardSense, &stability, &reps, &deleted); err != nil {
		t.Fatalf("query card: %v", err)
	}
	if cardSense != senseID || stability != 3.5 || reps != 2 || deleted != nil {
		t.Fatalf("card after reseed = sense %s stability %v reps %d deleted %v; want unchanged", cardSense, stability, reps, deleted)
	}
}

func TestLoadShouldSkipInvalidRowsAndLoadTheRest(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	csv := `headword,pos,CEFR
abandon,verb,B1
,noun,A1
ghost,noun,Z9
timid,adjective,C1
`
	res, err := seed.Load(ctx, pool, strings.NewReader(csv), "octanove")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if res.Loaded != 2 || res.Skipped != 2 {
		t.Fatalf("result = %+v, want Loaded 2 Skipped 2", res)
	}
	if got := countSenses(t, pool); got != 2 {
		t.Fatalf("senses = %d, want 2", got)
	}
}
