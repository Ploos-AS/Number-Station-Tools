package main

import (
	"strings"
	"testing"
)

func TestM52RestorePlanningEmbedded(t *testing.T) {
	index, err := webFS.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	script, err := webFS.ReadFile("web/archive-restore.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"recording-archive-plan", "Plan restore", "recording-archive-plan-results"} {
		if !strings.Contains(string(index), want) {
			t.Fatalf("index missing %q", want)
		}
	}
	for _, want := range []string{"/api/recording-archive/plan", "annotations_import", "annotations_skip", "renderPlan"} {
		if !strings.Contains(string(script), want) {
			t.Fatalf("archive restore script missing %q", want)
		}
	}
}
