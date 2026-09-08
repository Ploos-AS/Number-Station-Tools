package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func restoreTestDB(t *testing.T) *store {
	t.Helper()
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
	return db
}

func restoreTestArchive(t *testing.T, audio []byte, mutate func(*recordingBundleManifest)) []byte {
	t.Helper()
	sum := sha256.Sum256(audio)
	manifest := recordingBundleManifest{Version: recordingBundleVersion, Items: []recordingBundleItem{{
		Recording: recording{
			ID: "r1", ObservationID: "o1", Path: "audio/r1.wav", Format: "wav",
			SizeBytes: int64(len(audio)), DurationMS: 1000, SampleRateHz: 8000, Channels: 1,
			SHA256: hex.EncodeToString(sum[:]), Managed: true, OriginalName: "capture.wav",
		},
		Annotations: []recordingAnnotation{{ID: "a-source", RecordingID: "r1", StartMS: 250, Type: annotationTypeStationID, Label: "ID"}},
	}}}
	if mutate != nil {
		mutate(&manifest)
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "manifest.json", Mode: 0o644, Size: int64(len(manifestBytes))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(manifestBytes); err != nil {
		t.Fatal(err)
	}
	if err := tw.WriteHeader(&tar.Header{Name: "audio/r1.wav", Mode: 0o644, Size: int64(len(audio))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(audio); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestPortableArchiveRestoreAndDuplicateSuppression(t *testing.T) {
	db := restoreTestDB(t)
	rs, err := openRecordingStore(filepath.Join(t.TempDir(), "recordings.json"))
	if err != nil {
		t.Fatal(err)
	}
	audioDir := filepath.Join(t.TempDir(), "audio")
	audio := testWAV(8000, 1, 1000)
	archive := restoreTestArchive(t, audio, nil)

	result, err := restorePortableArchive(bytes.NewReader(archive), db, rs, audioDir)
	if err != nil {
		t.Fatal(err)
	}
	if result.ImportedRecords != 1 || result.ImportedAnnotations != 1 || result.Conflicts != 0 || result.Unmatched != 0 {
		t.Fatalf("restore result = %#v", result)
	}
	rec, ok := rs.byID("r1")
	if !ok || !rec.Managed || len(rec.Waveform) == 0 || len(rec.Fingerprint) == 0 {
		t.Fatalf("restored recording = %#v", rec)
	}
	stored, err := os.ReadFile(filepath.Join(audioDir, "r1.wav"))
	if err != nil || !bytes.Equal(stored, audio) {
		t.Fatalf("restored audio mismatch: err=%v", err)
	}
	annotations, err := rs.listAnnotations("r1")
	if err != nil || len(annotations) != 1 || annotations[0].ID == "a-source" {
		t.Fatalf("restored annotations = %#v err=%v", annotations, err)
	}

	result, err = restorePortableArchive(bytes.NewReader(archive), db, rs, audioDir)
	if err != nil {
		t.Fatal(err)
	}
	if result.ImportedRecords != 0 || result.ImportedAnnotations != 0 || result.Duplicates != 2 {
		t.Fatalf("second restore result = %#v", result)
	}
}

func TestPortableArchiveRestoreRejectsChecksumMismatchAtomically(t *testing.T) {
	db := restoreTestDB(t)
	rs, _ := openRecordingStore(filepath.Join(t.TempDir(), "recordings.json"))
	audioDir := filepath.Join(t.TempDir(), "audio")
	audio := testWAV(8000, 1, 1000)
	archive := restoreTestArchive(t, audio, func(manifest *recordingBundleManifest) {
		manifest.Items[0].Recording.SHA256 = strings.Repeat("0", 64)
	})
	if _, err := restorePortableArchive(bytes.NewReader(archive), db, rs, audioDir); err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("expected checksum error, got %v", err)
	}
	if len(rs.list()) != 0 {
		t.Fatal("failed restore changed recording store")
	}
	if _, err := os.Stat(filepath.Join(audioDir, "r1.wav")); !os.IsNotExist(err) {
		t.Fatalf("failed restore left destination audio: %v", err)
	}
}

func TestPortableArchiveRestoreRejectsIDCollision(t *testing.T) {
	db := restoreTestDB(t)
	rs, _ := openRecordingStore(filepath.Join(t.TempDir(), "recordings.json"))
	rs.data.Recordings = []recording{{ID: "r1", ObservationID: "o1", Path: "external.wav", Format: "wav", SHA256: strings.Repeat("1", 64)}}
	audioDir := filepath.Join(t.TempDir(), "audio")
	result, err := restorePortableArchive(bytes.NewReader(restoreTestArchive(t, testWAV(8000, 1, 1000), nil)), db, rs, audioDir)
	if err != nil {
		t.Fatal(err)
	}
	if result.Conflicts != 1 || result.ImportedRecords != 0 || len(rs.list()) != 1 {
		t.Fatalf("collision result=%#v recordings=%#v", result, rs.list())
	}
}

func TestPortableArchiveRestoreAPI(t *testing.T) {
	db := restoreTestDB(t)
	rs, _ := openRecordingStore(filepath.Join(t.TempDir(), "recordings.json"))
	audioDir := filepath.Join(t.TempDir(), "audio")
	mux := http.NewServeMux()
	registerRecordingRestoreHandler(mux, db, rs, audioDir)
	archive := restoreTestArchive(t, testWAV(8000, 1, 1000), nil)
	req := httptest.NewRequest(http.MethodPost, "/api/recording-archive/import", bytes.NewReader(archive))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var result portableRestoreResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.ImportedRecords != 1 {
		t.Fatalf("result=%#v", result)
	}
}
