package main

import (
	"fmt"
	"strings"
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

func newFetcher() func() (string, bool) {
	seen := 0
	return func() (string, bool) {
		seen++
		if seen%5 == 0 {
			return "", false // simulated duplicate: skip it
		}
		return fmt.Sprintf("\t item %d from café corner \n", seen), true
	}
}

func cleanTitle(raw string) string {
	return strings.TrimSpace(raw)
}

func main() {
	defer fmt.Println(appName, "shutting down")

	status := statusFetching
	titles := make([]string, 0, maxItems)

	fetchNext := newFetcher()
	for len(titles) < maxItems {
		title, ok := fetchNext()
		if !ok {
			continue
		}
		titles = append(titles, cleanTitle(title))
	}
	status = statusDone

	switch status {
	case statusDone:
		fmt.Printf("%s done: %d items fetched\n", appName, len(titles))
		fmt.Printf("latest: %q\n", titles[len(titles)-3:])
	case statusFailed:
		fmt.Printf("%s failed after %d items\n", appName, len(titles))
	default:
		fmt.Println(appName, "stopped in an unexpected state")
	}
}
