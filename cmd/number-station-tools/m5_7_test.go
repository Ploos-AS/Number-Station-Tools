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

func TestRestoreReceiptAnchorExactAndExtended(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receipts.json")
	store, err := openRestoreReceiptStore(path)
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.add(restoreReceipt{Mode: "full", ImportedRecordings: 1})
	if err != nil {
		t.Fatal(err)
	}
	anchor, err := store.anchor()
	if err != nil {
		t.Fatal(err)
	}
	if anchor.Version != restoreReceiptAnchorVersion || anchor.ReceiptCount != 1 || anchor.HeadHash != first.ReceiptHash {
		t.Fatalf("anchor=%#v first=%#v", anchor, first)
	}
	exact := store.verifyAnchor(anchor)
	if !exact.Valid || exact.Status != "match" || exact.CurrentHeadHash != first.ReceiptHash {
		t.Fatalf("exact=%#v", exact)
	}
	second, err := store.add(restoreReceipt{Mode: "full", ImportedRecordings: 2})
	if err != nil {
		t.Fatal(err)
	}
	extended := store.verifyAnchor(anchor)
	if !extended.Valid || extended.Status != "extended" || extended.LocalCount != 2 || extended.CurrentHeadHash != second.ReceiptHash {
		t.Fatalf("extended=%#v", extended)
	}
}

func TestRestoreReceiptAnchorRejectsRewrittenPrefix(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receipts.json")
	store, err := openRestoreReceiptStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.add(restoreReceipt{Mode: "full", ImportedRecordings: 1}); err != nil {
		t.Fatal(err)
	}
	anchor, err := store.anchor()
	if err != nil {
		t.Fatal(err)
	}
	anchor.HeadHash = strings.Repeat("f", 64)
	result := store.verifyAnchor(anchor)
	if result.Valid || !strings.Contains(result.Error, "prefix") {
		t.Fatalf("result=%#v", result)
	}
}

func TestRestoreReceiptAnchorAPI(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/api/recording-archive/receipts/anchor", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("anchor status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Header().Get("Content-Disposition"), "number-station-restore-receipt-anchor.json") {
		t.Fatalf("content-disposition=%q", w.Header().Get("Content-Disposition"))
	}
	var anchor restoreReceiptAnchor
	if err := json.Unmarshal(w.Body.Bytes(), &anchor); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(anchor)
	req = httptest.NewRequest(http.MethodPost, "/api/recording-archive/receipts/anchor/verify", bytes.NewReader(body))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("verify status=%d body=%s", w.Code, w.Body.String())
	}
	var result restoreReceiptAnchorVerification
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Valid || result.Status != "match" || result.AnchorCount != 1 {
		t.Fatalf("result=%#v", result)
	}
}

func TestRestoreReceiptAnchorAPIMismatchAndStrictJSON(t *testing.T) {
	db := restoreTestDB(t)
	rs, _ := openRecordingStore(filepath.Join(t.TempDir(), "recordings.json"))
	mux := http.NewServeMux()
	registerRecordingRestoreHandler(mux, db, rs, filepath.Join(t.TempDir(), "audio"))

	mismatch := `{"version":1,"receipt_count":1,"head_hash":"` + strings.Repeat("a", 64) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/recording-archive/receipts/anchor/verify", strings.NewReader(mismatch))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("mismatch status=%d body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/recording-archive/receipts/anchor/verify", strings.NewReader(`{"version":1,"receipt_count":0,"unexpected":true}`))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("strict-json status=%d body=%s", w.Code, w.Body.String())
	}
}
