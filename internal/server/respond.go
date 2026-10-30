package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/soockee/go-service-template/internal/notes"
	"github.com/soockee/go-service-template/internal/views"
)

// handlerFunc is the handler shape used throughout: do the work, write
// the response on success, return an error otherwise. Return errors
// before writing anything; once the body has started, the status is sent.
type handlerFunc func(w http.ResponseWriter, r *http.Request) error

// Error is an error with an HTTP status and a message that is safe to
// show to clients.
type Error struct {
	Status int
	Msg    string
	Err    error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%d %s: %v", e.Status, e.Msg, e.Err)
	}
	return fmt.Sprintf("%d %s", e.Status, e.Msg)
}

func (e *Error) Unwrap() error { return e.Err }

func httpError(status int, msg string) *Error {
	return &Error{Status: status, Msg: msg}
}

// toHTTPError maps any error to a status and client-safe message. Domain
// sentinel errors are mapped here, in one place; anything unknown is a
// 500 whose details stay in the log.
func toHTTPError(err error) *Error {
	var he *Error
	switch {
	case errors.As(err, &he):
		return he
	case errors.Is(err, notes.ErrNotFound):
		return &Error{Status: http.StatusNotFound, Msg: err.Error(), Err: err}
	case errors.Is(err, notes.ErrInvalid):
		return &Error{Status: http.StatusBadRequest, Msg: err.Error(), Err: err}
	default:
		return &Error{Status: http.StatusInternalServerError, Msg: "internal server error", Err: err}
	}
}

func (s *Server) logError(r *http.Request, he *Error) {
	if he.Status >= http.StatusInternalServerError {
		s.log.ErrorContext(r.Context(), "request failed", "status", he.Status, "err", he.Err)
	}
}

// api adapts a handlerFunc for JSON endpoints.
func (s *Server) api(h handlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			he := toHTTPError(err)
			s.logError(r, he)
			s.writeJSON(w, r, he.Status, map[string]string{"error": he.Msg})
		}
	})
}

// page adapts a handlerFunc for HTML pages.
func (s *Server) page(h handlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			he := toHTTPError(err)
			s.logError(r, he)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(he.Status)
			if err := views.ErrorPage(he.Status, he.Msg).Render(r.Context(), w); err != nil {
				s.log.ErrorContext(r.Context(), "render error page", "err", err)
			}
		}
	})
}

func (s *Server) writeJSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		s.log.ErrorContext(r.Context(), "write json", "err", err)
	}
}

const maxBodyBytes = 1 << 20

// decodeJSON reads a JSON request body into v, rejecting unknown fields
// and bodies over 1 MiB.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return &Error{Status: http.StatusBadRequest, Msg: "invalid JSON body", Err: err}
	}
	return nil
}

func notFound(http.ResponseWriter, *http.Request) error {
	return httpError(http.StatusNotFound, "not found")
}
