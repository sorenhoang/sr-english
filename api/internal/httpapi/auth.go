package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/sorenhoang/sr-english/api/internal/user"
)

type ctxKey int

const userKey ctxKey = iota

// requireUser authenticates the bearer token, enforces the email allowlist and loads the user.
func requireUser(d Deps, users user.Repository) func(http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, e := range d.AllowedEmails {
		allowed[strings.ToLower(strings.TrimSpace(e))] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || token == "" || d.Verifier == nil {
				writeError(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
				return
			}
			id, err := d.Verifier.Verify(token)
			if err != nil {
				slog.Info("rejected token", "err", err)
				writeError(w, http.StatusUnauthorized, "unauthorized", "invalid token")
				return
			}
			if !allowed[strings.ToLower(id.Email)] {
				writeError(w, http.StatusForbidden, "forbidden", "account is not allowed")
				return
			}
			u, err := users.Upsert(r.Context(), id.Sub, id.Email)
			if err != nil {
				slog.Error("upsert user", "err", err)
				writeError(w, http.StatusInternalServerError, "internal", "internal server error")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey, u)))
		})
	}
}

func currentUser(r *http.Request) user.User {
	u, _ := r.Context().Value(userKey).(user.User)
	return u
}

// writeError follows the frontend's expected shape: {"error": {"code", "message"}}.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
