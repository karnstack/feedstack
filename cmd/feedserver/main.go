package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"feedstack/fetch"
	"feedstack/internal/config"
)

const (
	addr            = ":8080"
	refreshEvery    = 15 * time.Minute
	shutdownTimeout = 10 * time.Second
)

func main() {
	defer fmt.Println("feedserver: goodbye")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	lines, err := config.Load("feeds.txt")
	if err != nil {
		fmt.Println("feedserver cannot start:", err)
		return
	}
	if len(lines) == 0 {
		fmt.Println("feedserver cannot start: no sources in feeds.txt")
		return
	}

	srcs := make([]fetch.Source, 0, len(lines))
	for _, line := range lines {
		srcs = append(srcs, fetch.WithLogging(fetch.New(line)))
	}

	seen := fetch.NewSeenSet()
	st := &store{}

	refresh := func() {
		items, errs := fetch.Aggregate(ctx, srcs, seen)
		for _, err := range errs {
			fmt.Println("refresh:", err)
		}
		st.add(items)
	}
	refresh()

	go func() {
		ticker := time.NewTicker(refreshEvery)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				refresh()
			case <-ctx.Done():
				return
			}
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", st.handleHealthz)
	mux.HandleFunc("GET /items", st.handleItems)
	mux.HandleFunc("GET /feeds/{source}", st.handleFeed)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	go func() {
		fmt.Println("feedserver listening on", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Println("server:", err)
			stop() // exit through the same drain as ctrl-C
		}
	}()

	<-ctx.Done()

	shutCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		fmt.Println("shutdown:", err)
	}
}
