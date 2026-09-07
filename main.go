package main

import "fmt"

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
		return fmt.Sprintf("item %d", seen), true
	}
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
		itemsFetched++
		lastTitle = title
	}
	status = statusDone

	switch status {
	case statusDone:
		fmt.Println(appName, "done:", itemsFetched, "items fetched, last was", lastTitle)
	case statusFailed:
		fmt.Println(appName, "failed after", itemsFetched, "items")
	default:
		fmt.Println(appName, "stopped in an unexpected state")
	}
}
