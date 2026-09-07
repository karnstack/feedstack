package main

import "fmt"

const (
	appName        = "feedstack"
	defaultFeedURL = "https://go.dev/blog/feed.atom"
	maxItems       = 50
)

func main() {
	var itemsFetched int
	var lastError string

	fmt.Println(appName, "starting")
	fmt.Println("feed:", defaultFeedURL)
	fmt.Println("items fetched:", itemsFetched)
	fmt.Println("last error:", lastError)
}
