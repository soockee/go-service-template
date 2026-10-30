// Package config loads service configuration from environment variables.
// Every setting lives in Config; nothing else in the service reads the
// environment.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

type Config struct {
	// Addr is the listen address, e.g. ":8080". Env: ADDR.
	Addr string
	// Env is "dev" or "prod". Env: APP_ENV.
	Env string
	// LogLevel is debug, info, warn or error. Env: LOG_LEVEL.
	LogLevel slog.Level
	// LogFormat is "text" or "json"; defaults to text in dev, json in prod. Env: LOG_FORMAT.
	LogFormat string
	// DatabaseURL is a Postgres URL. Empty means an in-memory store. Env: DATABASE_URL.
	DatabaseURL string
	// CORSOrigins lists allowed origins; empty disables CORS. Env: CORS_ALLOWED_ORIGINS (comma-separated).
	CORSOrigins []string
	// ShutdownTimeout bounds graceful shutdown. Env: SHUTDOWN_TIMEOUT.
	ShutdownTimeout time.Duration
}

// Load reads the configuration from the process environment.
func Load() (Config, error) {
	return LoadFrom(os.Getenv)
}

// LoadFrom reads the configuration through getenv, so tests can supply
// their own environment.
func LoadFrom(getenv func(string) string) (Config, error) {
	get := func(key, def string) string {
		if v := strings.TrimSpace(getenv(key)); v != "" {
			return v
		}
		return def
	}

	cfg := Config{
		Addr:        get("ADDR", ":8080"),
		Env:         get("APP_ENV", "dev"),
		DatabaseURL: get("DATABASE_URL", ""),
	}
	var errs []error

	if cfg.Env != "dev" && cfg.Env != "prod" {
		errs = append(errs, fmt.Errorf("APP_ENV: want dev or prod, got %q", cfg.Env))
	}

	if err := cfg.LogLevel.UnmarshalText([]byte(get("LOG_LEVEL", "info"))); err != nil {
		errs = append(errs, fmt.Errorf("LOG_LEVEL: %w", err))
	}

	defFormat := "text"
	if cfg.Env == "prod" {
		defFormat = "json"
	}
	cfg.LogFormat = get("LOG_FORMAT", defFormat)
	if cfg.LogFormat != "text" && cfg.LogFormat != "json" {
		errs = append(errs, fmt.Errorf("LOG_FORMAT: want text or json, got %q", cfg.LogFormat))
	}

	for _, o := range strings.Split(get("CORS_ALLOWED_ORIGINS", ""), ",") {
		if o = strings.TrimSpace(o); o != "" {
			cfg.CORSOrigins = append(cfg.CORSOrigins, o)
		}
	}

	d, err := time.ParseDuration(get("SHUTDOWN_TIMEOUT", "10s"))
	if err != nil {
		errs = append(errs, fmt.Errorf("SHUTDOWN_TIMEOUT: %w", err))
	}
	cfg.ShutdownTimeout = d

	if cfg.Env == "prod" && cfg.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL: required when APP_ENV=prod"))
	}

	return cfg, errors.Join(errs...)
}
