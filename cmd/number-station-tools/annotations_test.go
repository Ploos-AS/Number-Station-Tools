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

func annotationTestStore(t *testing.T) (*store, *recordingStore) {
	t.Helper()
	db, err := openStore(filepath.Join(t.TempDir(), "data.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.addStation(station{ID: "s1", Name: "Test"}); err != nil {
		t.Fatal(err)
	}
	if err := db.addObservation(observation{ID: "o1", StationID: "s1", HeardAt: time.Now().UTC(), FrequencyHz: 5000000}); err != nil {
		t.Fatal(err)
	}
	rs, err := openRecordingStore(filepath.Join(t.TempDir(), "recordings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := rs.add(db, recording{ID: "r1", ObservationID: "o1", Path: "audio/r1.wav", Format: "wav", DurationMS: 10000}); err != nil {
		t.Fatal(err)
	}
	return db, rs
}

func TestRecordingAnnotationCRUDAndPersistence(t *testing.T) {
	_, rs := annotationTestStore(t)
	created, err := rs.addAnnotation("r1", recordingAnnotation{StartMS: 1250, Label: "Call-up", Notes: "first voice"})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.RecordingID != "r1" {
		t.Fatalf("unexpected annotation: %#v", created)
	}
	interval, err := rs.addAnnotation("r1", recordingAnnotation{StartMS: 2000, EndMS: 4500, Label: "Message"})
	if err != nil {
		t.Fatal(err)
	}
	items, err := rs.listAnnotations("r1")
	if err != nil || len(items) != 2 || items[0].ID != created.ID || items[1].ID != interval.ID {
		t.Fatalf("annotations = %#v err=%v", items, err)
	}
	updated, err := rs.updateAnnotation("r1", created.ID, recordingAnnotation{StartMS: 1500, Label: "Station ID", Notes: "clear ID"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != created.ID || updated.StartMS != 1500 || updated.Label != "Station ID" {
		t.Fatalf("updated = %#v", updated)
	}
	reopened, err := openRecordingStore(rs.path)
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := reopened.listAnnotations("r1")
	if err != nil || len(persisted) != 2 {
		t.Fatalf("persisted = %#v err=%v", persisted, err)
	}
	if err := reopened.deleteAnnotation("r1", created.ID); err != nil {
		t.Fatal(err)
	}
	if err := reopened.delete("r1"); err != nil {
		t.Fatal(err)
	}
	if len(reopened.data.Annotations) != 0 {
		t.Fatalf("recording deletion left annotations: %#v", reopened.data.Annotations)
	}
}

func TestRecordingAnnotationValidation(t *testing.T) {
	_, rs := annotationTestStore(t)
	cases := []recordingAnnotation{
		{StartMS: -1, Label: "bad"},
		{StartMS: 9000, EndMS: 8000, Label: "bad"},
		{StartMS: 11000, Label: "past end"},
		{StartMS: 1000, Label: ""},
	}
	for _, test := range cases {
		if _, err := rs.addAnnotation("r1", test); err == nil {
			t.Fatalf("expected validation error for %#v", test)
		}
	}
}

func TestRecordingAnnotationAPI(t *testing.T) {
	_, rs := annotationTestStore(t)
	mux := http.NewServeMux()
	registerAnnotationHandlers(mux, rs)

	body, _ := json.Marshal(recordingAnnotation{StartMS: 1000, EndMS: 2500, Label: "Tone change", Notes: "local note"})
	req := httptest.NewRequest(http.MethodPost, "/api/recordings/r1/annotations", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST status=%d body=%s", w.Code, w.Body.String())
	}
	var created recordingAnnotation
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/recordings/r1/annotations", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET status=%d", w.Code)
	}
	var items []recordingAnnotation
	if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil || len(items) != 1 || items[0].ID != created.ID {
		t.Fatalf("GET annotations=%#v err=%v", items, err)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/recordings/r1/annotations/"+created.ID, nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("DELETE status=%d body=%s", w.Code, w.Body.String())
	}
}
