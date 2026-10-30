package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/soockee/go-service-template/internal/notes"
)

// Notes implements notes.Store. Column order in each SELECT matches the
// field order of notes.Note, so rows scan with RowToStructByPos.
type Notes struct {
	pool *pgxpool.Pool
}

var _ notes.Store = (*Notes)(nil)

func NewNotes(pool *pgxpool.Pool) *Notes {
	return &Notes{pool: pool}
}

const noteColumns = `id, title, body, created_at`

func (s *Notes) CreateNote(ctx context.Context, title, body string) (notes.Note, error) {
	rows, _ := s.pool.Query(ctx,
		`INSERT INTO notes (title, body) VALUES ($1, $2) RETURNING `+noteColumns,
		title, body)
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[notes.Note])
}

func (s *Notes) GetNote(ctx context.Context, id int64) (notes.Note, error) {
	rows, _ := s.pool.Query(ctx, `SELECT `+noteColumns+` FROM notes WHERE id = $1`, id)
	n, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[notes.Note])
	if errors.Is(err, pgx.ErrNoRows) {
		return notes.Note{}, notes.ErrNotFound
	}
	return n, err
}

func (s *Notes) ListNotes(ctx context.Context) ([]notes.Note, error) {
	rows, _ := s.pool.Query(ctx, `SELECT `+noteColumns+` FROM notes ORDER BY id DESC`)
	return pgx.CollectRows(rows, pgx.RowToStructByPos[notes.Note])
}
