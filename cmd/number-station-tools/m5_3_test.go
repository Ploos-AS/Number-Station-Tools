package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func selectiveRestoreArchive(t *testing.T) []byte {
	t.Helper()
	return restoreTestArchive(t, testWAV(8000, 1, 1000), func(manifest *recordingBundleManifest) {
		manifest.Items = append(manifest.Items, recordingBundleItem{Recording: recording{
			ID: "r2", ObservationID: "o1", Path: "external/r2.wav", Format: "wav",
			SizeBytes: 1, DurationMS: 1, SampleRateHz: 8000, Channels: 1,
		}})
	})
}

func TestPortableArchiveSelectiveRestore(t *testing.T) {
	db := restoreTestDB(t)
	rs, _ := openRecordingStore(filepath.Join(t.TempDir(), "recordings.json"))
	audioDir := filepath.Join(t.TempDir(), "audio")
	archive := selectiveRestoreArchive(t)

	result, err := restorePortableArchiveSelected(bytes.NewReader(archive), db, rs, audioDir, []string{"r1"})
	if err != nil {
		t.Fatal(err)
	}
	if result.ImportedRecords != 1 || result.ImportedAnnotations != 1 || result.Selected != 1 || result.SkippedByPolicy != 1 {
		t.Fatalf("result=%#v", result)
	}
	if _, ok := rs.byID("r1"); !ok {
		t.Fatal("selected recording was not imported")
	}
	if _, ok := rs.byID("r2"); ok {
		t.Fatal("unselected recording was imported")
	}
}

func TestPortableArchiveSelectiveRestoreRejectsUnsafeSelections(t *testing.T) {
	db := restoreTestDB(t)
	rs, _ := openRecordingStore(filepath.Join(t.TempDir(), "recordings.json"))
	audioDir := filepath.Join(t.TempDir(), "audio")
	archive := selectiveRestoreArchive(t)

	if _, err := restorePortableArchiveSelected(bytes.NewReader(archive), db, rs, audioDir, []string{"missing"}); err == nil || !strings.Contains(err.Error(), "not present") {
		t.Fatalf("expected unknown selection error, got %v", err)
	}
	if _, err := restorePortableArchiveSelected(bytes.NewReader(archive), db, rs, audioDir, nil); err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("expected empty selection error, got %v", err)
	}

	if _, err := restorePortableArchiveSelected(bytes.NewReader(archive), db, rs, audioDir, []string{"r1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := restorePortableArchiveSelected(bytes.NewReader(archive), db, rs, audioDir, []string{"r1"}); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate selection rejection, got %v", err)
	}
}

func TestPortableArchiveSelectiveRestoreAPI(t *testing.T) {
	db := restoreTestDB(t)
	rs, _ := openRecordingStore(filepath.Join(t.TempDir(), "recordings.json"))
	audioDir := filepath.Join(t.TempDir(), "audio")
	mux := http.NewServeMux()
	registerRecordingRestoreHandler(mux, db, rs, audioDir)

	req := httptest.NewRequest(http.MethodPost, "/api/recording-archive/import-selected?recording_id=r1", bytes.NewReader(selectiveRestoreArchive(t)))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var result portableSelectiveRestoreResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.ImportedRecords != 1 || result.Selected != 1 {
		t.Fatalf("result=%#v", result)
	}
}

func TestM53SelectiveRestoreEmbedded(t *testing.T) {
	js, err := webFS.ReadFile("web/archive-restore.js")
	if err != nil {
		t.Fatal(err)
	}
	text := string(js)
	for _, want := range []string{"Restore selected", "/api/recording-archive/import-selected?", "data-restore-recording-id", "skipped_by_policy"} {
		if !strings.Contains(text, want) {
			t.Fatalf("archive restore JS missing %q", want)
		}
	}
}
