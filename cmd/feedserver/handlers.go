package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"feedstack/feed"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		fmt.Println("write response:", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *store) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"items":  len(s.snapshot()),
	})
}

func (s *store) handleItems(w http.ResponseWriter, r *http.Request) {
	items := s.snapshot()

	if q := r.URL.Query().Get("limit"); q != "" {
		n, err := strconv.Atoi(q)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, "limit must be a non-negative number")
			return
		}
		items = items[:min(n, len(items))]
	}

	if items == nil {
		items = []feed.Item{} // encode [], never null
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *store) handleFeed(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("source")

	var matched []feed.Item
	for _, item := range s.snapshot() {
		if item.Source == name {
			matched = append(matched, item)
		}
	}
	if matched == nil {
		writeError(w, http.StatusNotFound, "unknown source: "+name)
		return
	}
	writeJSON(w, http.StatusOK, matched)
}
