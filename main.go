package main

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	appName  = "feedstack"
	maxItems = 50
	feedSize = 80
)

const (
	statusIdle = iota
	statusFetching
	statusDone
	statusFailed
)

type feedItem struct {
	title     string
	link      string
	source    string
	published time.Time
}

var errFeedExhausted = errors.New("feed exhausted")

type fetchError struct {
	source string
	itemID int
}

func (e *fetchError) Error() string {
	return fmt.Sprintf("item %d: connection dropped", e.itemID)
}

func newFetcher() func() (feedItem, error) {
	n := 0
	return func() (feedItem, error) {
		n++
		if n > feedSize {
			return feedItem{}, fmt.Errorf("café corner: %w", errFeedExhausted)
		}
		if n%7 == 0 {
			return feedItem{}, fmt.Errorf("café corner: %w", &fetchError{source: "café corner", itemID: n})
		}
		id := n
		if n%5 == 0 {
			id = n - 1 // the feed re-serves the previous item
		}
		return feedItem{
			title:  fmt.Sprintf("\t item %d from café corner \n", id),
			link:   fmt.Sprintf("https://cafecorner.example/items/%d", id),
			source: "café corner",
		}, nil
	}
}

func cleanTitle(raw string) string {
	return strings.TrimSpace(raw)
}

func main() {
	defer fmt.Println(appName, "shutting down")

	status := statusFetching
	items := make([]feedItem, 0, maxItems)
	seen := make(map[string]bool)
	var dropped []int
	var lastErr error

	fetchNext := newFetcher()
	for len(items) < maxItems {
		item, err := fetchNext()
		if err != nil {
			var fe *fetchError
			if errors.As(err, &fe) {
				dropped = append(dropped, fe.itemID)
				continue
			}
			lastErr = err
			status = statusFailed
			break
		}
		if seen[item.link] {
			continue
		}
		seen[item.link] = true
		item.title = cleanTitle(item.title)
		items = append(items, item)
	}
	if status != statusFailed {
		status = statusDone
	}

	switch status {
	case statusDone:
		fmt.Printf("%s done: %d items fetched, %d fetches dropped\n", appName, len(items), len(dropped))
		fmt.Println("dropped item ids:", dropped)
		fmt.Println("latest:")
		for _, item := range items[len(items)-3:] {
			fmt.Printf("  %q -> %s\n", item.title, item.link)
		}
	case statusFailed:
		if errors.Is(lastErr, errFeedExhausted) {
			fmt.Printf("%s: feed ran dry after %d items\n", appName, len(items))
		} else {
			fmt.Printf("%s failed after %d items: %v\n", appName, len(items), lastErr)
		}
	default:
		fmt.Println(appName, "stopped in an unexpected state")
	}
}
