package enrich_test

import (
	"context"
	"testing"

	"github.com/sorenhoang/sr-english/api/internal/enrich"
	"github.com/sorenhoang/sr-english/api/internal/testdb"
)

func TestRunShouldOnlyAskForActiveSensesOfTheRequestedLevel(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	insertSense(t, pool, "alpha", "A2", nil)
	insertSense(t, pool, "beta", "B1", nil)
	insertSense(t, pool, "gamma", "B1", nil)
	if _, err := pool.Exec(ctx, `UPDATE senses SET deleted_at = now() WHERE headword = 'gamma'`); err != nil {
		t.Fatalf("soft delete gamma: %v", err)
	}
	llm := &fakeLLM{answers: map[string]string{
		"alpha": validAnswer("alpha"), "beta": validAnswer("beta"), "gamma": validAnswer("gamma"),
	}}

	res, err := enrich.Run(ctx, pool, llm, "B1", 100)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Updated != 1 || len(llm.asked) != 1 || llm.asked[0] != "beta" {
		t.Fatalf("result %+v asked %v, want only beta (A2 and soft-deleted senses excluded)", res, llm.asked)
	}
}

func TestRunShouldResetReviewedWhenItRewritesASense(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	insertSense(t, pool, "delta", "B1", nil)
	if _, err := pool.Exec(ctx, `UPDATE senses SET reviewed = TRUE WHERE headword = 'delta'`); err != nil {
		t.Fatalf("mark reviewed: %v", err)
	}
	llm := &fakeLLM{answers: map[string]string{"delta": validAnswer("delta")}}

	if _, err := enrich.Run(ctx, pool, llm, "B1", 10); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := loadSense(t, pool, "delta"); got.reviewed {
		t.Fatalf("reviewed = true after machine rewrite, want false")
	}
}

func TestRunShouldRejectNonPositiveLimit(t *testing.T) {
	pool := testdb.New(t)
	if _, err := enrich.Run(context.Background(), pool, &fakeLLM{}, "B1", 0); err == nil {
		t.Fatalf("Run with limit 0 returned nil error, want an error")
	}
}
