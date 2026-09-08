package main

import (
	"strings"
	"testing"
)

func TestM49RecordingBundleEmbedded(t *testing.T) {
	index, err := webFS.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	page := string(index)
	for _, want := range []string{
		"Portable recording manifest",
		"/api/recording-bundle",
		"Export recording manifest",
		"does not contain audio files",
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("index missing %q", want)
		}
	}
}
