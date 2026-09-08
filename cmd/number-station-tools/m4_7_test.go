package main

import (
	"strings"
	"testing"
)

func TestM47CrossRecordingAnnotationSearchEmbedded(t *testing.T) {
	script, err := webFS.ReadFile("web/annotation-search.js")
	if err != nil {
		t.Fatal(err)
	}
	index, err := webFS.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}

	js := string(script)
	for _, want := range []string{
		"/api/annotations?",
		"data-annotation-search-recording",
		"jumpToHit",
		"audio.currentTime",
		"scrollIntoView",
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("annotation-search.js missing %q", want)
		}
	}

	html := string(index)
	for _, want := range []string{"Annotation index", "annotation-search-form", "annotation-search-results", "annotation-search.js"} {
		if !strings.Contains(html, want) {
			t.Fatalf("index.html missing %q", want)
		}
	}
}
