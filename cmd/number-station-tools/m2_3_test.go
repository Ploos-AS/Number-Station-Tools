package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestObservationScheduleLinkIntegrity(t *testing.T) {
	s, err := openStore(filepath.Join(t.TempDir(), "data.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.addStation(station{ID: "a", Name: "A"}); err != nil {
		t.Fatal(err)
	}
	if err := s.addStation(station{ID: "b", Name: "B"}); err != nil {
		t.Fatal(err)
	}
	if err := s.addSchedule(schedule{ID: "s1", StationID: "a", FrequencyHz: 1000, StartUTC: "10:00", EndUTC: "10:05", Weekdays: []int{1}}); err != nil {
		t.Fatal(err)
	}
	if err := s.addObservation(observation{ID: "o1", StationID: "a", ScheduleID: "s1", HeardAt: time.Now().UTC(), FrequencyHz: 1000}); err != nil {
		t.Fatalf("valid link rejected: %v", err)
	}
	if err := s.addObservation(observation{ID: "o2", StationID: "b", ScheduleID: "s1", HeardAt: time.Now().UTC(), FrequencyHz: 1000}); err == nil {
		t.Fatal("expected cross-station schedule link rejection")
	}
	if err := s.addObservation(observation{ID: "o3", StationID: "a", ScheduleID: "missing", HeardAt: time.Now().UTC(), FrequencyHz: 1000}); err == nil {
		t.Fatal("expected missing schedule rejection")
	}
}

func TestObservationUpdateDeleteAndScheduleGuard(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	s, err := openStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.addStation(station{ID: "a", Name: "A"}); err != nil {
		t.Fatal(err)
	}
	if err := s.addSchedule(schedule{ID: "s1", StationID: "a", FrequencyHz: 1000, StartUTC: "10:00", EndUTC: "10:05", Weekdays: []int{1}}); err != nil {
		t.Fatal(err)
	}
	if err := s.addObservation(observation{ID: "o1", StationID: "a", ScheduleID: "s1", HeardAt: time.Now().UTC(), FrequencyHz: 1000}); err != nil {
		t.Fatal(err)
	}
	if err := s.deleteSchedule("s1"); err == nil {
		t.Fatal("expected linked schedule delete to be blocked")
	}
	updated := observation{StationID: "a", HeardAt: time.Now().UTC(), FrequencyHz: 2000, Signal: "S9"}
	if err := s.updateObservation("o1", updated); err != nil {
		t.Fatal(err)
	}
	if s.data.Observations[0].ID != "o1" || s.data.Observations[0].FrequencyHz != 2000 || s.data.Observations[0].ScheduleID != "" {
		t.Fatalf("unexpected update: %#v", s.data.Observations[0])
	}
	if err := s.deleteSchedule("s1"); err != nil {
		t.Fatalf("unlinked schedule should delete: %v", err)
	}
	if err := s.deleteObservation("o1"); err != nil {
		t.Fatal(err)
	}
	if len(s.data.Observations) != 0 {
		t.Fatalf("observation not deleted: %#v", s.data.Observations)
	}
}
