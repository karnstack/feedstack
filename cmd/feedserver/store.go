package main

import (
	"slices"
	"sync"

	"feedstack/feed"
)

type store struct {
	mu    sync.RWMutex
	items []feed.Item
}

func (s *store) snapshot() []feed.Item {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.items
}

func (s *store) len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

func (s *store) add(fresh []feed.Item) {
	if len(fresh) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	merged := append(slices.Clone(s.items), fresh...)
	slices.SortFunc(merged, func(a, b feed.Item) int {
		return b.Published.Compare(a.Published)
	})
	s.items = merged
}
