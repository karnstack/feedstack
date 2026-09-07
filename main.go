package main

import (
	"fmt"
	"strings"
	"time"
)

const (
	appName  = "feedstack"
	maxItems = 50
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

func newFetcher() func() feedItem {
	n := 0
	return func() feedItem {
		n++
		id := n
		if n%5 == 0 {
			id = n - 1 // the feed re-serves the previous item
		}
		return feedItem{
			title:  fmt.Sprintf("\t item %d from café corner \n", id),
			link:   fmt.Sprintf("https://cafecorner.example/items/%d", id),
			source: "café corner",
		}
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

	fetchNext := newFetcher()
	for len(items) < maxItems {
		item := fetchNext()
		if seen[item.link] {
			continue
		}
		seen[item.link] = true
		item.title = cleanTitle(item.title)
		items = append(items, item)
	}
	status = statusDone

	switch status {
	case statusDone:
		fmt.Printf("%s done: %d items fetched\n", appName, len(items))
		fmt.Println("latest:")
		for _, item := range items[len(items)-3:] {
			fmt.Printf("  %q -> %s\n", item.title, item.link)
		}
	case statusFailed:
		fmt.Printf("%s failed after %d items\n", appName, len(items))
	default:
		fmt.Println(appName, "stopped in an unexpected state")
	}
}
