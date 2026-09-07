package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"os/signal"
	"slices"

	"feedstack/feed"
	"feedstack/fetch"
	"feedstack/internal/config"
)

const appName = "feedstack"

type status int

const (
	statusIdle status = iota
	statusFetching
	statusDone
	statusFailed
)

func (s status) String() string {
	switch s {
	case statusIdle:
		return "idle"
	case statusFetching:
		return "fetching"
	case statusDone:
		return "done"
	case statusFailed:
		return "failed"
	}
	panic(fmt.Sprintf("unknown status: %d", int(s)))
}

func countBy[T any, K comparable](s []T, key func(T) K) map[K]int {
	counts := make(map[K]int)
	for _, v := range s {
		counts[key(v)]++
	}
	return counts
}

func saveItems(path string, items []feed.Item) error {
	var buf bytes.Buffer
	if err := feed.WriteItems(&buf, items); err != nil {
		return fmt.Errorf("save items: %w", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("save items: %w", err)
	}
	return nil
}

func report(items []feed.Item, errs []error, total int) {
	for _, err := range errs {
		fmt.Println(appName, "source failed:", err)
	}

	status := statusDone
	if len(errs) == total {
		status = statusFailed
	}

	switch status {
	case statusDone:
		fmt.Printf("%s %v: %d items from %d of %d sources\n",
			appName, status, len(items), total-len(errs), total)
		counts := countBy(items, func(it feed.Item) string { return it.Source })
		fmt.Println("by source:")
		for _, src := range slices.Sorted(maps.Keys(counts)) {
			fmt.Printf("  %s: %d\n", src, counts[src])
		}
		fmt.Println("latest:")
		_ = feed.WriteItems(os.Stdout, items[:min(3, len(items))])
	case statusFailed:
		fmt.Printf("%s %v: all %d sources failed\n", appName, status, total)
	default:
		fmt.Println(appName, "stopped in an unexpected state")
	}
}

func main() {
	defer fmt.Println(appName, "shutting down")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	lines, err := config.Load("feeds.txt")
	if err != nil {
		fmt.Println(appName, "cannot start:", err)
		return
	}
	if len(lines) == 0 {
		fmt.Println(appName, "cannot start: feed list is empty")
		return
	}
	fmt.Println(appName, "sources:", lines)

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	srcs := make([]fetch.Source, 0, len(lines))
	for _, line := range lines {
		srcs = append(srcs, fetch.WithLogging(fetch.New(line), logger))
	}

	seen := fetch.NewSeenSet()
	items, errs := fetch.Aggregate(ctx, srcs, seen, fetch.Options{})

	report(items, errs, len(srcs))

	if err := saveItems("items.txt", items); err != nil {
		fmt.Println(appName, "could not save items:", err)
	} else {
		fmt.Println("saved", len(items), "items to items.txt")
	}
}
