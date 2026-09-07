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

func main() {
	status := statusFetching
	var itemsSeen, itemsFetched int

	for {
		if itemsFetched == maxItems {
			status = statusDone
			break
		}

		itemsSeen++
		if itemsSeen%5 == 0 {
			continue // simulated duplicate: skip it, fetch nothing
		}
		itemsFetched++
	}

	switch status {
	case statusDone:
		fmt.Println(appName, "done:", itemsFetched, "items fetched,", itemsSeen, "seen")
	case statusFailed:
		fmt.Println(appName, "failed after", itemsFetched, "items")
	default:
		fmt.Println(appName, "stopped in an unexpected state")
	}
}
