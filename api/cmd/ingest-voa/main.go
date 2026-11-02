// Command ingest-voa fetches VOA Learning English article URLs (one per line) and stores them as passages.
// Content is public domain; the passage keeps its source URL and the VOA credit line.
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sorenhoang/sr-english/api/internal/config"
	"github.com/sorenhoang/sr-english/api/internal/content"
	"github.com/sorenhoang/sr-english/api/internal/db"
)

const (
	fetchTimeout = 20 * time.Second
	maxPageBytes = 5 << 20 // VOA article pages are ~100 KB
	politeDelay  = time.Second
)

func main() {
	urlsFile := flag.String("urls", "", "file with one VOA article URL per line")
	flag.Parse()
	if err := run(*urlsFile); err != nil {
		slog.Error("ingest failed", "err", err)
		os.Exit(1)
	}
}

func run(urlsFile string) error {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" || urlsFile == "" {
		return errors.New("DATABASE_URL and --urls are required")
	}
	urls, err := readURLs(urlsFile)
	if err != nil {
		return err
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	ing, err := content.NewIngester(ctx, pool, config.DefaultCoverage)
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: fetchTimeout,
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			if !strings.HasPrefix(req.URL.String(), allowedPrefix) {
				return fmt.Errorf("redirect outside %s refused", allowedPrefix)
			}
			return nil
		},
	}
	var failed int
	for i, u := range urls {
		if i > 0 {
			time.Sleep(politeDelay)
		}
		page, err := fetch(ctx, client, u)
		if err == nil {
			_, err = ing.Ingest(ctx, u, page)
		}
		if err != nil {
			failed++
			slog.Warn("skip url", "url", u, "err", err)
			continue
		}
		slog.Info("ingested", "url", u)
	}
	slog.Info("done", "total", len(urls), "failed", failed)
	if failed > 0 {
		return fmt.Errorf("%d of %d urls failed", failed, len(urls))
	}
	return nil
}

func readURLs(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open urls: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			slog.Warn("close urls file", "err", err)
		}
	}()
	var urls []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" && !strings.HasPrefix(line, "#") {
			urls = append(urls, line)
		}
	}
	return urls, sc.Err()
}

// allowedPrefix keeps the tool to VOA Learning English pages only.
const allowedPrefix = "https://learningenglish.voanews.com/"

func fetch(ctx context.Context, client *http.Client, url string) (string, error) {
	if !strings.HasPrefix(url, allowedPrefix) {
		return "", fmt.Errorf("only %s URLs are allowed", allowedPrefix)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			slog.Warn("close response", "err", err)
		}
	}()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, maxPageBytes+1))
	if err != nil {
		return "", err
	}
	if len(body) > maxPageBytes {
		return "", fmt.Errorf("page larger than %d bytes", maxPageBytes)
	}
	return string(body), nil
}
