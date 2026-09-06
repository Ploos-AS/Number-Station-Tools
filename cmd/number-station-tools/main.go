package main

import (
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

//go:embed web/*
var webFS embed.FS

type healthResponse struct {
	Status string `json:"status"`
	Time   string `json:"time"`
}

type station struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Aliases   []string `json:"aliases,omitempty"`
	Languages []string `json:"languages,omitempty"`
	Notes     string   `json:"notes,omitempty"`
}

type observation struct {
	ID          string    `json:"id"`
	StationID   string    `json:"station_id"`
	HeardAt     time.Time `json:"heard_at"`
	FrequencyHz int64     `json:"frequency_hz"`
	Mode        string    `json:"mode,omitempty"`
	Signal      string    `json:"signal,omitempty"`
	Message     string    `json:"message,omitempty"`
	Notes       string    `json:"notes,omitempty"`
}

type schedule struct {
	ID          string `json:"id"`
	StationID   string `json:"station_id"`
	FrequencyHz int64  `json:"frequency_hz"`
	StartUTC    string `json:"start_utc"`
	EndUTC      string `json:"end_utc"`
	Weekdays    []int  `json:"weekdays"`
	Mode        string `json:"mode,omitempty"`
	Notes       string `json:"notes,omitempty"`
}

type occurrence struct {
	ScheduleID  string    `json:"schedule_id"`
	StationID   string    `json:"station_id"`
	StationName string    `json:"station_name"`
	FrequencyHz int64     `json:"frequency_hz"`
	Mode        string    `json:"mode,omitempty"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Notes       string    `json:"notes,omitempty"`
}

type nowNextResponse struct {
	At   time.Time    `json:"at"`
	Now  []occurrence `json:"now"`
	Next []occurrence `json:"next"`
}

type dataFile struct {
	Stations     []station     `json:"stations"`
	Observations []observation `json:"observations"`
	Schedules    []schedule    `json:"schedules"`
}

type store struct {
	mu   sync.Mutex
	path string
	data dataFile
}

func openStore(path string) (*store, error) {
	s := &store{path: path, data: dataFile{Stations: []station{}, Observations: []observation{}, Schedules: []schedule{}}}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &s.data); err != nil {
		return nil, err
	}
	if s.data.Stations == nil {
		s.data.Stations = []station{}
	}
	if s.data.Observations == nil {
		s.data.Observations = []observation{}
	}
	if s.data.Schedules == nil {
		s.data.Schedules = []schedule{}
	}
	return s, nil
}

func (s *store) save() error {
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err = os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *store) stationExists(id string) bool {
	for _, st := range s.data.Stations {
		if st.ID == id {
			return true
		}
	}
	return false
}

func (s *store) stationName(id string) string {
	for _, st := range s.data.Stations {
		if st.ID == id {
			return st.Name
		}
	}
	return id
}

func (s *store) addStation(v station) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Stations = append(s.data.Stations, v)
	return s.save()
}

func (s *store) addObservation(v observation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.stationExists(v.StationID) {
		return errors.New("station does not exist")
	}
	s.data.Observations = append(s.data.Observations, v)
	return s.save()
}

func validHHMM(v string) bool {
	_, err := time.Parse("15:04", v)
	return err == nil
}

func validateSchedule(v schedule) error {
	if v.StationID == "" {
		return errors.New("station_id is required")
	}
	if v.FrequencyHz <= 0 {
		return errors.New("frequency_hz must be positive")
	}
	if !validHHMM(v.StartUTC) || !validHHMM(v.EndUTC) {
		return errors.New("start_utc and end_utc must be HH:MM")
	}
	if len(v.Weekdays) == 0 {
		return errors.New("weekdays is required")
	}
	seen := map[int]bool{}
	for _, d := range v.Weekdays {
		if d < 1 || d > 7 {
			return errors.New("weekdays must use ISO values 1..7")
		}
		if seen[d] {
			return errors.New("weekdays must not contain duplicates")
		}
		seen[d] = true
	}
	return nil
}

func (s *store) addSchedule(v schedule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.stationExists(v.StationID) {
		return errors.New("station does not exist")
	}
	if err := validateSchedule(v); err != nil {
		return err
	}
	s.data.Schedules = append(s.data.Schedules, v)
	return s.save()
}

func isoWeekday(t time.Time) int {
	if t.Weekday() == time.Sunday {
		return 7
	}
	return int(t.Weekday())
}

func containsDay(days []int, day int) bool {
	for _, d := range days {
		if d == day {
			return true
		}
	}
	return false
}

func clockOnDay(day time.Time, hhmm string) time.Time {
	parsed, _ := time.Parse("15:04", hhmm)
	return time.Date(day.Year(), day.Month(), day.Day(), parsed.Hour(), parsed.Minute(), 0, 0, time.UTC)
}

func scheduleOccurrence(v schedule, day time.Time, stationName string) occurrence {
	start := clockOnDay(day, v.StartUTC)
	end := clockOnDay(day, v.EndUTC)
	if !end.After(start) {
		end = end.Add(24 * time.Hour)
	}
	return occurrence{
		ScheduleID:  v.ID,
		StationID:   v.StationID,
		StationName: stationName,
		FrequencyHz: v.FrequencyHz,
		Mode:        v.Mode,
		Start:       start,
		End:         end,
		Notes:       v.Notes,
	}
}

func (s *store) nowNext(at time.Time, nextLimit int) nowNextResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	at = at.UTC()
	if nextLimit < 1 {
		nextLimit = 5
	}
	var active []occurrence
	var upcoming []occurrence
	startDay := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC).Add(-24 * time.Hour)
	for offset := 0; offset < 9; offset++ {
		day := startDay.AddDate(0, 0, offset)
		for _, sc := range s.data.Schedules {
			if !containsDay(sc.Weekdays, isoWeekday(day)) {
				continue
			}
			occ := scheduleOccurrence(sc, day, s.stationName(sc.StationID))
			if !at.Before(occ.Start) && at.Before(occ.End) {
				active = append(active, occ)
			} else if occ.Start.After(at) {
				upcoming = append(upcoming, occ)
			}
		}
	}
	sort.Slice(active, func(i, j int) bool { return active[i].Start.Before(active[j].Start) })
	sort.Slice(upcoming, func(i, j int) bool { return upcoming[i].Start.Before(upcoming[j].Start) })
	if len(upcoming) > nextLimit {
		upcoming = upcoming[:nextLimit]
	}
	return nowNextResponse{At: at, Now: active, Next: upcoming}
}

func id() string { return time.Now().UTC().Format("20060102T150405.000000000") }

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func main() {
	addr := getenv("NUMBER_STATION_TOOLS_ADDR", ":8080")
	path := getenv("NUMBER_STATION_TOOLS_DATA", "/data/number-station-tools.json")
	db, err := openStore(path)
	if err != nil {
		log.Fatal(err)
	}
	staticFS, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 200, healthResponse{"ok", time.Now().UTC().Format(time.RFC3339)})
	})
	mux.HandleFunc("GET /api/stations", func(w http.ResponseWriter, _ *http.Request) {
		db.mu.Lock()
		defer db.mu.Unlock()
		out := append([]station(nil), db.data.Stations...)
		sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
		writeJSON(w, 200, out)
	})
	mux.HandleFunc("POST /api/stations", func(w http.ResponseWriter, r *http.Request) {
		var v station
		if json.NewDecoder(r.Body).Decode(&v) != nil || strings.TrimSpace(v.Name) == "" {
			http.Error(w, "invalid station", 400)
			return
		}
		v.ID = id()
		if db.addStation(v) != nil {
			http.Error(w, "store error", 500)
			return
		}
		writeJSON(w, 201, v)
	})
	mux.HandleFunc("GET /api/observations", func(w http.ResponseWriter, _ *http.Request) {
		db.mu.Lock()
		defer db.mu.Unlock()
		writeJSON(w, 200, db.data.Observations)
	})
	mux.HandleFunc("POST /api/observations", func(w http.ResponseWriter, r *http.Request) {
		var v observation
		if json.NewDecoder(r.Body).Decode(&v) != nil || v.StationID == "" || v.FrequencyHz <= 0 {
			http.Error(w, "invalid observation", 400)
			return
		}
		if v.HeardAt.IsZero() {
			v.HeardAt = time.Now().UTC()
		}
		v.ID = id()
		if db.addObservation(v) != nil {
			http.Error(w, "store error", 400)
			return
		}
		writeJSON(w, 201, v)
	})
	mux.HandleFunc("GET /api/schedules", func(w http.ResponseWriter, _ *http.Request) {
		db.mu.Lock()
		defer db.mu.Unlock()
		writeJSON(w, 200, db.data.Schedules)
	})
	mux.HandleFunc("POST /api/schedules", func(w http.ResponseWriter, r *http.Request) {
		var v schedule
		if json.NewDecoder(r.Body).Decode(&v) != nil {
			http.Error(w, "invalid schedule", 400)
			return
		}
		v.ID = id()
		if err := db.addSchedule(v); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, 201, v)
	})
	mux.HandleFunc("GET /api/now-next", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 200, db.nowNext(time.Now().UTC(), 5))
	})
	mux.Handle("/", http.FileServer(http.FS(staticFS)))
	log.Printf("Number Station Tools listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
