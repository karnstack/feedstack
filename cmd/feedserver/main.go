package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"feedstack/fetch"
	"feedstack/internal/config"
)

func newLogger(format string) *slog.Logger {
	switch format {
	case "json":
		return slog.New(slog.NewJSONHandler(os.Stderr, nil))
	default:
		return slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
}

func main() {
	defer fmt.Println("feedserver: goodbye")

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "feedserver: cannot start:", err)
		os.Exit(1)
	}

	logger := newLogger(cfg.logFormat)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	lines, err := config.Load(cfg.feedsPath)
	if err != nil {
		logger.Error("cannot start", "err", err)
		os.Exit(1)
	}

	var srcs []fetch.Source
	for _, line := range lines {
		srcs = append(srcs, fetch.WithLogging(fetch.New(line), logger))
	}

	seen := fetch.NewSeenSet()
	st := &store{}
	m := &metrics{}

	refresh := func() {
		items, errs := fetch.Aggregate(ctx, srcs, seen, fetch.Options{
			Workers: cfg.workers,
			Timeout: cfg.fetchTimeout,
		})
		st.add(items)
		m.refreshes.Add(1)
		m.itemsAdded.Add(int64(len(items)))
		m.fetchErrors.Add(int64(len(errs)))
		m.lastRefresh.Store(time.Now().Unix())
		logger.Info("refresh complete",
			"new_items", len(items), "errors", len(errs), "store", st.len())
	}
	refresh() // warm start

	go func() {
		ticker := time.NewTicker(cfg.refresh)
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

	handler := logRequests(logger, m, newMux(st, m))
	srv := &http.Server{
		Addr:              cfg.addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	go func() {
		logger.Info("listening", "addr", cfg.addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("listen failed", "err", err)
			stop()
		}
	}()

	<-ctx.Done()

	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		logger.Error("shutdown", "err", err)
	}
}
