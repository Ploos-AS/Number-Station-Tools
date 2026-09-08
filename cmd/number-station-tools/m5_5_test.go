package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRestoreReceiptStorePersistsAndSorts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receipts.json")
	store, err := openRestoreReceiptStore(path)
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.add(restoreReceipt{Mode: "full", ImportedRecordings: 1})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.add(restoreReceipt{Mode: "selected", PlanToken: strings.Repeat("a", 64), SelectedRecordingIDs: []string{"r2", "r1"}, ImportedRecordings: 2})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == "" || first.At == "" || second.ID == "" || second.At == "" {
		t.Fatalf("missing receipt identity: first=%#v second=%#v", first, second)
	}
	if len(second.SelectedRecordingIDs) != 2 || second.SelectedRecordingIDs[0] != "r1" {
		t.Fatalf("selected ids not normalized: %#v", second.SelectedRecordingIDs)
	}
	reopened, err := openRestoreReceiptStore(path)
	if err != nil {
		t.Fatal(err)
	}
	items := reopened.list(10)
	if len(items) != 2 {
		t.Fatalf("receipts=%#v", items)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestRestoreReceiptAPI(t *testing.T) {
	db := restoreTestDB(t)
	recordingPath := filepath.Join(t.TempDir(), "recordings.json")
	rs, _ := openRecordingStore(recordingPath)
	audioDir := filepath.Join(t.TempDir(), "audio")
	mux := http.NewServeMux()
	registerRecordingRestoreHandler(mux, db, rs, audioDir)

	archive := restoreTestArchive(t, testWAV(8000, 1, 1000), nil)
	req := httptest.NewRequest(http.MethodPost, "/api/recording-archive/import", bytes.NewReader(archive))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("restore status=%d body=%s", w.Code, w.Body.String())
	}
	var restored restoreResponse
	if err := json.Unmarshal(w.Body.Bytes(), &restored); err != nil {
		t.Fatal(err)
	}
	if restored.ImportedRecords != 1 || restored.Receipt.ID == "" || restored.Receipt.Mode != "full" {
		t.Fatalf("restore response=%#v", restored)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/recording-archive/receipts?limit=10", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("receipts status=%d body=%s", w.Code, w.Body.String())
	}
	var receipts []restoreReceipt
	if err := json.Unmarshal(w.Body.Bytes(), &receipts); err != nil {
		t.Fatal(err)
	}
	if len(receipts) != 1 || receipts[0].ID != restored.Receipt.ID {
		t.Fatalf("receipts=%#v restored=%#v", receipts, restored.Receipt)
	}
}

func TestRestoreReceiptLimitValidation(t *testing.T) {
	db := restoreTestDB(t)
	rs, _ := openRecordingStore(filepath.Join(t.TempDir(), "recordings.json"))
	mux := http.NewServeMux()
	registerRecordingRestoreHandler(mux, db, rs, filepath.Join(t.TempDir(), "audio"))
	req := httptest.NewRequest(http.MethodGet, "/api/recording-archive/receipts?limit=999", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
