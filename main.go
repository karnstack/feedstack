package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	appName  = "feedstack"
	maxItems = 50
)

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
	return "unknown"
}

type feedItem struct {
	title     string
	link      string
	source    string
	published time.Time
}

func (it *feedItem) clean() {
	it.title = strings.TrimSpace(it.title)
}

func (it *feedItem) line() string {
	return fmt.Sprintf("%s -> %s", it.title, it.link)
}

type itemList []feedItem

func (ls itemList) latest(n int) itemList {
	start := len(ls) - n
	if start < 0 {
		start = 0
	}
	return ls[start:]
}

type wireFeed struct {
	Title string     `json:"title"`
	Items []wireItem `json:"items"`
}

type wireItem struct {
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Published time.Time `json:"date_published"`
}

func fetchFeed(url string) ([]feedItem, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: unexpected status %s", url, resp.Status)
	}

	var feed wireFeed
	if err := json.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, fmt.Errorf("decode %s: %w", url, err)
	}

	items := make([]feedItem, 0, len(feed.Items))
	for _, wi := range feed.Items {
		item := feedItem{
			title:     wi.Title,
			link:      wi.URL,
			source:    feed.Title,
			published: wi.Published,
		}
		item.clean()
		items = append(items, item)
	}
	return items, nil
}

const starterFeeds = `# feedstack feed list: one url per line
https://www.jsonfeed.org/feed.json
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
		if _, err := fmt.Fprintln(w, item.line()); err != nil {
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
	items := make(itemList, 0, maxItems)
	seen := make(map[string]bool)
	var failed int

	for _, url := range sources {
		fetched, err := fetchFeed(url)
		if err != nil {
			fmt.Println(appName, "skipping source:", err)
			failed++
			continue
		}
		for _, item := range fetched {
			if seen[item.link] {
				continue
			}
			seen[item.link] = true
			items = append(items, item)
		}
	}

	if failed == len(sources) {
		status = statusFailed
	} else {
		status = statusDone
	}

	switch status {
	case statusDone:
		fmt.Printf("%s %v: %d items from %d of %d sources\n",
			appName, status, len(items), len(sources)-failed, len(sources))
		fmt.Println("latest:")
		_ = writeItems(os.Stdout, items.latest(3))
		if err := saveItems("items.txt", items); err != nil {
			fmt.Println(appName, "could not save items:", err)
		} else {
			fmt.Println("saved", len(items), "items to items.txt")
		}
	case statusFailed:
		fmt.Printf("%s %v: all %d sources failed\n", appName, status, len(sources))
	default:
		fmt.Println(appName, "stopped in an unexpected state")
	}
}
