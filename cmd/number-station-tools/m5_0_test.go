package main

import (
	"strings"
	"testing"
)

func TestM50PortableArchiveEmbedded(t *testing.T) {
	index, err := webFS.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	html := string(index)
	for _, want := range []string{
		"/api/recording-archive",
		"Export full archive",
		"manifest.json",
		"managed WAV/FLAC",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("index missing %q", want)
		}
	}
}
