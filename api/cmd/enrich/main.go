// Command enrich fills meanings for senses of one CEFR level. Run by hand; it costs LLM tokens.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"

	"github.com/sorenhoang/sr-english/api/internal/db"
	"github.com/sorenhoang/sr-english/api/internal/enrich"
	"github.com/sorenhoang/sr-english/api/internal/llm"
)

const (
	defaultModel     = "claude-sonnet-5-5"
	defaultMaxTokens = 600
	defaultLimit     = 50
)

func main() {
	level := flag.String("level", "A1", "CEFR level to enrich (A1–C2)")
	limit := flag.Int("limit", defaultLimit, "maximum senses to enrich in this run")
	model := flag.String("model", defaultModel, "Anthropic model id")
	flag.Parse()
	if err := run(*level, *limit, *model); err != nil {
		slog.Error("enrich failed", "err", err)
		os.Exit(1)
	}
}

func run(level string, limit int, model string) error {
	url, key := os.Getenv("DATABASE_URL"), os.Getenv("ANTHROPIC_API_KEY")
	if url == "" || key == "" {
		return errors.New("DATABASE_URL and ANTHROPIC_API_KEY are required")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		return err
	}
	defer pool.Close()

	res, err := enrich.Run(ctx, pool, llm.NewAnthropic(key, model, defaultMaxTokens), level, limit)
	if err != nil {
		return err
	}
	slog.Info("enriched", "level", level, "updated", res.Updated, "failed", res.Failed)
	return nil
}
