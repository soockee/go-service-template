// Command server is the service entrypoint. It only wires things
// together: config → logger → store → server.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/soockee/go-service-template/assets"
	"github.com/soockee/go-service-template/internal/config"
	"github.com/soockee/go-service-template/internal/notes"
	"github.com/soockee/go-service-template/internal/server"
	"github.com/soockee/go-service-template/internal/store"
)

// version is set at build time: -ldflags "-X main.version=v1.2.3".
var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	log := server.NewLogger(os.Stderr, cfg)
	log.Info("starting", "version", version, "env", cfg.Env)

	deps := server.Deps{Assets: assets.FS}

	var noteStore notes.Store
	if cfg.DatabaseURL == "" {
		log.Warn("DATABASE_URL not set; using in-memory store")
		noteStore = notes.NewMemStore()
	} else {
		pool, err := store.Open(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
		if err := store.Migrate(ctx, pool, log); err != nil {
			return err
		}
		noteStore = store.NewNotes(pool)
		deps.Ready = pool.Ping
	}
	deps.Notes = notes.NewService(noteStore)

	return server.New(cfg, log, deps).Run(ctx)
}
