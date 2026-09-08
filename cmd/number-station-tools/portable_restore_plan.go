package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type portableRestorePlanEntry struct {
	RecordingID       string `json:"recording_id"`
	ObservationID     string `json:"observation_id"`
	Path              string `json:"path"`
	Managed           bool   `json:"managed"`
	Action            string `json:"action"`
	Reason            string `json:"reason,omitempty"`
	AnnotationsImport int    `json:"annotations_import"`
	AnnotationsSkip   int    `json:"annotations_skip"`
}

type portableRestorePlan struct {
	Import    int                        `json:"import"`
	Duplicate int                        `json:"duplicate"`
	Conflict  int                        `json:"conflict"`
	Unmatched int                        `json:"unmatched"`
	PlanToken string                     `json:"plan_token"`
	Entries   []portableRestorePlanEntry `json:"entries"`
}

type portablePlanFileFingerprint struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type portablePlanLocalState struct {
	Recordings   []recording           `json:"recordings"`
	Annotations  []recordingAnnotation `json:"annotations"`
	Observations []observation         `json:"observations"`
}

type portablePlanFingerprint struct {
	Manifest   recordingBundleManifest       `json:"manifest"`
	Files      []portablePlanFileFingerprint `json:"files"`
	Plan       portableRestorePlan           `json:"plan"`
	LocalState portablePlanLocalState        `json:"local_state"`
}

func snapshotPortablePlanLocalState(db *store, rs *recordingStore) portablePlanLocalState {
	rs.mu.Lock()
	recordings := append([]recording(nil), rs.data.Recordings...)
	annotations := append([]recordingAnnotation(nil), rs.data.Annotations...)
	rs.mu.Unlock()
	db.mu.Lock()
	observations := append([]observation(nil), db.data.Observations...)
	db.mu.Unlock()
	sort.Slice(recordings, func(i, j int) bool { return recordings[i].ID < recordings[j].ID })
	sort.Slice(annotations, func(i, j int) bool { return annotations[i].ID < annotations[j].ID })
	sort.Slice(observations, func(i, j int) bool { return observations[i].ID < observations[j].ID })
	return portablePlanLocalState{Recordings: recordings, Annotations: annotations, Observations: observations}
}

func computePortableRestorePlanToken(staged stagedPortableArchive, plan portableRestorePlan, localState portablePlanLocalState) (string, error) {
	files := make([]portablePlanFileFingerprint, 0, len(staged.files))
	for name, file := range staged.files {
		files = append(files, portablePlanFileFingerprint{Path: name, Size: file.size, SHA256: file.sha256})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	plan.PlanToken = ""
	body, err := json.Marshal(portablePlanFingerprint{Manifest: staged.manifest, Files: files, Plan: plan, LocalState: localState})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}

func planPortableRestore(src io.Reader, db *store, rs *recordingStore, audioDir string) (portableRestorePlan, error) {
	staged, err := readPortableArchive(src, audioDir)
	if err != nil {
		return portableRestorePlan{}, err
	}
	defer os.RemoveAll(staged.dir)
	return planStagedPortableRestore(staged, db, rs, audioDir)
}

func planStagedPortableRestore(staged stagedPortableArchive, db *store, rs *recordingStore, audioDir string) (portableRestorePlan, error) {
	plan := portableRestorePlan{Entries: []portableRestorePlanEntry{}}
	seenIDs := make(map[string]struct{}, len(staged.manifest.Items))
	requiredFiles := make(map[string]struct{})

	rs.mu.Lock()
	existingRecordings := append([]recording(nil), rs.data.Recordings...)
	existingAnnotations := append([]recordingAnnotation(nil), rs.data.Annotations...)
	rs.mu.Unlock()
	existingByID := make(map[string]recording, len(existingRecordings))
	for _, rec := range existingRecordings {
		existingByID[rec.ID] = rec
	}

	for _, item := range staged.manifest.Items {
		rec := portableRecording(item.Recording)
		rec.ID = strings.TrimSpace(rec.ID)
		if rec.ID == "" {
			return portableRestorePlan{}, errors.New("manifest contains recording without id")
		}
		if _, exists := seenIDs[rec.ID]; exists {
			return portableRestorePlan{}, fmt.Errorf("manifest contains duplicate recording id %s", rec.ID)
		}
		seenIDs[rec.ID] = struct{}{}

		entry := portableRestorePlanEntry{RecordingID: rec.ID, ObservationID: rec.ObservationID, Path: rec.Path, Managed: rec.Managed}
		if rec.Managed {
			archivePath, err := managedArchivePath(rec)
			if err != nil {
				return portableRestorePlan{}, fmt.Errorf("recording %s: %w", rec.ID, err)
			}
			file, ok := staged.files[archivePath]
			if !ok {
				return portableRestorePlan{}, fmt.Errorf("managed recording %s is missing %s", rec.ID, archivePath)
			}
			requiredFiles[archivePath] = struct{}{}
			rebuilt, err := prepareRestoredRecording(rec, file)
			if err != nil {
				return portableRestorePlan{}, fmt.Errorf("recording %s: %w", rec.ID, err)
			}
			rec = rebuilt
		} else if err := validateRecording(rec); err != nil {
			return portableRestorePlan{}, fmt.Errorf("recording %s: %w", rec.ID, err)
		}

		if !observationExists(db, rec.ObservationID) {
			entry.Action = "unmatched"
			entry.Reason = "observation does not exist locally"
			plan.Unmatched++
			plan.Entries = append(plan.Entries, entry)
			continue
		}

		if existing, ok := existingByID[rec.ID]; ok {
			if samePortableRecording(existing, rec) {
				entry.Action = "duplicate"
				entry.Reason = "equivalent recording already exists"
				plan.Duplicate++
				rec = existing
			} else {
				entry.Action = "conflict"
				entry.Reason = "recording id already exists with different content"
				plan.Conflict++
				plan.Entries = append(plan.Entries, entry)
				continue
			}
		} else if rec.Managed {
			dest := filepath.Join(audioDir, filepath.Base(rec.Path))
			if _, err := os.Lstat(dest); err == nil {
				entry.Action = "conflict"
				entry.Reason = "managed audio destination already exists"
				plan.Conflict++
				plan.Entries = append(plan.Entries, entry)
				continue
			} else if !errors.Is(err, os.ErrNotExist) {
				return portableRestorePlan{}, fmt.Errorf("cannot inspect destination for %s", rec.ID)
			}
			entry.Action = "import"
			plan.Import++
		} else {
			entry.Action = "import"
			plan.Import++
		}

		planned := append([]recordingAnnotation(nil), existingAnnotations...)
		for _, incoming := range item.Annotations {
			annotation := incoming
			annotation.ID = ""
			annotation.RecordingID = rec.ID
			annotation.Type = normalizeAnnotationType(annotation.Type)
			annotation.Label = strings.TrimSpace(annotation.Label)
			annotation.Notes = strings.TrimSpace(annotation.Notes)
			if err := validateRecordingAnnotation(rec, annotation); err != nil {
				return portableRestorePlan{}, fmt.Errorf("invalid annotation for recording %s: %w", rec.ID, err)
			}
			duplicate := false
			for _, existing := range planned {
				if sameAnnotation(existing, annotation) {
					duplicate = true
					break
				}
			}
			if duplicate {
				entry.AnnotationsSkip++
				continue
			}
			planned = append(planned, annotation)
			entry.AnnotationsImport++
		}
		plan.Entries = append(plan.Entries, entry)
	}

	for name := range staged.files {
		if _, ok := requiredFiles[name]; !ok {
			return portableRestorePlan{}, fmt.Errorf("archive contains unreferenced audio entry %q", name)
		}
	}
	localState := snapshotPortablePlanLocalState(db, rs)
	token, err := computePortableRestorePlanToken(staged, plan, localState)
	if err != nil {
		return portableRestorePlan{}, errors.New("cannot fingerprint restore plan")
	}
	plan.PlanToken = token
	return plan, nil
}
