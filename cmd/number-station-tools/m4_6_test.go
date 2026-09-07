package main

import (
	"strings"
	"testing"
)

func TestM46AnnotationCategoriesEmbedded(t *testing.T) {
	annotations, err := webFS.ReadFile("web/annotations.js")
	if err != nil {
		t.Fatal(err)
	}
	css, err := webFS.ReadFile("web/style.css")
	if err != nil {
		t.Fatal(err)
	}

	js := string(annotations)
	for _, want := range []string{
		`"call-up", "station ID", "message", "tone", "noise", "fade", "other"`,
		`name="type"`,
		`data-annotation-filter`,
		`annotation-type-${typeSlug`,
		`type: data.get("type") || "other"`,
		`type: "other", label: "Bookmark"`,
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("annotations.js missing %q", want)
		}
	}

	styles := string(css)
	for _, want := range []string{
		".annotation-type-call-up",
		".annotation-type-station-id",
		".annotation-type-message",
		".annotation-type-tone",
		".annotation-type-noise",
		".annotation-type-fade",
		".annotation-type-other",
		".annotation-filter-row",
	} {
		if !strings.Contains(styles, want) {
			t.Fatalf("style.css missing %q", want)
		}
	}
}
