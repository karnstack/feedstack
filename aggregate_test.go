package main

import (
	"context"
	"errors"
	"testing"
)

type stubSource struct {
	items []feedItem
	err   error
}

func (s stubSource) fetch(ctx context.Context) ([]feedItem, error) {
	return s.items, s.err
}

func TestAggregate(t *testing.T) {
	one := feedItem{title: "one", link: "https://feeds.example/one"}
	two := feedItem{title: "two", link: "https://feeds.example/two"}

	srcs := []source{
		stubSource{items: []feedItem{one, two}},
		stubSource{items: []feedItem{one}}, // duplicate of a link already served
		stubSource{err: errors.New("connection dropped")},
	}

	items, errs := aggregate(context.Background(), srcs, newSeenSet())

	if len(items) != 2 {
		t.Errorf("aggregate() returned %d items, want 2", len(items))
	}
	if len(errs) != 1 {
		t.Errorf("aggregate() returned %d errors, want 1", len(errs))
	}
}

func TestSeenSetAdd(t *testing.T) {
	s := newSeenSet()
	if !s.add("https://feeds.example/one") {
		t.Error("first add returned false, want true")
	}
	if s.add("https://feeds.example/one") {
		t.Error("second add returned true, want false")
	}
}
