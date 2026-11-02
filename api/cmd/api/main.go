package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	_ "time/tzdata" // embed IANA zones so slim images can validate user timezones

	"github.com/sorenhoang/sr-english/api/internal/auth"
	"github.com/sorenhoang/sr-english/api/internal/config"
	"github.com/sorenhoang/sr-english/api/internal/db"
	"github.com/sorenhoang/sr-english/api/internal/httpapi"
)

// Supabase puts this audience on every signed-in user's access token.
const supabaseAudience = "authenticated"

func main() {
	if err := run(); err != nil {
		slog.Error("api stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	connectCtx, cancel := context.WithTimeout(ctx, cfg.StartupTimeout)
	pool, err := db.Connect(connectCtx, cfg.DatabaseURL)
	cancel()
	if err != nil {
		return err
	}
	defer pool.Close()

	verifier, err := auth.NewVerifier(ctx, auth.VerifierConfig{
		JWKSURL: cfg.JWKSURL, Issuer: cfg.Issuer, Audience: supabaseAudience,
	})
	if err != nil {
		return err
	}
	deps := httpapi.Deps{Pool: pool, Verifier: verifier, AllowedEmails: cfg.AllowedEmails, Learning: cfg.Learning}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.NewRouter(deps),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
	}
	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "port", cfg.Port)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
