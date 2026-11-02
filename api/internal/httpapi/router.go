package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sorenhoang/sr-english/api/internal/auth"
	"github.com/sorenhoang/sr-english/api/internal/config"
	"github.com/sorenhoang/sr-english/api/internal/content"
	"github.com/sorenhoang/sr-english/api/internal/settings"
	"github.com/sorenhoang/sr-english/api/internal/stats"
	"github.com/sorenhoang/sr-english/api/internal/user"
	"github.com/sorenhoang/sr-english/api/internal/vocab"
)

// Deps holds everything handlers need.
type Deps struct {
	Pool          *pgxpool.Pool
	Verifier      *auth.Verifier
	AllowedEmails []string
	Learning      config.Learning
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	users := user.NewRepository(d.Pool)
	r.Route("/api/v1", func(api chi.Router) {
		api.Use(requireUser(d, users))
		api.Get("/me", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, currentUser(r))
		})
		settingsRepo := settings.NewRepository(d.Pool, d.Learning)
		vocabRepo := vocab.NewRepository(d.Pool)
		settingsHandler{repo: settingsRepo, learning: d.Learning}.register(api)
		contentRepo := content.NewRepository(d.Pool)
		sessionHandler{settings: settingsRepo, vocab: vocabRepo, content: contentRepo, learning: d.Learning}.register(api)
		passagesHandler{content: contentRepo}.register(api)
		wordsHandler{vocab: vocabRepo}.register(api)
		statsHandler{settings: settingsRepo, stats: stats.NewRepository(d.Pool)}.register(api)
		reviewHandler{vocab: vocabRepo, scheduler: vocab.NewScheduler(d.Learning.DesiredRetention)}.register(api)
	})
	return r
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode response", "err", err)
	}
}
