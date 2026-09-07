package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManagedAudioUploadAndDownload(t *testing.T) {
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
	audioDir := filepath.Join(t.TempDir(), "audio")
	mux := http.NewServeMux()
	registerAudioHandlers(mux, db, rs, audioDir)

	audio := []byte("RIFF-test-audio")
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if err := mw.WriteField("observation_id", "o1"); err != nil {
		t.Fatal(err)
	}
	if err := mw.WriteField("notes", "test upload"); err != nil {
		t.Fatal(err)
	}
	part, err := mw.CreateFormFile("file", "capture.WAV")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(audio); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/audio", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("upload status = %d body=%s", w.Code, w.Body.String())
	}
	var rec recording
	if err := json.Unmarshal(w.Body.Bytes(), &rec); err != nil {
		t.Fatal(err)
	}
	if !rec.Managed || rec.Format != "wav" || rec.OriginalName != "capture.WAV" || rec.SizeBytes != int64(len(audio)) {
		t.Fatalf("unexpected recording: %#v", rec)
	}
	sum := sha256.Sum256(audio)
	if rec.SHA256 != hex.EncodeToString(sum[:]) {
		t.Fatalf("sha256 = %q", rec.SHA256)
	}
	stored := filepath.Join(audioDir, filepath.Base(rec.Path))
	got, err := os.ReadFile(stored)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, audio) {
		t.Fatal("stored audio differs")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/recordings/"+rec.ID+"/file", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !bytes.Equal(w.Body.Bytes(), audio) {
		t.Fatalf("download status=%d body=%q", w.Code, w.Body.Bytes())
	}
}

func TestManagedAudioRejectsUnsupportedExtension(t *testing.T) {
	db, _ := openStore(filepath.Join(t.TempDir(), "data.json"))
	_ = db.addStation(station{ID: "s1", Name: "Test"})
	_ = db.addObservation(observation{ID: "o1", StationID: "s1", HeardAt: time.Now().UTC(), FrequencyHz: 1000})
	rs, _ := openRecordingStore(filepath.Join(t.TempDir(), "recordings.json"))
	mux := http.NewServeMux()
	registerAudioHandlers(mux, db, rs, filepath.Join(t.TempDir(), "audio"))

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("observation_id", "o1")
	part, _ := mw.CreateFormFile("file", "capture.mp3")
	_, _ = part.Write([]byte("not supported"))
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/audio", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", w.Code)
	}
}
