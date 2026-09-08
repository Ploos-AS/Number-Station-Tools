package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
	PreviousHash         string   `json:"previous_hash,omitempty"`
	ReceiptHash          string   `json:"receipt_hash"`
}

type restoreReceiptData struct {
	Receipts []restoreReceipt `json:"receipts"`
}

type restoreReceiptVerification struct {
	Valid           bool   `json:"valid"`
	Count           int    `json:"count"`
	HeadHash        string `json:"head_hash,omitempty"`
	FailedIndex     int    `json:"failed_index,omitempty"`
	FailedReceiptID string `json:"failed_receipt_id,omitempty"`
	Error           string `json:"error,omitempty"`
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
	if len(s.data.Receipts) > 0 && receiptChainIsLegacy(s.data.Receipts) {
		sealReceiptChain(s.data.Receipts)
		if err := s.save(); err != nil {
			return nil, fmt.Errorf("cannot migrate legacy restore receipts: %w", err)
		}
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

func receiptChainIsLegacy(receipts []restoreReceipt) bool {
	if len(receipts) == 0 {
		return false
	}
	for _, receipt := range receipts {
		if strings.TrimSpace(receipt.PreviousHash) != "" || strings.TrimSpace(receipt.ReceiptHash) != "" {
			return false
		}
	}
	return true
}

func restoreReceiptHash(receipt restoreReceipt) (string, error) {
	receipt.ReceiptHash = ""
	body, err := json.Marshal(receipt)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}

func sealReceiptChain(receipts []restoreReceipt) {
	previous := ""
	for i := range receipts {
		receipts[i].PreviousHash = previous
		hash, _ := restoreReceiptHash(receipts[i])
		receipts[i].ReceiptHash = hash
		previous = hash
	}
}

func verifyReceiptChain(receipts []restoreReceipt) restoreReceiptVerification {
	verification := restoreReceiptVerification{Valid: true, Count: len(receipts)}
	previous := ""
	for i, receipt := range receipts {
		storedPrevious := strings.ToLower(strings.TrimSpace(receipt.PreviousHash))
		storedHash := strings.ToLower(strings.TrimSpace(receipt.ReceiptHash))
		if storedPrevious != previous {
			return restoreReceiptVerification{Valid: false, Count: len(receipts), FailedIndex: i, FailedReceiptID: receipt.ID, Error: "previous_hash does not match preceding receipt"}
		}
		if len(storedHash) != 64 {
			return restoreReceiptVerification{Valid: false, Count: len(receipts), FailedIndex: i, FailedReceiptID: receipt.ID, Error: "receipt_hash is missing or invalid"}
		}
		if _, err := hex.DecodeString(storedHash); err != nil {
			return restoreReceiptVerification{Valid: false, Count: len(receipts), FailedIndex: i, FailedReceiptID: receipt.ID, Error: "receipt_hash is not hexadecimal"}
		}
		receipt.PreviousHash = storedPrevious
		receipt.ReceiptHash = storedHash
		expected, err := restoreReceiptHash(receipt)
		if err != nil {
			return restoreReceiptVerification{Valid: false, Count: len(receipts), FailedIndex: i, FailedReceiptID: receipt.ID, Error: "cannot calculate receipt hash"}
		}
		if expected != storedHash {
			return restoreReceiptVerification{Valid: false, Count: len(receipts), FailedIndex: i, FailedReceiptID: receipt.ID, Error: "receipt content hash mismatch"}
		}
		previous = storedHash
	}
	verification.HeadHash = previous
	return verification
}

func (s *restoreReceiptStore) add(receipt restoreReceipt) (restoreReceipt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	verification := verifyReceiptChain(s.data.Receipts)
	if !verification.Valid {
		return restoreReceipt{}, errors.New("restore receipt chain integrity check failed")
	}
	receipt.ID = id()
	receipt.At = time.Now().UTC().Format(time.RFC3339Nano)
	receipt.Mode = strings.TrimSpace(receipt.Mode)
	if receipt.Mode != "full" && receipt.Mode != "selected" {
		return restoreReceipt{}, errors.New("invalid restore receipt mode")
	}
	receipt.PlanToken = strings.ToLower(strings.TrimSpace(receipt.PlanToken))
	receipt.SelectedRecordingIDs = append([]string(nil), receipt.SelectedRecordingIDs...)
	sort.Strings(receipt.SelectedRecordingIDs)
	receipt.PreviousHash = verification.HeadHash
	receipt.ReceiptHash = ""
	hash, err := restoreReceiptHash(receipt)
	if err != nil {
		return restoreReceipt{}, err
	}
	receipt.ReceiptHash = hash
	s.data.Receipts = append(s.data.Receipts, receipt)
	if err := s.save(); err != nil {
		s.data.Receipts = s.data.Receipts[:len(s.data.Receipts)-1]
		return restoreReceipt{}, err
	}
	return receipt, nil
}

func (s *restoreReceiptStore) verify() restoreReceiptVerification {
	s.mu.Lock()
	defer s.mu.Unlock()
	return verifyReceiptChain(s.data.Receipts)
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
