package main

import (
	"strings"
	"testing"
)

func TestM44VisualAnnotationMarkersEmbedded(t *testing.T) {
	annotations, err := webFS.ReadFile("web/annotations.js")
	if err != nil {
		t.Fatal(err)
	}
	css, err := webFS.ReadFile("web/style.css")
	if err != nil {
		t.Fatal(err)
	}
	index, err := webFS.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}

	js := string(annotations)
	for _, want := range []string{
		"annotation-overlay",
		"annotation-point",
		"annotation-interval",
		"data-annotation-marker",
		"audio.currentTime = Number(marker.dataset.annotationMarker) / 1000",
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("annotations.js missing %q", want)
		}
	}

	styles := string(css)
	for _, want := range []string{".annotation-overlay", ".annotation-marker", ".annotation-interval"} {
		if !strings.Contains(styles, want) {
			t.Fatalf("style.css missing %q", want)
		}
	}

	if !strings.Contains(string(index), "annotations.js") {
		t.Fatal("index does not load annotation marker UI")
	}
}
