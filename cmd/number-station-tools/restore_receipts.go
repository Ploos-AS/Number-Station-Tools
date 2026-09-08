package main

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type restoreReceipt struct {
	ID                   string   `json:"id"`
	At                   string   `json:"at"`
	Mode                 string   `json:"mode"`
	PlanToken            string   `json:"plan_token,omitempty"`
	SelectedRecordingIDs []string `json:"selected_recording_ids,omitempty"`
	ImportedRecordings   int      `json:"imported_recordings"`
	ImportedAnnotations  int      `json:"imported_annotations"`
	Duplicates           int      `json:"duplicates"`
	Conflicts            int      `json:"conflicts"`
	Unmatched            int      `json:"unmatched"`
	SkippedByPolicy      int      `json:"skipped_by_policy,omitempty"`
}

type restoreReceiptData struct {
	Receipts []restoreReceipt `json:"receipts"`
}

type restoreReceiptStore struct {
	mu   sync.Mutex
	path string
	data restoreReceiptData
}

func openRestoreReceiptStore(path string) (*restoreReceiptStore, error) {
	s := &restoreReceiptStore{path: path, data: restoreReceiptData{Receipts: []restoreReceipt{}}}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &s.data); err != nil {
		return nil, err
	}
	if s.data.Receipts == nil {
		s.data.Receipts = []restoreReceipt{}
	}
	return s, nil
}

func (s *restoreReceiptStore) save() error {
	body, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *restoreReceiptStore) add(receipt restoreReceipt) (restoreReceipt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	receipt.ID = id()
	receipt.At = time.Now().UTC().Format(time.RFC3339Nano)
	receipt.Mode = strings.TrimSpace(receipt.Mode)
	if receipt.Mode != "full" && receipt.Mode != "selected" {
		return restoreReceipt{}, errors.New("invalid restore receipt mode")
	}
	receipt.PlanToken = strings.ToLower(strings.TrimSpace(receipt.PlanToken))
	receipt.SelectedRecordingIDs = append([]string(nil), receipt.SelectedRecordingIDs...)
	sort.Strings(receipt.SelectedRecordingIDs)
	s.data.Receipts = append(s.data.Receipts, receipt)
	if err := s.save(); err != nil {
		s.data.Receipts = s.data.Receipts[:len(s.data.Receipts)-1]
		return restoreReceipt{}, err
	}
	return receipt, nil
}

func (s *restoreReceiptStore) list(limit int) []restoreReceipt {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	out := append([]restoreReceipt(nil), s.data.Receipts...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].At == out[j].At {
			return out[i].ID > out[j].ID
		}
		return out[i].At > out[j].At
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

type restoreResponse struct {
	portableRestoreResult
	Receipt restoreReceipt `json:"receipt"`
}

type selectiveRestoreResponse struct {
	portableSelectiveRestoreResult
	Receipt restoreReceipt `json:"receipt"`
}
