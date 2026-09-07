package fetch

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

const counterfeitFeed = `{
	"version": "https://jsonfeed.org/version/1.1",
	"title": "counterfeit corner",
	"items": [
		{"title": "second post", "url": "https://counterfeit.example/two", "date_published": "2026-06-02T09:00:00Z"},
		{"title": "  first post  ", "url": "https://counterfeit.example/one", "date_published": "2026-06-01T09:00:00Z"}
	]
}`

func TestHTTPSourceFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, counterfeitFeed)
	}))
	defer srv.Close()

	items, err := New(srv.URL).Fetch(t.Context())
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if got, want := items[1].Title, "first post"; got != want {
		t.Errorf("title = %q, want %q (clean should scrub the wire title)", got, want)
	}
	if got, want := items[0].Source, "counterfeit corner"; got != want {
		t.Errorf("source = %q, want %q (the feed names itself)", got, want)
	}
}

func TestHTTPSourceFetchServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "the upstream is having a day", http.StatusInternalServerError)
	}))
	defer srv.Close()

	if _, err := New(srv.URL).Fetch(t.Context()); err == nil {
		t.Fatal("want an error for a 500 response, got nil")
	}
}

func TestAggregateOverHTTP(t *testing.T) {
	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, counterfeitFeed)
	}))
	defer healthy.Close()

	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer broken.Close()

	srcs := []Source{New(healthy.URL), New(broken.URL)}
	items, errs := Aggregate(t.Context(), srcs, NewSeenSet())

	if len(items) != 2 {
		t.Errorf("items = %d, want 2", len(items))
	}
	if len(errs) != 1 {
		t.Errorf("errs = %d, want 1", len(errs))
	}
}
