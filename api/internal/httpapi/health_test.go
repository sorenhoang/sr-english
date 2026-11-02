package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sorenhoang/sr-english/api/internal/httpapi"
)

func TestHealthShouldReturnOKWhenServerIsUp(t *testing.T) {
	srv := httptest.NewServer(httpapi.NewRouter(httpapi.Deps{}))
	defer srv.Close()

	res, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf(`body["status"] = %q, want "ok"`, body["status"])
	}
}
