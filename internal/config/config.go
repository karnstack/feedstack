package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
)

const starterFeeds = `# feedstack feed list: one url per line
https://www.jsonfeed.org/feed.json
`

func parse(r io.Reader) ([]string, error) {
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

// Load reads the feed list at path, creating a starter file on first run.
func Load(path string) ([]string, error) {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		fmt.Println("feedstack first run: writing starter", path)
		if err := os.WriteFile(path, []byte(starterFeeds), 0o644); err != nil {
			return nil, fmt.Errorf("write starter feed list: %w", err)
		}
		return parse(strings.NewReader(starterFeeds))
	}
	if err != nil {
		return nil, fmt.Errorf("load feed list: %w", err)
	}
	defer f.Close()
	return parse(f)
}
