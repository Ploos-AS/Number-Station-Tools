package main

import (
	"strings"
	"testing"
)

func TestM48AnnotationTransferFrontendIsEmbedded(t *testing.T) {
	index, err := webFS.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	script, err := webFS.ReadFile("web/annotation-transfer.js")
	if err != nil {
		t.Fatal(err)
	}

	html := string(index)
	for _, want := range []string{
		"/api/annotations/export?format=json",
		"/api/annotations/export?format=csv",
		`id="annotation-import-form"`,
		`<script src="/annotation-transfer.js"></script>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("index.html missing %q", want)
		}
	}

	js := string(script)
	for _, want := range []string{
		`fetch("/api/annotations/import"`,
		`method: "POST"`,
		"result.imported",
		"result.duplicates",
		"result.unmatched",
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("annotation-transfer.js missing %q", want)
		}
	}
}
