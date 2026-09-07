package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestRecordingAPILifecycle(t *testing.T) {
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
	rs, err := openRecordingStore(filepath.Join(t.TempDir(), "recordings.json"))
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	registerRecordingHandlers(mux, db, rs)

	payload := recording{ObservationID: "o1", Path: "recordings/o1.flac", Format: "flac", DurationMS: 12000, SampleRateHz: 48000, Channels: 1}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/recordings", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, body=%s", w.Code, w.Body.String())
	}
	var created recording
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.ObservationID != "o1" {
		t.Fatalf("unexpected created recording: %#v", created)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/observations/o1/recordings", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET status = %d", w.Code)
	}
	var listed []recording
	if err := json.Unmarshal(w.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("unexpected list: %#v", listed)
	}

	payload.Path = "recordings/o1.wav"
	payload.Format = "wav"
	body, _ = json.Marshal(payload)
	req = httptest.NewRequest(http.MethodPut, "/api/recordings/"+created.ID, bytes.NewReader(body))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/recordings/"+created.ID, nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("DELETE status = %d, body=%s", w.Code, w.Body.String())
	}
	if got := rs.list(); len(got) != 0 {
		t.Fatalf("recordings remain: %#v", got)
	}
}
