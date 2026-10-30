package server

import (
	"bytes"
	"net/http"

	"github.com/a-h/templ"
)

// render writes a templ component. It renders into a buffer first so a
// template error still becomes a proper error response.
func render(w http.ResponseWriter, r *http.Request, status int, c templ.Component) error {
	var buf bytes.Buffer
	if err := c.Render(r.Context(), &buf); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, err := buf.WriteTo(w)
	return err
}
