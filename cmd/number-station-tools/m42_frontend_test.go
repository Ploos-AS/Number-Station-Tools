package main

import (
	"strings"
	"testing"
)

func TestM42LoopWorkbenchIsEmbedded(t *testing.T) {
	b, err := webFS.ReadFile("web/loop.js")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{"data-loop-start", "data-loop-end", "data-loop-toggle", "audio.currentTime = start", "loop-region-active"} {
		if !strings.Contains(s, want) {
			t.Fatalf("loop.js missing %q", want)
		}
	}

	index, err := webFS.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(index), `<script src="/loop.js"></script>`) {
		t.Fatal("index.html does not load loop.js")
	}
}
