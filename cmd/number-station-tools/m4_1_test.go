package main

import (
	"strings"
	"testing"
)

func TestInteractiveSeekingFrontendIsEmbedded(t *testing.T) {
	b, err := webFS.ReadFile("web/playback.js")
	if err != nil {
		t.Fatal(err)
	}
	script := string(b)
	for _, required := range []string{
		"pointerdown",
		"pointermove",
		"setPointerCapture",
		"audio.currentTime = ratio * duration",
		"ArrowLeft",
		"ArrowRight",
		"Home",
		"End",
		"aria-valuenow",
	} {
		if !strings.Contains(script, required) {
			t.Fatalf("playback.js missing interactive seeking token %q", required)
		}
	}
}
