package fetch

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"feedstack/feed"
)

type stubSource struct {
	items []feed.Item
	err   error
}

func (s stubSource) Fetch(ctx context.Context) ([]feed.Item, error) {
	return s.items, s.err
}

func TestAggregate(t *testing.T) {
	first := feed.Item{
		Title:     "first post",
		Link:      "https://a.example/1",
		Source:    "stub feed",
		Published: time.Date(2026, 6, 2, 8, 0, 0, 0, time.UTC),
	}
	second := feed.Item{
		Title:     "second post",
		Link:      "https://a.example/2",
		Source:    "stub feed",
		Published: time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC),
	}

	srcs := []Source{
		stubSource{items: []feed.Item{first, second}},
		stubSource{items: []feed.Item{first}}, // duplicate of a link already served
		stubSource{err: errors.New("connection dropped")},
	}

	got, errs := Aggregate(context.Background(), srcs, NewSeenSet())

	want := []feed.Item{
		{Title: "first post", Link: "https://a.example/1", Source: "stub feed", Published: first.Published},
		{Title: "second post", Link: "https://a.example/2", Source: "stub feed", Published: second.Published},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("aggregate items mismatch (-want +got):\n%s", diff)
	}
	if len(errs) != 1 {
		t.Errorf("Aggregate() returned %d errors, want 1", len(errs))
	}
}

func TestSeenSetAdd(t *testing.T) {
	s := NewSeenSet()
	if !s.add("https://feeds.example/one") {
		t.Error("first add returned false, want true")
	}
	if s.add("https://feeds.example/one") {
		t.Error("second add returned true, want false")
	}
}
