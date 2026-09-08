package main

import (
	"strings"
	"testing"
)

func TestM51ArchiveRestoreEmbedded(t *testing.T) {
	index, err := webFS.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	page := string(index)
	for _, want := range []string{
		"Verify and restore archive",
		"recording-archive-import-form",
		"/archive-restore.js",
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("index missing %q", want)
		}
	}
	js, err := webFS.ReadFile("web/archive-restore.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"/api/recording-archive/import", "imported_recordings", "numberstation:archive-restored"} {
		if !strings.Contains(string(js), want) {
			t.Fatalf("archive restore JS missing %q", want)
		}
	}
}
