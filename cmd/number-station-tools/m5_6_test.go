package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRestoreReceiptHashChain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receipts.json")
	store, err := openRestoreReceiptStore(path)
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.add(restoreReceipt{Mode: "full", ImportedRecordings: 1})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.add(restoreReceipt{Mode: "selected", PlanToken: strings.Repeat("a", 64), SelectedRecordingIDs: []string{"r2", "r1"}, ImportedRecordings: 1})
	if err != nil {
		t.Fatal(err)
	}
	if first.PreviousHash != "" || len(first.ReceiptHash) != 64 {
		t.Fatalf("first receipt chain fields=%#v", first)
	}
	if second.PreviousHash != first.ReceiptHash || len(second.ReceiptHash) != 64 {
		t.Fatalf("second receipt chain fields=%#v first=%#v", second, first)
	}
	verification := store.verify()
	if !verification.Valid || verification.Count != 2 || verification.HeadHash != second.ReceiptHash {
		t.Fatalf("verification=%#v", verification)
	}
}

func TestRestoreReceiptHashChainDetectsTampering(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receipts.json")
	store, err := openRestoreReceiptStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.add(restoreReceipt{Mode: "full", ImportedRecordings: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.add(restoreReceipt{Mode: "full", ImportedRecordings: 2}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var data restoreReceiptData
	if err := json.Unmarshal(body, &data); err != nil {
		t.Fatal(err)
	}
	data.Receipts[0].ImportedRecordings = 99
	body, _ = json.MarshalIndent(data, "", "  ")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	tampered, err := openRestoreReceiptStore(path)
	if err != nil {
		t.Fatal(err)
	}
	verification := tampered.verify()
	if verification.Valid || verification.FailedIndex != 0 || verification.Error == "" {
		t.Fatalf("verification=%#v", verification)
	}
	if _, err := tampered.add(restoreReceipt{Mode: "full"}); err == nil || !strings.Contains(err.Error(), "integrity") {
		t.Fatalf("expected integrity rejection, got %v", err)
	}
}

func TestRestoreReceiptLegacyMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receipts.json")
	legacy := restoreReceiptData{Receipts: []restoreReceipt{
		{ID: "old-1", At: "2026-01-01T00:00:00Z", Mode: "full", ImportedRecordings: 1},
		{ID: "old-2", At: "2026-01-02T00:00:00Z", Mode: "full", ImportedRecordings: 2},
	}}
	body, _ := json.MarshalIndent(legacy, "", "  ")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := openRestoreReceiptStore(path)
	if err != nil {
		t.Fatal(err)
	}
	verification := store.verify()
	if !verification.Valid || verification.Count != 2 || verification.HeadHash == "" {
		t.Fatalf("verification=%#v", verification)
	}
	reopened, err := openRestoreReceiptStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reopened.verify().Valid {
		t.Fatal("migrated receipt chain did not persist")
	}
}

func TestRestoreReceiptVerifyAPI(t *testing.T) {
	db := restoreTestDB(t)
	recordingPath := filepath.Join(t.TempDir(), "recordings.json")
	rs, _ := openRecordingStore(recordingPath)
	receipts, err := openRestoreReceiptStore(recordingPath + ".restore-receipts.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := receipts.add(restoreReceipt{Mode: "full", ImportedRecordings: 1}); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	registerRecordingRestoreHandler(mux, db, rs, filepath.Join(t.TempDir(), "audio"))
	req := httptest.NewRequest(http.MethodGet, "/api/recording-archive/receipts/verify", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var verification restoreReceiptVerification
	if err := json.Unmarshal(w.Body.Bytes(), &verification); err != nil {
		t.Fatal(err)
	}
	if !verification.Valid || verification.Count != 1 || len(verification.HeadHash) != 64 {
		t.Fatalf("verification=%#v", verification)
	}
}
