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

func TestPortableRestorePlanTokenDeterministic(t *testing.T) {
	db := restoreTestDB(t)
	rs, _ := openRecordingStore(filepath.Join(t.TempDir(), "recordings.json"))
	audioDir := filepath.Join(t.TempDir(), "audio")
	archive := selectiveRestoreArchive(t)

	first, err := planPortableRestore(bytes.NewReader(archive), db, rs, audioDir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := planPortableRestore(bytes.NewReader(archive), db, rs, audioDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.PlanToken) != 64 || first.PlanToken != second.PlanToken {
		t.Fatalf("plan tokens first=%q second=%q", first.PlanToken, second.PlanToken)
	}
}

func TestPortableRestorePlanTokenRejectsLocalStateDrift(t *testing.T) {
	db := restoreTestDB(t)
	rs, _ := openRecordingStore(filepath.Join(t.TempDir(), "recordings.json"))
	audioDir := filepath.Join(t.TempDir(), "audio")
	archive := selectiveRestoreArchive(t)
	plan, err := planPortableRestore(bytes.NewReader(archive), db, rs, audioDir)
	if err != nil {
		t.Fatal(err)
	}

	rs.mu.Lock()
	rs.data.Annotations = append(rs.data.Annotations, recordingAnnotation{ID: "local-drift", RecordingID: "other", StartMS: 1, Label: "changed"})
	rs.mu.Unlock()

	_, err = restorePortableArchiveSelectedWithPlanToken(bytes.NewReader(archive), db, rs, audioDir, []string{"r1"}, plan.PlanToken)
	if err == nil || !strings.Contains(err.Error(), "plan changed") {
		t.Fatalf("expected stale plan rejection, got %v", err)
	}
	if len(rs.list()) != 0 {
		t.Fatal("stale plan changed recording store")
	}
}

func TestPortableRestorePlanTokenRejectsArchiveDrift(t *testing.T) {
	db := restoreTestDB(t)
	rs, _ := openRecordingStore(filepath.Join(t.TempDir(), "recordings.json"))
	audioDir := filepath.Join(t.TempDir(), "audio")
	archive := selectiveRestoreArchive(t)
	plan, err := planPortableRestore(bytes.NewReader(archive), db, rs, audioDir)
	if err != nil {
		t.Fatal(err)
	}
	changed := restoreTestArchive(t, testWAV(8000, 1, 1000), func(manifest *recordingBundleManifest) {
		manifest.Items[0].Recording.Notes = "changed archive metadata"
	})
	_, err = restorePortableArchiveSelectedWithPlanToken(bytes.NewReader(changed), db, rs, audioDir, []string{"r1"}, plan.PlanToken)
	if err == nil || !strings.Contains(err.Error(), "plan changed") {
		t.Fatalf("expected archive drift rejection, got %v", err)
	}
}

func TestPortableRestorePlanTokenAPI(t *testing.T) {
	db := restoreTestDB(t)
	rs, _ := openRecordingStore(filepath.Join(t.TempDir(), "recordings.json"))
	audioDir := filepath.Join(t.TempDir(), "audio")
	mux := http.NewServeMux()
	registerRecordingRestoreHandler(mux, db, rs, audioDir)
	archive := selectiveRestoreArchive(t)

	planReq := httptest.NewRequest(http.MethodPost, "/api/recording-archive/plan", bytes.NewReader(archive))
	planW := httptest.NewRecorder()
	mux.ServeHTTP(planW, planReq)
	if planW.Code != http.StatusOK {
		t.Fatalf("plan status=%d body=%s", planW.Code, planW.Body.String())
	}
	var plan portableRestorePlan
	if err := json.Unmarshal(planW.Body.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.PlanToken) != 64 {
		t.Fatalf("plan token=%q", plan.PlanToken)
	}

	missingReq := httptest.NewRequest(http.MethodPost, "/api/recording-archive/import-selected?recording_id=r1", bytes.NewReader(archive))
	missingW := httptest.NewRecorder()
	mux.ServeHTTP(missingW, missingReq)
	if missingW.Code != http.StatusPreconditionRequired {
		t.Fatalf("missing token status=%d body=%s", missingW.Code, missingW.Body.String())
	}

	query := "/api/recording-archive/import-selected?recording_id=r1&plan_token=" + plan.PlanToken
	req := httptest.NewRequest(http.MethodPost, query, bytes.NewReader(archive))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestM54RestorePlanTokenEmbedded(t *testing.T) {
	js, err := webFS.ReadFile("web/archive-restore.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"plan.plan_token", "plan_token", "Plan token", "approved plan"} {
		if !strings.Contains(string(js), want) {
			t.Fatalf("archive restore JS missing %q", want)
		}
	}
}
