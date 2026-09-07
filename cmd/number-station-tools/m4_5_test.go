package main

import (
	"strings"
	"testing"
)

func TestM45AnnotationEditingAndDirectCreationEmbedded(t *testing.T) {
	annotations, err := webFS.ReadFile("web/annotations.js")
	if err != nil {
		t.Fatal(err)
	}

	js := string(annotations)
	for _, want := range []string{
		"data-annotation-edit",
		"Update annotation",
		"method: annotationID ? \"PUT\" : \"POST\"",
		"contextmenu",
		"createBookmarkAt",
		"label: \"Bookmark\"",
		"event.key.toLowerCase() !== \"b\"",
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("annotations.js missing %q", want)
		}
	}
}
