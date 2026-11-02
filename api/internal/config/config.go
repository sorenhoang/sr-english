package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPort              = "8080"
	defaultStartupTimeout    = 10 * time.Second
	defaultReadHeaderTimeout = 5 * time.Second
	defaultShutdownTimeout   = 10 * time.Second
)

type Config struct {
	Port              string
	DatabaseURL       string
	JWKSURL           string
	Issuer            string
	AllowedEmails     []string
	Learning          Learning
	StartupTimeout    time.Duration
	ReadHeaderTimeout time.Duration
	ShutdownTimeout   time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Port:        getenv("PORT", defaultPort),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	// Supabase: JWKS at <project>/auth/v1/.well-known/jwks.json, issuer <project>/auth/v1.
	cfg.JWKSURL, cfg.Issuer = os.Getenv("SUPABASE_JWKS_URL"), os.Getenv("SUPABASE_ISSUER")
	if cfg.JWKSURL == "" || cfg.Issuer == "" {
		return Config{}, errors.New("SUPABASE_JWKS_URL and SUPABASE_ISSUER are required")
	}
	for _, e := range strings.Split(os.Getenv("ALLOWED_EMAILS"), ",") {
		if e = strings.TrimSpace(e); e != "" {
			cfg.AllowedEmails = append(cfg.AllowedEmails, e)
		}
	}
	if len(cfg.AllowedEmails) == 0 {
		return Config{}, errors.New("ALLOWED_EMAILS is required (comma-separated)")
	}
	if _, err := strconv.Atoi(cfg.Port); err != nil {
		return Config{}, fmt.Errorf("PORT must be a number: %w", err)
	}
	var err error
	if cfg.StartupTimeout, err = duration("STARTUP_TIMEOUT", defaultStartupTimeout); err != nil {
		return Config{}, err
	}
	if cfg.ReadHeaderTimeout, err = duration("READ_HEADER_TIMEOUT", defaultReadHeaderTimeout); err != nil {
		return Config{}, err
	}
	if cfg.ShutdownTimeout, err = duration("SHUTDOWN_TIMEOUT", defaultShutdownTimeout); err != nil {
		return Config{}, err
	}
	if cfg.Learning, err = loadLearning(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func duration(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration like 5s: %w", key, err)
	}
	return d, nil
}
