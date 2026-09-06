package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStorePersistsM1Data(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	s, err := openStore(path)
	if err != nil {
		t.Fatal(err)
	}
	st := station{ID: "e11", Name: "E11 Oblique", Aliases: []string{"E11"}}
	if err := s.addStation(st); err != nil {
		t.Fatal(err)
	}
	if err := s.addObservation(observation{ID: "o1", StationID: "e11", HeardAt: time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC), FrequencyHz: 5730000, Mode: "USB"}); err != nil {
		t.Fatal(err)
	}
	r, err := openStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.data.Stations) != 1 || r.data.Stations[0].Name != "E11 Oblique" {
		t.Fatalf("station persistence failed: %#v", r.data.Stations)
	}
	if len(r.data.Observations) != 1 || r.data.Observations[0].FrequencyHz != 5730000 {
		t.Fatalf("observation persistence failed: %#v", r.data.Observations)
	}
}

func TestObservationRequiresExistingStation(t *testing.T) {
	s, err := openStore(filepath.Join(t.TempDir(), "data.json"))
	if err != nil {
		t.Fatal(err)
	}
	err = s.addObservation(observation{ID: "o1", StationID: "missing", FrequencyHz: 4625000})
	if err == nil {
		t.Fatal("expected missing station to be rejected")
	}
}
