package server

import (
	"context"
	"net/http"
	"time"
)

// handleHealthz answers whether the process is up (liveness).
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, r, http.StatusOK, map[string]string{"status": "ok"})
}

// handleReadyz answers whether dependencies are usable (readiness).
func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.ready(ctx); err != nil {
		s.log.WarnContext(r.Context(), "not ready", "err", err)
		s.writeJSON(w, r, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
		return
	}
	s.writeJSON(w, r, http.StatusOK, map[string]string{"status": "ok"})
}
