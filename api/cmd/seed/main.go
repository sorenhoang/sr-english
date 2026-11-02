// Command seed loads the bundled CEFR word lists into the database.
//
// Data: CEFR-J Vocabulary Profile 1.5 (Tono Laboratory, TUFS) and Octanove Vocabulary Profile C1/C2 (CC BY-SA 4.0).
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/sorenhoang/sr-english/api/internal/db"
	"github.com/sorenhoang/sr-english/api/internal/seed"
)

var sources = []struct{ file, name string }{
	{"cefrj-vocabulary-profile-1.5.csv", "cefrj"},
	{"octanove-vocabulary-profile-c1c2-1.0.csv", "octanove"},
}

func main() {
	dataDir := flag.String("data", "data", "directory holding the CEFR csv files")
	flag.Parse()
	if err := run(*dataDir); err != nil {
		slog.Error("seed failed", "err", err)
		os.Exit(1)
	}
}

func run(dataDir string) error {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if err := db.Migrate(url); err != nil {
		return err
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		return err
	}
	defer pool.Close()

	for _, s := range sources {
		f, err := os.Open(filepath.Join(dataDir, s.file))
		if err != nil {
			return fmt.Errorf("open %s: %w", s.file, err)
		}
		res, err := seed.Load(ctx, pool, f, s.name)
		if closeErr := f.Close(); closeErr != nil {
			slog.Warn("close csv", "file", s.file, "err", closeErr)
		}
		if err != nil {
			return err
		}
		slog.Info("seeded", "source", s.name, "loaded", res.Loaded, "skipped", res.Skipped)
	}
	return nil
}
