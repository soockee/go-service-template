// Package server is the HTTP layer: the http.Server, routes, middleware
// and handlers. Handlers are methods on Server and return an error; the
// adapters in respond.go turn errors into responses.
package server

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/soockee/go-service-template/internal/config"
	"github.com/soockee/go-service-template/internal/notes"
)

// Deps are the things handlers need. main builds them; tests pass fakes.
type Deps struct {
	Notes *notes.Service
	// Ready reports whether dependencies (e.g. the database) are usable.
	Ready func(context.Context) error
	// Assets is served under /assets/.
	Assets fs.FS
}

type Server struct {
	cfg    config.Config
	log    *slog.Logger
	notes  *notes.Service
	ready  func(context.Context) error
	assets fs.FS
}

func New(cfg config.Config, log *slog.Logger, deps Deps) *Server {
	ready := deps.Ready
	if ready == nil {
		ready = func(context.Context) error { return nil }
	}
	return &Server{
		cfg:    cfg,
		log:    log,
		notes:  deps.Notes,
		ready:  ready,
		assets: deps.Assets,
	}
}

// Handler returns the full handler: routes wrapped in middleware.
func (s *Server) Handler() http.Handler {
	return s.middleware(s.routes())
}

// Run serves on cfg.Addr until ctx is cancelled, then shuts down
// gracefully within cfg.ShutdownTimeout.
func (s *Server) Run(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.cfg.Addr)
	if err != nil {
		return err
	}
	return s.Serve(ctx, ln)
}

// Serve is Run on an existing listener.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	srv := &http.Server{
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		ErrorLog:          slog.NewLogLogger(s.log.Handler(), slog.LevelWarn),
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}

	errCh := make(chan error, 1)
	go func() {
		s.log.Info("listening", "addr", ln.Addr().String())
		errCh <- srv.Serve(ln)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	s.log.Info("shutting down", "timeout", s.cfg.ShutdownTimeout)
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
