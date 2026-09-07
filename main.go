package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const appName = "feedstack"

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
	panic(fmt.Sprintf("unknown status: %d", int(s)))
}

type feedItem struct {
	title     string
	link      string
	source    string
	published time.Time
}

func (it *feedItem) clean() {
	it.title = strings.TrimSpace(it.title)
	if len(it.title) > maxTitleLen {
		cut := maxTitleLen
		for cut > 0 && !utf8.RuneStart(it.title[cut]) {
			cut--
		}
		it.title = strings.TrimSpace(it.title[:cut])
	}
}

func (it *feedItem) line() string {
	return fmt.Sprintf("%s -> %s", it.title, it.link)
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
		if tags, ok := entry["tags"].([]any); ok && len(tags) > 0 {
			if label, ok := tags[0].(string); ok {
				title = "[" + label + "] " + title
			}
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

func (s fileSource) fetch(ctx context.Context) ([]feedItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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

func (s httpSource) fetch(ctx context.Context) ([]feedItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
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
	fetch(ctx context.Context) ([]feedItem, error)
}

type loggingSource struct {
	source
}

func (l loggingSource) fetch(ctx context.Context) ([]feedItem, error) {
	start := time.Now()
	items, err := l.source.fetch(ctx)
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

type seenSet struct {
	mu    sync.Mutex
	links map[string]struct{}
}

func newSeenSet() *seenSet {
	return &seenSet{links: make(map[string]struct{})}
}

func (s *seenSet) add(link string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.links[link]; ok {
		return false
	}
	s.links[link] = struct{}{}
	return true
}

const (
	maxWorkers   = 4
	fetchTimeout = 5 * time.Second
	maxTitleLen  = 80 // bytes; longer titles get cut
)

func safeFetch(ctx context.Context, src source) (items []feedItem, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("fetch panicked: %v", r)
		}
	}()
	return src.fetch(ctx)
}

func aggregate(ctx context.Context, srcs []source, seen *seenSet) ([]feedItem, []error) {
	jobs := make(chan source, len(srcs))
	for _, src := range srcs {
		jobs <- src
	}
	close(jobs)

	results := make(chan fetchResult)

	workers := min(maxWorkers, len(srcs))
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for src := range jobs {
				fetchCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
				batch, err := safeFetch(fetchCtx, src)
				cancel()

				var fresh []feedItem
				for _, item := range batch {
					if seen.add(item.link) {
						fresh = append(fresh, item)
					}
				}
				results <- fetchResult{items: fresh, err: err}
			}
		})
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var items []feedItem
	var errs []error
	for res := range results {
		if res.err != nil {
			errs = append(errs, res.err)
			continue
		}
		items = append(items, res.items...)
	}

	slices.SortFunc(items, func(a, b feedItem) int {
		return b.published.Compare(a.published)
	})
	return items, errs
}

func report(items []feedItem, errs []error, total int) {
	for _, err := range errs {
		fmt.Println(appName, "source failed:", err)
	}

	status := statusDone
	if len(errs) == total {
		status = statusFailed
	}

	switch status {
	case statusDone:
		fmt.Printf("%s %v: %d items from %d of %d sources\n",
			appName, status, len(items), total-len(errs), total)
		counts := countBy(items, func(it feedItem) string { return it.source })
		fmt.Println("by source:")
		for _, src := range slices.Sorted(maps.Keys(counts)) {
			fmt.Printf("  %s: %d\n", src, counts[src])
		}
		fmt.Println("latest:")
		_ = writeItems(os.Stdout, items[:min(3, len(items))])
	case statusFailed:
		fmt.Printf("%s %v: all %d sources failed\n", appName, status, total)
	default:
		fmt.Println(appName, "stopped in an unexpected state")
	}
}

func main() {
	defer fmt.Println(appName, "shutting down")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

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

	srcs := make([]source, 0, len(lines))
	for _, line := range lines {
		srcs = append(srcs, loggingSource{source: newSource(line)})
	}

	seen := newSeenSet()
	items, errs := aggregate(ctx, srcs, seen)

	report(items, errs, len(srcs))

	if err := saveItems("items.txt", items); err != nil {
		fmt.Println(appName, "could not save items:", err)
	} else {
		fmt.Println("saved", len(items), "items to items.txt")
	}
}
