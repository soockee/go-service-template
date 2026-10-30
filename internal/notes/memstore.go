package notes

import (
	"context"
	"slices"
	"sync"
	"time"
)

// MemStore is an in-memory Store for tests and for running the service
// locally without a database.
type MemStore struct {
	mu     sync.RWMutex
	nextID int64
	notes  map[int64]Note
}

func NewMemStore() *MemStore {
	return &MemStore{notes: make(map[int64]Note)}
}

func (m *MemStore) CreateNote(_ context.Context, title, body string) (Note, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	n := Note{ID: m.nextID, Title: title, Body: body, CreatedAt: time.Now().UTC()}
	m.notes[n.ID] = n
	return n, nil
}

func (m *MemStore) GetNote(_ context.Context, id int64) (Note, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n, ok := m.notes[id]
	if !ok {
		return Note{}, ErrNotFound
	}
	return n, nil
}

// ListNotes returns notes newest first, matching the Postgres store.
func (m *MemStore) ListNotes(_ context.Context) ([]Note, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Note, 0, len(m.notes))
	for _, n := range m.notes {
		out = append(out, n)
	}
	slices.SortFunc(out, func(a, b Note) int { return int(b.ID - a.ID) })
	return out, nil
}
