package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStorePersistsM1Data(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	s, err := openStore(path)
	if err != nil { t.Fatal(err) }
	st := station{ID: "e11", Name: "E11 Oblique", Aliases: []string{"E11"}}
	if err := s.addStation(st); err != nil { t.Fatal(err) }
	if err := s.addObservation(observation{ID: "o1", StationID: "e11", HeardAt: time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC), FrequencyHz: 5730000, Mode: "USB"}); err != nil { t.Fatal(err) }
	r, err := openStore(path)
	if err != nil { t.Fatal(err) }
	if len(r.data.Stations) != 1 || r.data.Stations[0].Name != "E11 Oblique" { t.Fatalf("station persistence failed: %#v", r.data.Stations) }
	if len(r.data.Observations) != 1 || r.data.Observations[0].FrequencyHz != 5730000 { t.Fatalf("observation persistence failed: %#v", r.data.Observations) }
}

func TestObservationRequiresExistingStation(t *testing.T) {
	s, err := openStore(filepath.Join(t.TempDir(), "data.json"))
	if err != nil { t.Fatal(err) }
	if err = s.addObservation(observation{ID: "o1", StationID: "missing", FrequencyHz: 4625000}); err == nil { t.Fatal("expected missing station to be rejected") }
}

func TestSchedulePersistsAndRequiresExistingStation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	s, err := openStore(path)
	if err != nil { t.Fatal(err) }
	if err := s.addStation(station{ID: "e11", Name: "E11 Oblique"}); err != nil { t.Fatal(err) }
	v := schedule{ID: "s1", StationID: "e11", FrequencyHz: 5730000, StartUTC: "20:00", EndUTC: "20:15", Weekdays: []int{1, 3, 5}, Mode: "USB"}
	if err := s.addSchedule(v); err != nil { t.Fatal(err) }
	r, err := openStore(path)
	if err != nil { t.Fatal(err) }
	if len(r.data.Schedules) != 1 || r.data.Schedules[0].FrequencyHz != 5730000 { t.Fatalf("schedule persistence failed: %#v", r.data.Schedules) }
	if err := s.addSchedule(schedule{StationID: "missing", FrequencyHz: 1, StartUTC: "00:00", EndUTC: "00:01", Weekdays: []int{1}}); err == nil { t.Fatal("expected missing station to be rejected") }
}

func TestScheduleValidation(t *testing.T) {
	valid := schedule{StationID: "e11", FrequencyHz: 5730000, StartUTC: "20:00", EndUTC: "20:15", Weekdays: []int{1, 7}}
	if err := validateSchedule(valid); err != nil { t.Fatalf("valid schedule rejected: %v", err) }
	cases := []schedule{
		{StationID: "e11", FrequencyHz: 0, StartUTC: "20:00", EndUTC: "20:15", Weekdays: []int{1}},
		{StationID: "e11", FrequencyHz: 1, StartUTC: "25:00", EndUTC: "20:15", Weekdays: []int{1}},
		{StationID: "e11", FrequencyHz: 1, StartUTC: "20:00", EndUTC: "20:15", Weekdays: []int{0}},
		{StationID: "e11", FrequencyHz: 1, StartUTC: "20:00", EndUTC: "20:15", Weekdays: []int{1, 1}},
	}
	for i, tc := range cases { if err := validateSchedule(tc); err == nil { t.Fatalf("case %d should fail", i) } }
}

func TestNowNextIncludesActiveAndUpcoming(t *testing.T) {
	s := &store{data: dataFile{Stations: []station{{ID: "e11", Name: "E11 Oblique"}}, Schedules: []schedule{{ID: "active", StationID: "e11", FrequencyHz: 5730000, StartUTC: "20:00", EndUTC: "20:15", Weekdays: []int{1}}, {ID: "next", StationID: "e11", FrequencyHz: 6925000, StartUTC: "21:00", EndUTC: "21:10", Weekdays: []int{1}}}}}
	got := s.nowNext(time.Date(2026, 9, 7, 20, 5, 0, 0, time.UTC), 5)
	if len(got.Now) != 1 || got.Now[0].ScheduleID != "active" { t.Fatalf("unexpected active schedules: %#v", got.Now) }
	if len(got.Next) == 0 || got.Next[0].ScheduleID != "next" { t.Fatalf("unexpected next schedules: %#v", got.Next) }
	if got.Now[0].StationName != "E11 Oblique" { t.Fatalf("station name missing: %#v", got.Now[0]) }
}

func TestNowNextHandlesOvernightSchedule(t *testing.T) {
	s := &store{data: dataFile{Stations: []station{{ID: "x", Name: "Night Station"}}, Schedules: []schedule{{ID: "night", StationID: "x", FrequencyHz: 1000, StartUTC: "23:55", EndUTC: "00:10", Weekdays: []int{1}}}}}
	got := s.nowNext(time.Date(2026, 9, 8, 0, 5, 0, 0, time.UTC), 5)
	if len(got.Now) != 1 || got.Now[0].ScheduleID != "night" { t.Fatalf("overnight schedule not active: %#v", got.Now) }
	if got.Now[0].End.Day() != 8 { t.Fatalf("overnight end should roll to next UTC day: %v", got.Now[0].End) }
}

func TestUpdateAndDeleteSchedule(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	s, err := openStore(path)
	if err != nil { t.Fatal(err) }
	if err := s.addStation(station{ID: "e11", Name: "E11"}); err != nil { t.Fatal(err) }
	if err := s.addSchedule(schedule{ID: "s1", StationID: "e11", FrequencyHz: 5730000, StartUTC: "20:00", EndUTC: "20:15", Weekdays: []int{1}}); err != nil { t.Fatal(err) }
	if err := s.updateSchedule("s1", schedule{StationID: "e11", FrequencyHz: 6925000, StartUTC: "21:00", EndUTC: "21:10", Weekdays: []int{2}}); err != nil { t.Fatal(err) }
	if s.data.Schedules[0].FrequencyHz != 6925000 || s.data.Schedules[0].ID != "s1" { t.Fatalf("schedule update failed: %#v", s.data.Schedules[0]) }
	if err := s.deleteSchedule("s1"); err != nil { t.Fatal(err) }
	if len(s.data.Schedules) != 0 { t.Fatalf("schedule delete failed: %#v", s.data.Schedules) }
}

func TestStationDeleteProtectsReferences(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	s, err := openStore(path)
	if err != nil { t.Fatal(err) }
	if err := s.addStation(station{ID: "e11", Name: "E11"}); err != nil { t.Fatal(err) }
	if err := s.updateStation("e11", station{Name: "E11 Oblique", Aliases: []string{"E11"}}); err != nil { t.Fatal(err) }
	if s.data.Stations[0].Name != "E11 Oblique" || s.data.Stations[0].ID != "e11" { t.Fatalf("station update failed: %#v", s.data.Stations[0]) }
	if err := s.addSchedule(schedule{ID: "s1", StationID: "e11", FrequencyHz: 5730000, StartUTC: "20:00", EndUTC: "20:15", Weekdays: []int{1}}); err != nil { t.Fatal(err) }
	if err := s.deleteStation("e11"); err == nil { t.Fatal("expected referenced station delete to fail") }
	if err := s.deleteSchedule("s1"); err != nil { t.Fatal(err) }
	if err := s.deleteStation("e11"); err != nil { t.Fatal(err) }
	if len(s.data.Stations) != 0 { t.Fatalf("station delete failed: %#v", s.data.Stations) }
}
