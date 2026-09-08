package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestPortableRestorePlanDoesNotMutateState(t *testing.T) {
	db := restoreTestDB(t)
	rs, err := openRecordingStore(filepath.Join(t.TempDir(), "recordings.json"))
	if err != nil {
		t.Fatal(err)
	}
	audioDir := filepath.Join(t.TempDir(), "audio")
	archive := restoreTestArchive(t, testWAV(8000, 1, 1000), nil)

	plan, err := planPortableRestore(bytes.NewReader(archive), db, rs, audioDir)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Import != 1 || plan.Duplicate != 0 || plan.Conflict != 0 || plan.Unmatched != 0 || len(plan.Entries) != 1 {
		t.Fatalf("plan = %#v", plan)
	}
	if plan.Entries[0].Action != "import" || plan.Entries[0].AnnotationsImport != 1 {
		t.Fatalf("entry = %#v", plan.Entries[0])
	}
	if len(rs.list()) != 0 {
		t.Fatal("dry-run changed recording metadata")
	}
	if _, err := os.Stat(filepath.Join(audioDir, "r1.wav")); !os.IsNotExist(err) {
		t.Fatalf("dry-run created destination audio: %v", err)
	}
}

func TestPortableRestorePlanClassifiesDuplicateConflictAndUnmatched(t *testing.T) {
	db := restoreTestDB(t)
	audio := testWAV(8000, 1, 1000)
	archive := restoreTestArchive(t, audio, nil)
	audioDir := filepath.Join(t.TempDir(), "audio")
	rs, _ := openRecordingStore(filepath.Join(t.TempDir(), "recordings.json"))

	if _, err := restorePortableArchive(bytes.NewReader(archive), db, rs, audioDir); err != nil {
		t.Fatal(err)
	}
	plan, err := planPortableRestore(bytes.NewReader(archive), db, rs, audioDir)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Duplicate != 1 || plan.Entries[0].Action != "duplicate" || plan.Entries[0].AnnotationsSkip != 1 {
		t.Fatalf("duplicate plan = %#v", plan)
	}

	rs.data.Recordings[0].Notes = "different"
	plan, err = planPortableRestore(bytes.NewReader(archive), db, rs, audioDir)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Conflict != 1 || plan.Entries[0].Action != "conflict" {
		t.Fatalf("conflict plan = %#v", plan)
	}

	unmatched := restoreTestArchive(t, audio, func(manifest *recordingBundleManifest) {
		manifest.Items[0].Recording.ObservationID = "missing"
	})
	plan, err = planPortableRestore(bytes.NewReader(unmatched), db, &recordingStore{data: recordingData{Recordings: []recording{}, Annotations: []recordingAnnotation{}}}, filepath.Join(t.TempDir(), "audio"))
	if err != nil {
		t.Fatal(err)
	}
	if plan.Unmatched != 1 || plan.Entries[0].Action != "unmatched" {
		t.Fatalf("unmatched plan = %#v", plan)
	}
}

func TestPortableRestorePlanAPI(t *testing.T) {
	db := restoreTestDB(t)
	rs, _ := openRecordingStore(filepath.Join(t.TempDir(), "recordings.json"))
	audioDir := filepath.Join(t.TempDir(), "audio")
	mux := http.NewServeMux()
	registerRecordingRestoreHandler(mux, db, rs, audioDir)
	archive := restoreTestArchive(t, testWAV(8000, 1, 1000), nil)
	req := httptest.NewRequest(http.MethodPost, "/api/recording-archive/plan", bytes.NewReader(archive))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var plan portableRestorePlan
	if err := json.Unmarshal(w.Body.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	if plan.Import != 1 || len(plan.Entries) != 1 {
		t.Fatalf("plan=%#v", plan)
	}
}
