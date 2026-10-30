package server

import "net/http"

// routes is the one place that lists every endpoint. Patterns use the
// stdlib ServeMux syntax: "METHOD /path/{param}".
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /readyz", s.handleReadyz)
	if s.assets != nil {
		mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServerFS(s.assets)))
	}

	// JSON API. Errors render as {"error": "..."}.
	mux.Handle("GET /api/notes", s.api(s.handleListNotes))
	mux.Handle("POST /api/notes", s.api(s.handleCreateNote))
	mux.Handle("GET /api/notes/{id}", s.api(s.handleGetNote))
	// Catch-alls are GET-only so wrong methods on real routes still get 405.
	mux.Handle("GET /api/", s.api(notFound))

	// HTML pages (templ + htmx). Errors render as an error page.
	mux.Handle("GET /{$}", s.page(s.handleHomePage))
	mux.Handle("POST /notes", s.page(s.handleCreateNotePage))
	mux.Handle("GET /", s.page(notFound))

	return mux
}
