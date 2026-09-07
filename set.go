package main

type set[K comparable] map[K]struct{}

func (s set[K]) add(k K) { s[k] = struct{}{} }

func (s set[K]) has(k K) bool {
	_, ok := s[k]
	return ok
}
