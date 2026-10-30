package server

import (
	"net/http"
	"strconv"

	"github.com/soockee/go-service-template/internal/views"
)

// --- JSON API --------------------------------------------------------

func (s *Server) handleListNotes(w http.ResponseWriter, r *http.Request) error {
	list, err := s.notes.List(r.Context())
	if err != nil {
		return err
	}
	s.writeJSON(w, r, http.StatusOK, list)
	return nil
}

type createNoteRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func (s *Server) handleCreateNote(w http.ResponseWriter, r *http.Request) error {
	var req createNoteRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return err
	}
	n, err := s.notes.Create(r.Context(), req.Title, req.Body)
	if err != nil {
		return err
	}
	w.Header().Set("Location", "/api/notes/"+strconv.FormatInt(n.ID, 10))
	s.writeJSON(w, r, http.StatusCreated, n)
	return nil
}

func (s *Server) handleGetNote(w http.ResponseWriter, r *http.Request) error {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return httpError(http.StatusBadRequest, "id must be an integer")
	}
	n, err := s.notes.Get(r.Context(), id)
	if err != nil {
		return err
	}
	s.writeJSON(w, r, http.StatusOK, n)
	return nil
}

// --- HTML pages ------------------------------------------------------

func (s *Server) handleHomePage(w http.ResponseWriter, r *http.Request) error {
	list, err := s.notes.List(r.Context())
	if err != nil {
		return err
	}
	return render(w, r, http.StatusOK, views.Home(list))
}

// handleCreateNotePage serves the form on the home page. With htmx it
// returns just the new list item; without JavaScript it redirects back.
func (s *Server) handleCreateNotePage(w http.ResponseWriter, r *http.Request) error {
	n, err := s.notes.Create(r.Context(), r.FormValue("title"), r.FormValue("body"))
	if err != nil {
		return err
	}
	if r.Header.Get("HX-Request") == "true" {
		return render(w, r, http.StatusCreated, views.NoteItem(n))
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}
