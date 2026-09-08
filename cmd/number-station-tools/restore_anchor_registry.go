package main

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"sync"
	"time"
)

type restoreAnchorRegistryEntry struct {
	ID           string `json:"id"`
	ExportedAt   string `json:"exported_at"`
	ReceiptCount int    `json:"receipt_count"`
	HeadHash     string `json:"head_hash,omitempty"`
}

type restoreAnchorRegistryData struct {
	Entries []restoreAnchorRegistryEntry `json:"entries"`
}

type restoreAnchorRegistryStatus struct {
	Anchored          bool                       `json:"anchored"`
	LastAnchor        *restoreAnchorRegistryEntry `json:"last_anchor,omitempty"`
	LocalReceiptCount int                        `json:"local_receipt_count"`
	CurrentHeadHash   string                     `json:"current_head_hash,omitempty"`
	ReceiptsSince     int                        `json:"receipts_since_last_anchor"`
	Status            string                     `json:"status"`
}

type restoreAnchorRegistry struct {
	mu   sync.Mutex
	path string
	data restoreAnchorRegistryData
}

func openRestoreAnchorRegistry(path string) (*restoreAnchorRegistry, error) {
	r := &restoreAnchorRegistry{path: path, data: restoreAnchorRegistryData{Entries: []restoreAnchorRegistryEntry{}}}
	body, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return r, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(body, &r.data); err != nil {
		return nil, err
	}
	if r.data.Entries == nil {
		r.data.Entries = []restoreAnchorRegistryEntry{}
	}
	return r, nil
}

func (r *restoreAnchorRegistry) save() error {
	body, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := r.path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, r.path)
}

func (r *restoreAnchorRegistry) record(anchor restoreReceiptAnchor) (restoreAnchorRegistryEntry, error) {
	if err := validateRestoreReceiptAnchor(anchor); err != nil {
		return restoreAnchorRegistryEntry{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	entry := restoreAnchorRegistryEntry{ID: id(), ExportedAt: time.Now().UTC().Format(time.RFC3339Nano), ReceiptCount: anchor.ReceiptCount, HeadHash: anchor.HeadHash}
	r.data.Entries = append(r.data.Entries, entry)
	if err := r.save(); err != nil {
		r.data.Entries = r.data.Entries[:len(r.data.Entries)-1]
		return restoreAnchorRegistryEntry{}, err
	}
	return entry, nil
}

func (r *restoreAnchorRegistry) list(limit int) []restoreAnchorRegistryEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := append([]restoreAnchorRegistryEntry(nil), r.data.Entries...)
	sort.Slice(out, func(i, j int) bool { return out[i].ExportedAt > out[j].ExportedAt })
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func (r *restoreAnchorRegistry) status(receipts *restoreReceiptStore) (restoreAnchorRegistryStatus, error) {
	anchor, err := receipts.anchor()
	if err != nil {
		return restoreAnchorRegistryStatus{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	result := restoreAnchorRegistryStatus{LocalReceiptCount: anchor.ReceiptCount, CurrentHeadHash: anchor.HeadHash, Status: "never-anchored"}
	if len(r.data.Entries) == 0 {
		return result, nil
	}
	last := r.data.Entries[len(r.data.Entries)-1]
	result.Anchored = true
	result.LastAnchor = &last
	verification := receipts.verifyAnchor(restoreReceiptAnchor{Version: restoreReceiptAnchorVersion, ReceiptCount: last.ReceiptCount, HeadHash: last.HeadHash})
	if !verification.Valid {
		result.Status = "mismatch"
		return result, errors.New("last registered anchor does not match local receipt chain")
	}
	result.ReceiptsSince = anchor.ReceiptCount - last.ReceiptCount
	if result.ReceiptsSince == 0 {
		result.Status = "current"
	} else {
		result.Status = "extended"
	}
	return result, nil
}
