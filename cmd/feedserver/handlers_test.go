package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"feedstack/feed"
)

func testStore(t *testing.T) *store {
	t.Helper()
	st := &store{}
	st.add([]feed.Item{
		{
			Title:     "hello server",
			Link:      "https://feedstack.example/hello",
			Source:    "feedstack dev notes",
			Published: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		},
		{
			Title:     "counterfeits in court",
			Link:      "https://feedstack.example/counterfeits",
			Source:    "feedstack dev notes",
			Published: time.Date(2026, 6, 3, 9, 0, 0, 0, time.UTC),
		},
		{
			Title:     "the recorder remembers",
			Link:      "https://morningbrew.example/recorder",
			Source:    "morning brew",
			Published: time.Date(2026, 6, 2, 9, 0, 0, 0, time.UTC),
		},
	})
	return st
}

func TestHandleHealthz(t *testing.T) {
	st := testStore(t)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	st.handleHealthz(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	var got struct {
		Status string `json:"status"`
		Items  int    `json:"items"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.Status != "ok" {
		t.Errorf("status field = %q, want %q", got.Status, "ok")
	}
	if got.Items != 3 {
		t.Errorf("items field = %d, want 3", got.Items)
	}
}

func TestHandleItems(t *testing.T) {
	st := testStore(t)
	req := httptest.NewRequest(http.MethodGet, "/items?limit=2", nil)
	rec := httptest.NewRecorder()

	st.handleItems(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got []feed.Item
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	titles := make([]string, 0, len(got))
	for _, item := range got {
		titles = append(titles, item.Title)
	}
	want := []string{"counterfeits in court", "the recorder remembers"}
	if diff := cmp.Diff(want, titles); diff != "" {
		t.Errorf("titles mismatch (-want +got):\n%s", diff)
	}
}

func TestHandleItemsEmptyStore(t *testing.T) {
	st := &store{} // exactly the server at boot, before the first refresh
	req := httptest.NewRequest(http.MethodGet, "/items", nil)
	rec := httptest.NewRecorder()

	st.handleItems(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got []feed.Item
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got == nil {
		t.Error("body decoded to nil: the API answered null, want []")
	}
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

func TestHandleItemsLimit(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		wantStatus int
		wantLen    int
	}{
		{"no limit returns everything", "/items", http.StatusOK, 3},
		{"limit clamps politely", "/items?limit=999", http.StatusOK, 3},
		{"limit zero is legal", "/items?limit=0", http.StatusOK, 0},
		{"limit must be a number", "/items?limit=banana", http.StatusBadRequest, 0},
		{"limit must not be negative", "/items?limit=-1", http.StatusBadRequest, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := testStore(t)
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			rec := httptest.NewRecorder()

			st.handleItems(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantStatus != http.StatusOK {
				var e struct {
					Error string `json:"error"`
				}
				if err := json.NewDecoder(rec.Body).Decode(&e); err != nil {
					t.Fatalf("decode error body: %v", err)
				}
				if e.Error == "" {
					t.Error("error body carries no message")
				}
				return
			}

			var got []feed.Item
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestHandleFeed(t *testing.T) {
	st := testStore(t)
	req := httptest.NewRequest(http.MethodGet, "/feeds/anything", nil)
	req.SetPathValue("source", "morning brew")
	rec := httptest.NewRecorder()

	st.handleFeed(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got []feed.Item
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(got) != 1 || got[0].Source != "morning brew" {
		t.Errorf("got %d items, want morning brew's 1", len(got))
	}
}

func TestRouting(t *testing.T) {
	st := testStore(t)
	mux := newMux(st)

	tests := []struct {
		name       string
		method     string
		target     string
		wantStatus int
	}{
		{"healthz answers", http.MethodGet, "/healthz", http.StatusOK},
		{"items answers", http.MethodGet, "/items", http.StatusOK},
		{"wrong method is refused", http.MethodPost, "/items", http.StatusMethodNotAllowed},
		{"unknown path is refused", http.MethodGet, "/nope", http.StatusNotFound},
		{"wildcard matches a source", http.MethodGet, "/feeds/morning%20brew", http.StatusOK},
		{"unknown source is a 404", http.MethodGet, "/feeds/the%20daily%20void", http.StatusNotFound},
		{"bad limit is refused", http.MethodGet, "/items?limit=banana", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.target, nil)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("%s %s: status = %d, want %d",
					tt.method, tt.target, rec.Code, tt.wantStatus)
			}
		})
	}
}
