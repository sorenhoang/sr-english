package httpapi_test

import (
	"net/http"
	"testing"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"

	"github.com/sorenhoang/sr-english/api/internal/apitest"
)

func TestSaveWordShouldGiveEachUserTheirOwnCustomSense(t *testing.T) {
	env := apitest.New(t)
	tokenA := env.Token(t, "sub-a", apitest.AllowedEmail)
	tokenB := env.Token(t, "sub-b", apitest.OtherEmail)
	env.UserID(t, tokenA)
	env.UserID(t, tokenB)
	passage, _ := passageWithWords(t, env)

	_, a1 := saveWord(t, env, tokenA, passage, 5)
	_, a2 := saveWord(t, env, tokenA, passage, 5)
	_, b := saveWord(t, env, tokenB, passage, 5)
	if a1.SenseID == "" || a2.SenseID != a1.SenseID || a2.CardID != a1.CardID {
		t.Fatalf("A saved twice: %+v then %+v, want the same sense and card", a1, a2)
	}
	if b.SenseID == a1.SenseID || b.CardID == a1.CardID {
		t.Fatalf("B got A's custom sense/card (%+v vs %+v), want separate ones", b, a1)
	}
	if status, _ := review(t, env, tokenB, a1.SenseID, int(fsrs.Good)); status != http.StatusNotFound {
		t.Fatalf("B reviewing A's custom sense: status = %d, want 404", status)
	}
}

func TestPassageShouldShowAWordAsSavedRightAfterSavingIt(t *testing.T) {
	env := apitest.New(t)
	tokenA := env.Token(t, "sub-a", apitest.AllowedEmail)
	tokenB := env.Token(t, "sub-b", apitest.OtherEmail)
	env.UserID(t, tokenA)
	env.UserID(t, tokenB)
	passage, _ := passageWithWords(t, env)
	if status, _ := saveWord(t, env, tokenA, passage, 4); status != http.StatusOK {
		t.Fatalf("save status = %d", status)
	}

	savedFlag := func(token string) bool {
		status, raw := env.Do(t, http.MethodGet, "/api/v1/passage?id="+passage, token, nil)
		if status != http.StatusOK {
			t.Fatalf("passage status = %d", status)
		}
		var p struct {
			Tokens []struct {
				Text  string `json:"text"`
				Saved bool   `json:"saved"`
			} `json:"tokens"`
		}
		apitest.Decode(t, raw, &p)
		return p.Tokens[4].Saved
	}
	if !savedFlag(tokenA) {
		t.Errorf("user A: 'mitigated' not marked saved after save-word")
	}
	if savedFlag(tokenB) {
		t.Errorf("user B: 'mitigated' marked saved, want only the saver")
	}
}

func TestSessionTodayShouldNotQueueSavedWordsThatHaveNoMeaningYet(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	env.UserID(t, token)
	passage, _ := passageWithWords(t, env)
	if status, _ := saveWord(t, env, token, passage, 5); status != http.StatusOK {
		t.Fatalf("save custom word status = %d", status)
	}
	if status, _ := saveWord(t, env, token, passage, 4); status != http.StatusOK {
		t.Fatalf("save listed word status = %d", status)
	}

	review := today(t, env, token).task(t, "vocab_review")
	if len(review) != 1 || review[0].Headword != "mitigate" {
		t.Fatalf("vocab_review = %+v, want only 'mitigate' (custom 'kubernetes' has no meaning to quiz yet)", review)
	}
}
