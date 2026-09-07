package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"os"
	"slices"
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

func lastN[T any](s []T, n int) []T {
	if n > len(s) {
		n = len(s)
	}
	return s[len(s)-n:]
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

func decodeItems(r io.Reader) ([]feedItem, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read feed: %w", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("decode feed: %w", err)
	}

	if v, ok := doc["version"].(string); ok && strings.HasPrefix(v, "https://jsonfeed.org/") {
		return decodeJSONFeed(data)
	}
	if _, ok := doc["channel"].(map[string]any); ok {
		return decodeLegacy(doc)
	}
	return nil, errors.New("unknown feed format")
}

func decodeJSONFeed(data []byte) ([]feedItem, error) {
	var feed wireFeed
	if err := json.Unmarshal(data, &feed); err != nil {
		return nil, fmt.Errorf("decode json feed: %w", err)
	}
	items := make([]feedItem, 0, len(feed.Items))
	for _, w := range feed.Items {
		item := feedItem{
			title:     w.Title,
			link:      w.URL,
			source:    feed.Title,
			published: w.Published,
		}
		item.clean()
		items = append(items, item)
	}
	return items, nil
}

func decodeLegacy(doc map[string]any) ([]feedItem, error) {
	channel, ok := doc["channel"].(map[string]any)
	if !ok {
		return nil, errors.New("legacy feed: no channel object")
	}
	source, _ := channel["title"].(string)

	rawItems, ok := channel["items"].([]any)
	if !ok {
		return nil, errors.New("legacy feed: no items array")
	}

	items := make([]feedItem, 0, len(rawItems))
	for _, raw := range rawItems {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		title, ok := entry["headline"].(string)
		if !ok {
			continue
		}
		link, ok := entry["url"].(string)
		if !ok {
			continue
		}
		item := feedItem{title: title, link: link, source: source}
		item.clean()
		items = append(items, item)
	}
	return items, nil
}

type fileSource struct {
	path string
}

func (s fileSource) fetch() ([]feedItem, error) {
	f, err := os.Open(s.path)
	if err != nil {
		return nil, fmt.Errorf("open feed file: %w", err)
	}
	defer f.Close()
	items, err := decodeItems(f)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", s.path, err)
	}
	return items, nil
}

type httpSource struct {
	url string
}

func (s httpSource) fetch() ([]feedItem, error) {
	resp, err := http.Get(s.url)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", s.url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: unexpected status %s", s.url, resp.Status)
	}
	items, err := decodeItems(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", s.url, err)
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

type source interface {
	fetch() ([]feedItem, error)
}

type loggingSource struct {
	source
}

func (l loggingSource) fetch() ([]feedItem, error) {
	start := time.Now()
	items, err := l.source.fetch()
	if err != nil {
		fmt.Printf("%T: failed after %v: %v\n", l.source, time.Since(start), err)
		return nil, err
	}
	fmt.Printf("%T: %d items in %v\n", l.source, len(items), time.Since(start))
	return items, nil
}

func newSource(line string) source {
	if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
		return httpSource{url: line}
	}
	return fileSource{path: line}
}

type fetchResult struct {
	items []feedItem
	err   error
}

func main() {
	defer fmt.Println(appName, "shutting down")

	lines, err := loadFeeds("feeds.txt")
	if err != nil {
		fmt.Println(appName, "cannot start:", err)
		return
	}
	if len(lines) == 0 {
		fmt.Println(appName, "cannot start: feed list is empty")
		return
	}
	fmt.Println(appName, "sources:", lines)

	status := statusFetching
	var failed int

	srcs := make([]source, 0, len(lines))
	for _, line := range lines {
		srcs = append(srcs, loggingSource{source: newSource(line)})
	}

	results := make(chan fetchResult)

	for _, src := range srcs {
		go func() {
			batch, err := src.fetch()
			results <- fetchResult{items: batch, err: err}
		}()
	}

	var fetched []feedItem
	for range srcs {
		res := <-results
		if res.err != nil {
			fmt.Println(appName, "source failed:", res.err)
			failed++
			continue
		}
		fetched = append(fetched, res.items...)
	}

	items := slices.Collect(dedupedBy(fetched, func(it feedItem) string {
		return it.link
	}))

	if failed == len(srcs) {
		status = statusFailed
	} else {
		status = statusDone
	}

	switch status {
	case statusDone:
		fmt.Printf("%s %v: %d items from %d of %d sources\n",
			appName, status, len(items), len(srcs)-failed, len(srcs))
		counts := countBy(items, func(it feedItem) string { return it.source })
		fmt.Println("by source:")
		for _, src := range slices.Sorted(maps.Keys(counts)) {
			fmt.Printf("  %s: %d\n", src, counts[src])
		}
		fmt.Println("latest:")
		_ = writeItems(os.Stdout, lastN(items, 3))
		if err := saveItems("items.txt", items); err != nil {
			fmt.Println(appName, "could not save items:", err)
		} else {
			fmt.Println("saved", len(items), "items to items.txt")
		}
	case statusFailed:
		fmt.Printf("%s %v: all %d sources failed\n", appName, status, len(srcs))
	default:
		fmt.Println(appName, "stopped in an unexpected state")
	}
}
