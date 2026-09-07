package feed

import (
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"
)

const maxTitleLen = 80

// Item is one feed entry in feedstack's own vocabulary.
type Item struct {
	Title     string    `json:"title"`
	Link      string    `json:"link"`
	Source    string    `json:"source"`
	Published time.Time `json:"published"`
}

func (it *Item) clean() {
	it.Title = strings.TrimSpace(it.Title)
	if len(it.Title) > maxTitleLen {
		cut := maxTitleLen
		for cut > 0 && !utf8.RuneStart(it.Title[cut]) {
			cut--
		}
		it.Title = strings.TrimSpace(it.Title[:cut])
	}
}

func (it *Item) line() string {
	return fmt.Sprintf("%s -> %s", it.Title, it.Link)
}

func WriteItems(w io.Writer, items []Item) error {
	for _, item := range items {
		if _, err := fmt.Fprintln(w, item.line()); err != nil {
			return fmt.Errorf("write item: %w", err)
		}
	}
	return nil
}
