// Package idgen isolates UUID generation so tests can pin IDs.
package idgen

import (
	"sync"

	"github.com/google/uuid"
)

type IDGen interface {
	New() uuid.UUID
}

type UUID struct{}

func (UUID) New() uuid.UUID { return uuid.New() }

type Static struct {
	mu  sync.Mutex
	ids []uuid.UUID
	idx int
}

func NewStatic(ids ...uuid.UUID) *Static { return &Static{ids: ids} }

func (s *Static) New() uuid.UUID {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.ids) == 0 {
		return uuid.Nil
	}
	v := s.ids[s.idx%len(s.ids)]
	s.idx++
	return v
}
