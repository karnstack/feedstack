package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"feedstack/feed"
)

func newMux(st *store, m *metrics) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", st.handleHealthz)
	mux.HandleFunc("GET /items", st.handleItems)
	mux.HandleFunc("GET /feeds/{source}", st.handleFeed)
	mux.HandleFunc("GET /metrics", m.handleMetrics)
	return mux
}

type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.code = code
	r.ResponseWriter.WriteHeader(code)
}

func logRequests(log *slog.Logger, m *metrics, next http.Handler) http.Handler {
	var nextID atomic.Int64
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, code: http.StatusOK}
		reqLog := log.With("request_id", nextID.Add(1), "method", r.Method, "path", r.URL.Path)

		next.ServeHTTP(rec, r)

		m.requests.Add(1)
		reqLog.Info("request", "status", rec.code, "elapsed", time.Since(start))
	})
}

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
