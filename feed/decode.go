package feed

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

type wireFeed struct {
	Title string     `json:"title"`
	Items []wireItem `json:"items"`
}

type wireItem struct {
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Published time.Time `json:"date_published"`
}

func Decode(r io.Reader) ([]Item, error) {
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

func decodeJSONFeed(data []byte) ([]Item, error) {
	var feed wireFeed
	if err := json.Unmarshal(data, &feed); err != nil {
		return nil, fmt.Errorf("decode json feed: %w", err)
	}
	items := make([]Item, 0, len(feed.Items))
	for _, w := range feed.Items {
		item := Item{
			Title:     w.Title,
			Link:      w.URL,
			Source:    feed.Title,
			Published: w.Published,
		}
		item.clean()
		items = append(items, item)
	}
	return items, nil
}

func decodeLegacy(doc map[string]any) ([]Item, error) {
	channel, ok := doc["channel"].(map[string]any)
	if !ok {
		return nil, errors.New("legacy feed: no channel object")
	}
	source, _ := channel["title"].(string)

	rawItems, ok := channel["items"].([]any)
	if !ok {
		return nil, errors.New("legacy feed: no items array")
	}

	items := make([]Item, 0, len(rawItems))
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
		item := Item{Title: title, Link: link, Source: source}
		item.clean()
		items = append(items, item)
	}
	return items, nil
}
