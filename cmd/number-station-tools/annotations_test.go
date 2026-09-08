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
	if err != nil { t.Fatal(err) }
	if err := db.addStation(station{ID: "s1", Name: "Test"}); err != nil { t.Fatal(err) }
	if err := db.addObservation(observation{ID: "o1", StationID: "s1", HeardAt: time.Now().UTC(), FrequencyHz: 5000000}); err != nil { t.Fatal(err) }
	rs, err := openRecordingStore(filepath.Join(t.TempDir(), "recordings.json"))
	if err != nil { t.Fatal(err) }
	if err := rs.add(db, recording{ID: "r1", ObservationID: "o1", Path: "audio/r1.wav", Format: "wav", DurationMS: 10000}); err != nil { t.Fatal(err) }
	return db, rs
}

func TestRecordingAnnotationCRUDAndPersistence(t *testing.T) {
	_, rs := annotationTestStore(t)
	created, err := rs.addAnnotation("r1", recordingAnnotation{StartMS: 1250, Type: annotationTypeCallUp, Label: "Call-up", Notes: "first voice"})
	if err != nil { t.Fatal(err) }
	if created.ID == "" || created.RecordingID != "r1" || created.Type != annotationTypeCallUp { t.Fatalf("unexpected annotation: %#v", created) }
	interval, err := rs.addAnnotation("r1", recordingAnnotation{StartMS: 2000, EndMS: 4500, Type: annotationTypeMessage, Label: "Message"})
	if err != nil { t.Fatal(err) }
	items, err := rs.listAnnotations("r1")
	if err != nil || len(items) != 2 || items[0].ID != created.ID || items[1].ID != interval.ID { t.Fatalf("annotations = %#v err=%v", items, err) }
	updated, err := rs.updateAnnotation("r1", created.ID, recordingAnnotation{StartMS: 1500, Type: annotationTypeStationID, Label: "Station ID", Notes: "clear ID"})
	if err != nil { t.Fatal(err) }
	if updated.ID != created.ID || updated.StartMS != 1500 || updated.Type != annotationTypeStationID || updated.Label != "Station ID" { t.Fatalf("updated = %#v", updated) }
	reopened, err := openRecordingStore(rs.path)
	if err != nil { t.Fatal(err) }
	persisted, err := reopened.listAnnotations("r1")
	if err != nil || len(persisted) != 2 || persisted[0].Type != annotationTypeStationID || persisted[1].Type != annotationTypeMessage { t.Fatalf("persisted = %#v err=%v", persisted, err) }
	if err := reopened.deleteAnnotation("r1", created.ID); err != nil { t.Fatal(err) }
	if err := reopened.delete("r1"); err != nil { t.Fatal(err) }
	if len(reopened.data.Annotations) != 0 { t.Fatalf("recording deletion left annotations: %#v", reopened.data.Annotations) }
}

func TestRecordingAnnotationDefaultsLegacyTypeToOther(t *testing.T) {
	_, rs := annotationTestStore(t)
	created, err := rs.addAnnotation("r1", recordingAnnotation{StartMS: 500, Label: "Legacy-style bookmark"})
	if err != nil { t.Fatal(err) }
	if created.Type != annotationTypeOther { t.Fatalf("default type = %q", created.Type) }
}

func TestRecordingAnnotationValidation(t *testing.T) {
	_, rs := annotationTestStore(t)
	cases := []recordingAnnotation{
		{StartMS: -1, Type: annotationTypeOther, Label: "bad"},
		{StartMS: 9000, EndMS: 8000, Type: annotationTypeOther, Label: "bad"},
		{StartMS: 11000, Type: annotationTypeOther, Label: "past end"},
		{StartMS: 1000, Type: annotationTypeOther, Label: ""},
		{StartMS: 1000, Type: "speech", Label: "invalid type"},
	}
	for _, test := range cases { if _, err := rs.addAnnotation("r1", test); err == nil { t.Fatalf("expected validation error for %#v", test) } }
}

func TestRecordingAnnotationSearch(t *testing.T) {
	_, rs := annotationTestStore(t)
	if _, err := rs.addAnnotation("r1", recordingAnnotation{StartMS: 1000, Type: annotationTypeTone, Label: "Opening tone", Notes: "steady carrier"}); err != nil { t.Fatal(err) }
	if _, err := rs.addAnnotation("r1", recordingAnnotation{StartMS: 3000, Type: annotationTypeMessage, Label: "Message", Notes: "five figure groups"}); err != nil { t.Fatal(err) }

	hits, err := rs.searchAnnotations("carrier", "", 50)
	if err != nil || len(hits) != 1 || hits[0].Annotation.Type != annotationTypeTone || hits[0].Path != "audio/r1.wav" { t.Fatalf("search hits=%#v err=%v", hits, err) }
	hits, err = rs.searchAnnotations("", annotationTypeMessage, 50)
	if err != nil || len(hits) != 1 || hits[0].Annotation.Label != "Message" { t.Fatalf("type hits=%#v err=%v", hits, err) }
	if _, err := rs.searchAnnotations("", "speech", 50); err == nil { t.Fatal("expected invalid type error") }
}

func TestRecordingAnnotationAPI(t *testing.T) {
	_, rs := annotationTestStore(t)
	mux := http.NewServeMux()
	registerAnnotationHandlers(mux, rs)

	body, _ := json.Marshal(recordingAnnotation{StartMS: 1000, EndMS: 2500, Type: annotationTypeTone, Label: "Tone change", Notes: "local note"})
	req := httptest.NewRequest(http.MethodPost, "/api/recordings/r1/annotations", bytes.NewReader(body))
	w := httptest.NewRecorder(); mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated { t.Fatalf("POST status=%d body=%s", w.Code, w.Body.String()) }
	var created recordingAnnotation
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil { t.Fatal(err) }
	if created.Type != annotationTypeTone { t.Fatalf("POST type=%q", created.Type) }

	req = httptest.NewRequest(http.MethodGet, "/api/recordings/r1/annotations", nil)
	w = httptest.NewRecorder(); mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK { t.Fatalf("GET status=%d", w.Code) }
	var items []recordingAnnotation
	if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil || len(items) != 1 || items[0].ID != created.ID || items[0].Type != annotationTypeTone { t.Fatalf("GET annotations=%#v err=%v", items, err) }

	req = httptest.NewRequest(http.MethodGet, "/api/annotations?q=local&type=tone&limit=10", nil)
	w = httptest.NewRecorder(); mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK { t.Fatalf("search status=%d body=%s", w.Code, w.Body.String()) }
	var hits []annotationSearchHit
	if err := json.Unmarshal(w.Body.Bytes(), &hits); err != nil || len(hits) != 1 || hits[0].RecordingID != "r1" { t.Fatalf("search hits=%#v err=%v", hits, err) }

	req = httptest.NewRequest(http.MethodDelete, "/api/recordings/r1/annotations/"+created.ID, nil)
	w = httptest.NewRecorder(); mux.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent { t.Fatalf("DELETE status=%d body=%s", w.Code, w.Body.String()) }
}
