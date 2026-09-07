package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
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

func recordKeep(fetched *int, last *string, title string) {
	*fetched++
	*last = title
}

func main() {
	defer fmt.Println(appName, "shutting down")

	status := statusFetching
	var itemsFetched int
	var lastTitle string

	fetchNext := newFetcher()
	for itemsFetched < maxItems {
		title, ok := fetchNext()
		if !ok {
			continue
		}
		recordKeep(&itemsFetched, &lastTitle, cleanTitle(title))
	}
	status = statusDone

	switch status {
	case statusDone:
		fmt.Printf("%s done: %d items fetched, last title %q\n", appName, itemsFetched, lastTitle)
		fmt.Printf("that title is %d bytes, %d runes\n", len(lastTitle), utf8.RuneCountInString(lastTitle))
	case statusFailed:
		fmt.Printf("%s failed after %d items\n", appName, itemsFetched)
	default:
		fmt.Println(appName, "stopped in an unexpected state")
	}
}
