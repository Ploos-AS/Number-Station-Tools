package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRecordingMetadataPersistsAndLinksObservation(t *testing.T) {
	db, err := openStore(filepath.Join(t.TempDir(), "data.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.addStation(station{ID: "s1", Name: "Test"}); err != nil {
		t.Fatal(err)
	}
	if err := db.addObservation(observation{ID: "o1", StationID: "s1", HeardAt: time.Now().UTC(), FrequencyHz: 4625000}); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "recordings.json")
	rs, err := openRecordingStore(path)
	if err != nil {
		t.Fatal(err)
	}
	rec := recording{
		ID:            "r1",
		ObservationID: "o1",
		Path:          "recordings/o1.flac",
		Format:        "FLAC",
		SizeBytes:     12345,
		DurationMS:    30000,
		SampleRateHz:  48000,
		Channels:      1,
		SHA256:        strings.Repeat("a", 64),
	}
	if err := rs.add(db, rec); err != nil {
		t.Fatal(err)
	}

	reloaded, err := openRecordingStore(path)
	if err != nil {
		t.Fatal(err)
	}
	got := reloaded.list()
	if len(got) != 1 {
		t.Fatalf("recordings = %d, want 1", len(got))
	}
	if got[0].ObservationID != "o1" || got[0].Format != "flac" || got[0].SampleRateHz != 48000 {
		t.Fatalf("unexpected recording: %#v", got[0])
	}
}

func TestRecordingValidationAndObservationIntegrity(t *testing.T) {
	db, err := openStore(filepath.Join(t.TempDir(), "data.json"))
	if err != nil {
		t.Fatal(err)
	}
	rs, err := openRecordingStore(filepath.Join(t.TempDir(), "recordings.json"))
	if err != nil {
		t.Fatal(err)
	}

	if err := rs.add(db, recording{ID: "r1", ObservationID: "missing", Path: "x.wav", Format: "wav"}); err == nil {
		t.Fatal("expected missing observation to be rejected")
	}
	if err := validateRecording(recording{ObservationID: "o1", Path: "x.mp3", Format: "mp3"}); err == nil {
		t.Fatal("expected unsupported format rejection")
	}
	if err := validateRecording(recording{ObservationID: "o1", Path: "x.wav", Format: "wav", SHA256: "abc"}); err == nil {
		t.Fatal("expected invalid sha256 rejection")
	}
	if err := validateRecording(recording{ObservationID: "o1", Path: "x.wav", Format: "wav", DurationMS: -1}); err == nil {
		t.Fatal("expected negative metadata rejection")
	}
}

func TestRecordingUpdateDelete(t *testing.T) {
	db, err := openStore(filepath.Join(t.TempDir(), "data.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.addStation(station{ID: "s1", Name: "Test"}); err != nil {
		t.Fatal(err)
	}
	if err := db.addObservation(observation{ID: "o1", StationID: "s1", HeardAt: time.Now().UTC(), FrequencyHz: 1000}); err != nil {
		t.Fatal(err)
	}
	rs, err := openRecordingStore(filepath.Join(t.TempDir(), "recordings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := rs.add(db, recording{ID: "r1", ObservationID: "o1", Path: "a.wav", Format: "wav"}); err != nil {
		t.Fatal(err)
	}
	if !rs.observationReferenced("o1") {
		t.Fatal("expected observation reference")
	}
	if err := rs.update(db, "r1", recording{ObservationID: "o1", Path: "b.flac", Format: "flac", DurationMS: 5000}); err != nil {
		t.Fatal(err)
	}
	if got := rs.list()[0]; got.ID != "r1" || got.Path != "b.flac" || got.DurationMS != 5000 {
		t.Fatalf("unexpected update: %#v", got)
	}
	if err := rs.delete("r1"); err != nil {
		t.Fatal(err)
	}
	if len(rs.list()) != 0 {
		t.Fatal("recording not deleted")
	}
}
