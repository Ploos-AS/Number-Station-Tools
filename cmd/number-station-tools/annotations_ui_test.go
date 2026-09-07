package main

import (
	"strings"
	"testing"
)

func TestRecordingAnnotationUIEmbedded(t *testing.T) {
	b, err := webFS.ReadFile("web/annotations.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(b)
	for _, marker := range []string{"/annotations", "data-annotation-current", "data-annotation-seek", "data-annotation-delete", "Save annotation"} {
		if !strings.Contains(js, marker) {
			t.Fatalf("annotations.js missing %q", marker)
		}
	}
	index, err := webFS.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(index), `<script src="/annotations.js"></script>`) {
		t.Fatal("index does not load annotations.js")
	}
}
