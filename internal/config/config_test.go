package config

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestParseFeeds(t *testing.T) {
	input := "# feedstack sources\n\nhttps://www.jsonfeed.org/feed.json\n  local.json  \n"

	got, err := parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse() returned error: %v", err)
	}

	want := []string{"https://www.jsonfeed.org/feed.json", "local.json"}
	if !slices.Equal(got, want) {
		t.Errorf("parse() = %q, want %q", got, want)
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("connection reset")
}

func TestParseFeedsReadError(t *testing.T) {
	_, err := parse(failingReader{})
	if err == nil {
		t.Fatal("parse() returned nil error for a failing reader, want non-nil")
	}
}
