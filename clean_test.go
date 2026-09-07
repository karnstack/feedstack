package main

import "testing"

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
