package notes

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestServiceCreate(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		wantErr error
	}{
		{name: "ok", title: "  groceries  "},
		{name: "empty title", title: "   ", wantErr: ErrInvalid},
		{name: "title too long", title: strings.Repeat("x", maxTitle+1), wantErr: ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(NewMemStore())
			n, err := svc.Create(context.Background(), tt.title, "body")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if err == nil && n.Title != strings.TrimSpace(tt.title) {
				t.Errorf("Title = %q, want trimmed", n.Title)
			}
		})
	}
}

func TestMemStoreListNewestFirst(t *testing.T) {
	ctx := context.Background()
	s := NewMemStore()
	for _, title := range []string{"a", "b", "c"} {
		if _, err := s.CreateNote(ctx, title, ""); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.ListNotes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].Title != "c" || got[2].Title != "a" {
		t.Errorf("ListNotes order = %+v", got)
	}
	if _, err := s.GetNote(ctx, 99); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetNote(99) err = %v, want ErrNotFound", err)
	}
}
