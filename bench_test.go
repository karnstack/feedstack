package main

import (
	"strings"
	"testing"
)

var benchTitles = []string{
	"go ships",
	"the quiet week",
	"what the race detector saw next",
	"errors are values, and so are titles",
	"the snowball melts in milliseconds, again",
}

func sampleTitles(n int) []string {
	titles := make([]string, n)
	for i := range titles {
		titles[i] = benchTitles[i%len(benchTitles)]
	}
	return titles
}

func joinPlus(titles []string) string {
	out := ""
	for _, t := range titles {
		line := t + "\n"
		out += line
	}
	return out
}

func joinBuilder(titles []string) string {
	var b strings.Builder
	for _, t := range titles {
		b.WriteString(t)
		b.WriteString("\n")
	}
	return b.String()
}

func BenchmarkJoinPlus(b *testing.B) {
	titles := sampleTitles(50)
	b.ReportAllocs()
	for b.Loop() {
		joinPlus(titles)
	}
}

func BenchmarkJoinBuilder(b *testing.B) {
	titles := sampleTitles(50)
	b.ReportAllocs()
	for b.Loop() {
		joinBuilder(titles)
	}
}
