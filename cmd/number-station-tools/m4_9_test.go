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
		"/api/recording-bundle",
		"Export manifest only",
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("index missing %q", want)
		}
	}
}
