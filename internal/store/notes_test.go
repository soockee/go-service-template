package store

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/soockee/go-service-template/internal/notes"
)

// Store tests run against a real Postgres. Set TEST_DATABASE_URL (CI
// does; locally: docker compose up -d db). Without it they are skipped.
func openTestDB(t *testing.T) *Notes {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := Migrate(ctx, pool, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE notes`); err != nil {
		t.Fatal(err)
	}
	return NewNotes(pool)
}

func TestNotesRoundTrip(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()

	a, err := s.CreateNote(ctx, "first", "body")
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.CreateNote(ctx, "second", "")
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.GetNote(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "first" || got.Body != "body" || got.CreatedAt.IsZero() {
		t.Errorf("GetNote = %+v", got)
	}

	list, err := s.ListNotes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != b.ID {
		t.Errorf("ListNotes = %+v, want newest first", list)
	}

	if _, err := s.GetNote(ctx, -1); !errors.Is(err, notes.ErrNotFound) {
		t.Errorf("GetNote(-1) err = %v, want ErrNotFound", err)
	}
}
