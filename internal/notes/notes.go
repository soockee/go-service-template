// Package notes is the example feature: the domain types, the rules that
// apply to them, and the Store interface this feature needs. Replace it
// with your own feature packages; keep the shape.
package notes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrNotFound = errors.New("note not found")
	ErrInvalid  = errors.New("invalid note")
)

type Note struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// Store is defined here, by the consumer, and implemented in
// internal/store (Postgres) and MemStore (tests, local runs).
type Store interface {
	CreateNote(ctx context.Context, title, body string) (Note, error)
	GetNote(ctx context.Context, id int64) (Note, error)
	ListNotes(ctx context.Context) ([]Note, error)
}

const maxTitle = 200

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create validates the input and stores a new note. Validation failures
// wrap ErrInvalid.
func (s *Service) Create(ctx context.Context, title, body string) (Note, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Note{}, fmt.Errorf("%w: title is required", ErrInvalid)
	}
	if len(title) > maxTitle {
		return Note{}, fmt.Errorf("%w: title longer than %d characters", ErrInvalid, maxTitle)
	}
	return s.store.CreateNote(ctx, title, strings.TrimSpace(body))
}

func (s *Service) Get(ctx context.Context, id int64) (Note, error) {
	return s.store.GetNote(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]Note, error) {
	return s.store.ListNotes(ctx)
}
