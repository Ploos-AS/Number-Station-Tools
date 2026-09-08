package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestRestoreAnchorRegistryTracksGrowth(t *testing.T) {
	dir := t.TempDir()
	receipts, err := openRestoreReceiptStore(filepath.Join(dir, "receipts.json"))
	if err != nil { t.Fatal(err) }
	registry, err := openRestoreAnchorRegistry(filepath.Join(dir, "anchors.json"))
	if err != nil { t.Fatal(err) }
	if _, err := receipts.add(restoreReceipt{Mode: "full", ImportedRecordings: 1}); err != nil { t.Fatal(err) }
	anchor, err := receipts.anchor()
	if err != nil { t.Fatal(err) }
	if _, err := registry.record(anchor); err != nil { t.Fatal(err) }
	status, err := registry.status(receipts)
	if err != nil { t.Fatal(err) }
	if !status.Anchored || status.Status != "current" || status.ReceiptsSince != 0 { t.Fatalf("status=%#v", status) }
	if _, err := receipts.add(restoreReceipt{Mode: "full", ImportedRecordings: 1}); err != nil { t.Fatal(err) }
	status, err = registry.status(receipts)
	if err != nil { t.Fatal(err) }
	if status.Status != "extended" || status.ReceiptsSince != 1 { t.Fatalf("status=%#v", status) }
}

func TestRestoreAnchorRegistryPersists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "anchors.json")
	registry, err := openRestoreAnchorRegistry(path)
	if err != nil { t.Fatal(err) }
	anchor := restoreReceiptAnchor{Version: 1, ReceiptCount: 1, HeadHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	entry, err := registry.record(anchor)
	if err != nil { t.Fatal(err) }
	reopened, err := openRestoreAnchorRegistry(path)
	if err != nil { t.Fatal(err) }
	entries := reopened.list(50)
	if len(entries) != 1 || entries[0].ID != entry.ID || entries[0].HeadHash != anchor.HeadHash { t.Fatalf("entries=%#v", entries) }
}

func TestRestoreAnchorExportRegistersHistoryAPI(t *testing.T) {
	db := restoreTestDB(t)
	dir := t.TempDir()
	recordingPath := filepath.Join(dir, "recordings.json")
	rs, err := openRecordingStore(recordingPath)
	if err != nil { t.Fatal(err) }
	receipts, err := openRestoreReceiptStore(recordingPath + ".restore-receipts.json")
	if err != nil { t.Fatal(err) }
	if _, err := receipts.add(restoreReceipt{Mode: "full", ImportedRecordings: 1}); err != nil { t.Fatal(err) }
	mux := http.NewServeMux()
	registerRecordingRestoreHandler(mux, db, rs, filepath.Join(dir, "audio"))

	exportReq := httptest.NewRequest(http.MethodGet, "/api/recording-archive/receipts/anchor", nil)
	exportW := httptest.NewRecorder()
	mux.ServeHTTP(exportW, exportReq)
	if exportW.Code != http.StatusOK { t.Fatalf("export status=%d body=%s", exportW.Code, exportW.Body.String()) }

	historyReq := httptest.NewRequest(http.MethodGet, "/api/recording-archive/receipts/anchors", nil)
	historyW := httptest.NewRecorder()
	mux.ServeHTTP(historyW, historyReq)
	if historyW.Code != http.StatusOK { t.Fatalf("history status=%d body=%s", historyW.Code, historyW.Body.String()) }
	var entries []restoreAnchorRegistryEntry
	if err := json.Unmarshal(historyW.Body.Bytes(), &entries); err != nil { t.Fatal(err) }
	if len(entries) != 1 || entries[0].ReceiptCount != 1 { t.Fatalf("entries=%#v", entries) }

	statusReq := httptest.NewRequest(http.MethodGet, "/api/recording-archive/receipts/anchors/status", nil)
	statusW := httptest.NewRecorder()
	mux.ServeHTTP(statusW, statusReq)
	if statusW.Code != http.StatusOK { t.Fatalf("status=%d body=%s", statusW.Code, statusW.Body.String()) }
	var status restoreAnchorRegistryStatus
	if err := json.Unmarshal(statusW.Body.Bytes(), &status); err != nil { t.Fatal(err) }
	if status.Status != "current" || status.ReceiptsSince != 0 { t.Fatalf("status=%#v", status) }
}
