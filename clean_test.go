package main

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCleanTitle(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "tabs and trailing newline", in: "\t item 62 from café corner \n", want: "item 62 from café corner"},
		{name: "already clean", in: "item 62", want: "item 62"},
		{name: "interior whitespace survives", in: "  item   62  ", want: "item   62"},
		{name: "empty string stays empty", in: "", want: ""},
		{name: "whitespace only becomes empty", in: " \t\n ", want: ""},
		{name: "long title cut to the cap", in: strings.Repeat("a", 100), want: strings.Repeat("a", 80)},
		{name: "title exactly at the cap survives", in: strings.Repeat("a", 80), want: strings.Repeat("a", 80)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := feedItem{title: tt.in}
			item.clean()
			if item.title != tt.want {
				t.Errorf("clean(%q) = %q, want %q", tt.in, item.title, tt.want)
			}
		})
	}
}

func FuzzCleanTitle(f *testing.F) {
	f.Add("\t item 62 from café corner \n")
	f.Add("item 62")
	f.Add(strings.Repeat("a", 100))

	f.Fuzz(func(t *testing.T, title string) {
		item := feedItem{title: title}
		item.clean()
		got := item.title

		if len(got) > len(title) {
			t.Errorf("clean(%q) grew the title: %d bytes in, %d bytes out", title, len(title), len(got))
		}
		if utf8.ValidString(title) && !utf8.ValidString(got) {
			t.Errorf("clean(%q) produced invalid UTF-8: %q", title, got)
		}
	})
}
