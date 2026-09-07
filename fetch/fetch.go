package fetch

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"feedstack/feed"
)

// Source is anything that can deliver a batch of feed items.
type Source interface {
	Fetch(ctx context.Context) ([]feed.Item, error)
}

type fileSource struct {
	path string
}

func (s fileSource) Fetch(ctx context.Context) ([]feed.Item, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f, err := os.Open(s.path)
	if err != nil {
		return nil, fmt.Errorf("open feed file: %w", err)
	}
	defer f.Close()
	items, err := feed.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", s.path, err)
	}
	return items, nil
}

type httpSource struct {
	url string
}

func (s httpSource) Fetch(ctx context.Context) ([]feed.Item, error) {
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
	items, err := feed.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", s.url, err)
	}
	return items, nil
}

// New picks the right source for one line of the feed list.
func New(line string) Source {
	if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
		return httpSource{url: line}
	}
	return fileSource{path: line}
}

type loggingSource struct {
	Source
}

func (l loggingSource) Fetch(ctx context.Context) ([]feed.Item, error) {
	start := time.Now()
	items, err := l.Source.Fetch(ctx)
	if err != nil {
		fmt.Printf("%T: failed after %v: %v\n", l.Source, time.Since(start), err)
		return nil, err
	}
	fmt.Printf("%T: %d items in %v\n", l.Source, len(items), time.Since(start))
	return items, nil
}

func WithLogging(src Source) Source {
	return loggingSource{Source: src}
}

type SeenSet struct {
	mu    sync.Mutex
	links map[string]struct{}
}

func NewSeenSet() *SeenSet {
	return &SeenSet{links: make(map[string]struct{})}
}

func (s *SeenSet) add(link string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.links[link]; ok {
		return false
	}
	s.links[link] = struct{}{}
	return true
}

type fetchResult struct {
	items []feed.Item
	err   error
}

const (
	maxWorkers   = 4
	fetchTimeout = 5 * time.Second
)

func safeFetch(ctx context.Context, src Source) (items []feed.Item, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("fetch panicked: %v", r)
		}
	}()
	return src.Fetch(ctx)
}

func Aggregate(ctx context.Context, srcs []Source, seen *SeenSet) ([]feed.Item, []error) {
	jobs := make(chan Source, len(srcs))
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

				var fresh []feed.Item
				for _, item := range batch {
					if seen.add(item.Link) {
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

	var items []feed.Item
	var errs []error
	for res := range results {
		if res.err != nil {
			errs = append(errs, res.err)
			continue
		}
		items = append(items, res.items...)
	}

	slices.SortFunc(items, func(a, b feed.Item) int {
		return b.Published.Compare(a.Published)
	})
	return items, errs
}
