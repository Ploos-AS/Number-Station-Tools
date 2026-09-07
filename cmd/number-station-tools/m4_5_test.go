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
	index, err := webFS.ReadFile("web/index.html")
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

	if !strings.Contains(string(index), "Number Station Tools M4.5") {
		t.Fatal("index does not identify M4.5")
	}
}
