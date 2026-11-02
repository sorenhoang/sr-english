package seed_test

import (
	"context"
	"strings"
	"testing"

	"github.com/sorenhoang/sr-english/api/internal/seed"
	"github.com/sorenhoang/sr-english/api/internal/testdb"
)

func TestLoadShouldSkipRowsWhoseHeadwordIsOnlySeparators(t *testing.T) {
	pool := testdb.New(t)
	csv := `headword,pos,CEFR
/,noun,A1
 / ,verb,A2
abandon,verb,B1
`
	res, err := seed.Load(context.Background(), pool, strings.NewReader(csv), "cefrj")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if res.Loaded != 1 || res.Skipped != 2 {
		t.Fatalf("result = %+v, want Loaded 1 Skipped 2", res)
	}
	if got := countSenses(t, pool); got != 1 {
		t.Fatalf("senses = %d, want 1", got)
	}
}
