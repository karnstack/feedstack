package main

import (
	"net/http"
	"sync/atomic"
)

type metrics struct {
	requests    atomic.Int64
	refreshes   atomic.Int64
	itemsAdded  atomic.Int64
	fetchErrors atomic.Int64
	lastRefresh atomic.Int64 // unix seconds
}

type metricsSnapshot struct {
	Requests    int64 `json:"requests"`
	Refreshes   int64 `json:"refreshes"`
	ItemsAdded  int64 `json:"items_added"`
	FetchErrors int64 `json:"fetch_errors"`
	LastRefresh int64 `json:"last_refresh_unix"`
}

func (m *metrics) handleMetrics(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, metricsSnapshot{
		Requests:    m.requests.Load(),
		Refreshes:   m.refreshes.Load(),
		ItemsAdded:  m.itemsAdded.Load(),
		FetchErrors: m.fetchErrors.Load(),
		LastRefresh: m.lastRefresh.Load(),
	})
}
