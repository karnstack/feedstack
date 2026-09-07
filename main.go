package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
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

func newFetcher(source string) func() (feedItem, error) {
	n := 0
	return func() (feedItem, error) {
		n++
		if n > feedSize {
			return feedItem{}, fmt.Errorf("%s: %w", source, errFeedExhausted)
		}
		if n%7 == 0 {
			return feedItem{}, fmt.Errorf("%s: %w", source, &fetchError{source: source, itemID: n})
		}
		id := n
		if n%5 == 0 {
			id = n - 1 // the feed re-serves the previous item
		}
		return feedItem{
			title:  fmt.Sprintf("\t item %d from %s \n", id, source),
			link:   fmt.Sprintf("https://cafecorner.example/items/%d", id),
			source: source,
		}, nil
	}
}

func cleanTitle(raw string) string {
	return strings.TrimSpace(raw)
}

const starterFeeds = `# feedstack feed list: one source per line
café corner
`

func parseFeeds(r io.Reader) ([]string, error) {
	var sources []string
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		sources = append(sources, line)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("parse feed list: %w", err)
	}
	return sources, nil
}

func loadFeeds(path string) ([]string, error) {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		fmt.Println(appName, "first run: writing starter", path)
		if err := os.WriteFile(path, []byte(starterFeeds), 0o644); err != nil {
			return nil, fmt.Errorf("write starter feed list: %w", err)
		}
		return parseFeeds(strings.NewReader(starterFeeds))
	}
	if err != nil {
		return nil, fmt.Errorf("load feed list: %w", err)
	}
	defer f.Close()
	return parseFeeds(f)
}

func writeItems(w io.Writer, items []feedItem) error {
	for _, item := range items {
		if _, err := fmt.Fprintf(w, "%s -> %s\n", item.title, item.link); err != nil {
			return fmt.Errorf("write item: %w", err)
		}
	}
	return nil
}

func saveItems(path string, items []feedItem) error {
	var buf bytes.Buffer
	if err := writeItems(&buf, items); err != nil {
		return fmt.Errorf("save items: %w", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("save items: %w", err)
	}
	return nil
}

func main() {
	defer fmt.Println(appName, "shutting down")

	sources, err := loadFeeds("feeds.txt")
	if err != nil {
		fmt.Println(appName, "cannot start:", err)
		return
	}
	if len(sources) == 0 {
		fmt.Println(appName, "cannot start: feed list is empty")
		return
	}
	fmt.Println(appName, "sources:", sources)

	status := statusFetching
	items := make([]feedItem, 0, maxItems)
	seen := make(map[string]bool)
	var dropped []int
	var lastErr error

	fetchNext := newFetcher(sources[0])
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
		_ = writeItems(os.Stdout, items[len(items)-3:])
		if err := saveItems("items.txt", items); err != nil {
			fmt.Println(appName, "could not save items:", err)
		} else {
			fmt.Println("saved", len(items), "items to items.txt")
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
