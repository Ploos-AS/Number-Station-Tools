package main

import (
	"encoding/hex"
	"errors"
	"strings"
)

const restoreReceiptAnchorVersion = 1

type restoreReceiptAnchor struct {
	Version      int    `json:"version"`
	ReceiptCount int    `json:"receipt_count"`
	HeadHash     string `json:"head_hash,omitempty"`
}

type restoreReceiptAnchorVerification struct {
	Valid           bool   `json:"valid"`
	AnchorCount     int    `json:"anchor_count"`
	LocalCount      int    `json:"local_count"`
	AnchorHash      string `json:"anchor_hash,omitempty"`
	CurrentHeadHash string `json:"current_head_hash,omitempty"`
	Status          string `json:"status,omitempty"`
	Error           string `json:"error,omitempty"`
}

func validateRestoreReceiptAnchor(anchor restoreReceiptAnchor) error {
	if anchor.Version != restoreReceiptAnchorVersion {
		return errors.New("unsupported restore receipt anchor version")
	}
	if anchor.ReceiptCount < 0 {
		return errors.New("receipt_count must not be negative")
	}
	anchor.HeadHash = strings.ToLower(strings.TrimSpace(anchor.HeadHash))
	if anchor.ReceiptCount == 0 {
		if anchor.HeadHash != "" {
			return errors.New("empty receipt anchor must not contain head_hash")
		}
		return nil
	}
	if len(anchor.HeadHash) != 64 {
		return errors.New("head_hash must contain 64 hexadecimal characters")
	}
	if _, err := hex.DecodeString(anchor.HeadHash); err != nil {
		return errors.New("head_hash must contain 64 hexadecimal characters")
	}
	return nil
}

func (s *restoreReceiptStore) anchor() (restoreReceiptAnchor, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	verification := verifyReceiptChain(s.data.Receipts)
	if !verification.Valid {
		return restoreReceiptAnchor{}, errors.New("restore receipt chain integrity check failed")
	}
	return restoreReceiptAnchor{
		Version:      restoreReceiptAnchorVersion,
		ReceiptCount: verification.Count,
		HeadHash:     verification.HeadHash,
	}, nil
}

func (s *restoreReceiptStore) verifyAnchor(anchor restoreReceiptAnchor) restoreReceiptAnchorVerification {
	s.mu.Lock()
	defer s.mu.Unlock()

	anchor.HeadHash = strings.ToLower(strings.TrimSpace(anchor.HeadHash))
	result := restoreReceiptAnchorVerification{
		AnchorCount: anchor.ReceiptCount,
		LocalCount:  len(s.data.Receipts),
		AnchorHash:  anchor.HeadHash,
	}
	chain := verifyReceiptChain(s.data.Receipts)
	result.CurrentHeadHash = chain.HeadHash
	if !chain.Valid {
		result.Error = "local restore receipt chain integrity check failed"
		return result
	}
	if err := validateRestoreReceiptAnchor(anchor); err != nil {
		result.Error = err.Error()
		return result
	}
	if anchor.ReceiptCount > len(s.data.Receipts) {
		result.Error = "anchor refers to more receipts than exist locally"
		return result
	}
	if anchor.ReceiptCount == 0 {
		result.Valid = true
		if len(s.data.Receipts) == 0 {
			result.Status = "match"
		} else {
			result.Status = "extended"
		}
		return result
	}
	expected := strings.ToLower(strings.TrimSpace(s.data.Receipts[anchor.ReceiptCount-1].ReceiptHash))
	if expected != anchor.HeadHash {
		result.Error = "anchor head_hash does not match the local receipt chain prefix"
		return result
	}
	result.Valid = true
	if anchor.ReceiptCount == len(s.data.Receipts) {
		result.Status = "match"
	} else {
		result.Status = "extended"
	}
	return result
}
