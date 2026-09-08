package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type portableSelectiveRestoreResult struct {
	portableRestoreResult
	Selected        int `json:"selected"`
	SkippedByPolicy int `json:"skipped_by_policy"`
}

func restorePortableArchiveSelected(src io.Reader, db *store, rs *recordingStore, audioDir string, selectedIDs []string) (portableSelectiveRestoreResult, error) {
	selected := make(map[string]struct{}, len(selectedIDs))
	for _, raw := range selectedIDs {
		id := strings.TrimSpace(raw)
		if id == "" {
			return portableSelectiveRestoreResult{}, errors.New("selected recording id must not be blank")
		}
		selected[id] = struct{}{}
	}
	if len(selected) == 0 {
		return portableSelectiveRestoreResult{}, errors.New("at least one recording_id must be selected")
	}
	if len(selected) > 10000 {
		return portableSelectiveRestoreResult{}, errors.New("too many selected recording ids")
	}

	staged, err := readPortableArchive(src, audioDir)
	if err != nil {
		return portableSelectiveRestoreResult{}, err
	}
	defer os.RemoveAll(staged.dir)

	result := portableSelectiveRestoreResult{Selected: len(selected)}
	seenIDs := make(map[string]struct{}, len(staged.manifest.Items))
	requiredFiles := make(map[string]struct{})
	manifestIDs := make(map[string]struct{}, len(staged.manifest.Items))

	type preparedItem struct {
		recording   recording
		annotations []recordingAnnotation
		staged      *stagedRestoreFile
	}
	prepared := make([]preparedItem, 0, len(selected))

	for _, item := range staged.manifest.Items {
		rec := portableRecording(item.Recording)
		rec.ID = strings.TrimSpace(rec.ID)
		if rec.ID == "" {
			return portableSelectiveRestoreResult{}, errors.New("manifest contains recording without id")
		}
		if _, exists := seenIDs[rec.ID]; exists {
			return portableSelectiveRestoreResult{}, fmt.Errorf("manifest contains duplicate recording id %s", rec.ID)
		}
		seenIDs[rec.ID] = struct{}{}
		manifestIDs[rec.ID] = struct{}{}

		var stagedFile *stagedRestoreFile
		if rec.Managed {
			archivePath, err := managedArchivePath(rec)
			if err != nil {
				return portableSelectiveRestoreResult{}, fmt.Errorf("recording %s: %w", rec.ID, err)
			}
			file, ok := staged.files[archivePath]
			if !ok {
				return portableSelectiveRestoreResult{}, fmt.Errorf("managed recording %s is missing %s", rec.ID, archivePath)
			}
			requiredFiles[archivePath] = struct{}{}
			rebuilt, err := prepareRestoredRecording(rec, file)
			if err != nil {
				return portableSelectiveRestoreResult{}, fmt.Errorf("recording %s: %w", rec.ID, err)
			}
			rec = rebuilt
			copy := file
			stagedFile = &copy
		} else if err := validateRecording(rec); err != nil {
			return portableSelectiveRestoreResult{}, fmt.Errorf("recording %s: %w", rec.ID, err)
		}

		_, wanted := selected[rec.ID]
		if !wanted {
			result.SkippedByPolicy++
			continue
		}
		if !observationExists(db, rec.ObservationID) {
			return portableSelectiveRestoreResult{}, fmt.Errorf("selected recording %s is unmatched: observation does not exist locally", rec.ID)
		}
		for _, incoming := range item.Annotations {
			annotation := incoming
			annotation.ID = ""
			annotation.RecordingID = rec.ID
			annotation.Type = normalizeAnnotationType(annotation.Type)
			annotation.Label = strings.TrimSpace(annotation.Label)
			annotation.Notes = strings.TrimSpace(annotation.Notes)
			if err := validateRecordingAnnotation(rec, annotation); err != nil {
				return portableSelectiveRestoreResult{}, fmt.Errorf("invalid annotation for recording %s: %w", rec.ID, err)
			}
		}
		prepared = append(prepared, preparedItem{recording: rec, annotations: item.Annotations, staged: stagedFile})
	}
	for id := range selected {
		if _, ok := manifestIDs[id]; !ok {
			return portableSelectiveRestoreResult{}, fmt.Errorf("selected recording %s is not present in archive", id)
		}
	}
	for name := range staged.files {
		if _, ok := requiredFiles[name]; !ok {
			return portableSelectiveRestoreResult{}, fmt.Errorf("archive contains unreferenced audio entry %q", name)
		}
	}

	rs.mu.Lock()
	defer rs.mu.Unlock()
	existingByID := make(map[string]recording, len(rs.data.Recordings))
	for _, rec := range rs.data.Recordings {
		existingByID[rec.ID] = rec
	}
	pendingRecordings := make([]recording, 0, len(prepared))
	pendingAnnotations := make([]recordingAnnotation, 0)
	type move struct{ from, to string }
	moves := make([]move, 0)

	for i := range prepared {
		entry := &prepared[i]
		target := entry.recording
		if existing, ok := existingByID[target.ID]; ok {
			if samePortableRecording(existing, target) {
				return portableSelectiveRestoreResult{}, fmt.Errorf("selected recording %s is a duplicate, not an import candidate", target.ID)
			}
			return portableSelectiveRestoreResult{}, fmt.Errorf("selected recording %s conflicts with an existing recording", target.ID)
		}
		if entry.staged != nil {
			dest := filepath.Join(audioDir, filepath.Base(entry.staged.archivePath))
			if _, err := os.Lstat(dest); err == nil {
				return portableSelectiveRestoreResult{}, fmt.Errorf("selected recording %s conflicts with an existing managed audio destination", target.ID)
			} else if !errors.Is(err, os.ErrNotExist) {
				return portableSelectiveRestoreResult{}, fmt.Errorf("cannot inspect destination for %s", target.ID)
			}
			moves = append(moves, move{from: entry.staged.path, to: dest})
		}
		pendingRecordings = append(pendingRecordings, target)
		existingByID[target.ID] = target
		result.ImportedRecords++

		for _, incoming := range entry.annotations {
			annotation := incoming
			annotation.ID = ""
			annotation.RecordingID = target.ID
			annotation.Type = normalizeAnnotationType(annotation.Type)
			annotation.Label = strings.TrimSpace(annotation.Label)
			annotation.Notes = strings.TrimSpace(annotation.Notes)
			duplicate := false
			for _, existing := range rs.data.Annotations {
				if sameAnnotation(existing, annotation) {
					duplicate = true
					break
				}
			}
			if !duplicate {
				for _, existing := range pendingAnnotations {
					if sameAnnotation(existing, annotation) {
						duplicate = true
						break
					}
				}
			}
			if duplicate {
				result.Duplicates++
				continue
			}
			annotation.ID = id()
			pendingAnnotations = append(pendingAnnotations, annotation)
			result.ImportedAnnotations++
		}
	}

	moved := make([]move, 0, len(moves))
	for _, mv := range moves {
		if err := os.Rename(mv.from, mv.to); err != nil {
			for j := len(moved) - 1; j >= 0; j-- {
				_ = os.Rename(moved[j].to, moved[j].from)
			}
			return portableSelectiveRestoreResult{}, errors.New("cannot finalize selected restored audio files")
		}
		moved = append(moved, mv)
	}
	oldRecordings := len(rs.data.Recordings)
	oldAnnotations := len(rs.data.Annotations)
	rs.data.Recordings = append(rs.data.Recordings, pendingRecordings...)
	rs.data.Annotations = append(rs.data.Annotations, pendingAnnotations...)
	if err := rs.save(); err != nil {
		rs.data.Recordings = rs.data.Recordings[:oldRecordings]
		rs.data.Annotations = rs.data.Annotations[:oldAnnotations]
		for j := len(moved) - 1; j >= 0; j-- {
			_ = os.Rename(moved[j].to, moved[j].from)
		}
		return portableSelectiveRestoreResult{}, err
	}
	return result, nil
}
