package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const annotationTransferVersion = 1

type annotationTransferRecording struct {
	ID         string `json:"id"`
	Path       string `json:"path"`
	SHA256     string `json:"sha256,omitempty"`
	DurationMS int64  `json:"duration_ms,omitempty"`
}

type annotationTransferItem struct {
	Recording  annotationTransferRecording `json:"recording"`
	Annotation recordingAnnotation         `json:"annotation"`
}

type annotationTransferBundle struct {
	Version int                      `json:"version"`
	Items   []annotationTransferItem `json:"items"`
}

type annotationImportResult struct {
	Imported  int `json:"imported"`
	Duplicates int `json:"duplicates"`
	Unmatched int `json:"unmatched"`
}

func (s *recordingStore) exportAnnotations() annotationTransferBundle {
	s.mu.Lock()
	defer s.mu.Unlock()
	recordings := make(map[string]recording, len(s.data.Recordings))
	for _, rec := range s.data.Recordings {
		recordings[rec.ID] = rec
	}
	items := make([]annotationTransferItem, 0, len(s.data.Annotations))
	for _, annotation := range s.data.Annotations {
		rec, ok := recordings[annotation.RecordingID]
		if !ok {
			continue
		}
		annotation.Type = normalizeAnnotationType(annotation.Type)
		items = append(items, annotationTransferItem{
			Recording: annotationTransferRecording{ID: rec.ID, Path: rec.Path, SHA256: rec.SHA256, DurationMS: rec.DurationMS},
			Annotation: annotation,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Recording.ID == items[j].Recording.ID {
			return items[i].Annotation.StartMS < items[j].Annotation.StartMS
		}
		return items[i].Recording.ID < items[j].Recording.ID
	})
	return annotationTransferBundle{Version: annotationTransferVersion, Items: items}
}

func encodeAnnotationCSV(bundle annotationTransferBundle) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write([]string{"recording_id", "recording_path", "recording_sha256", "duration_ms", "annotation_id", "start_ms", "end_ms", "type", "label", "notes"}); err != nil {
		return nil, err
	}
	for _, item := range bundle.Items {
		if err := w.Write([]string{
			item.Recording.ID,
			item.Recording.Path,
			item.Recording.SHA256,
			strconv.FormatInt(item.Recording.DurationMS, 10),
			item.Annotation.ID,
			strconv.FormatInt(item.Annotation.StartMS, 10),
			strconv.FormatInt(item.Annotation.EndMS, 10),
			item.Annotation.Type,
			item.Annotation.Label,
			item.Annotation.Notes,
		}); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func sameAnnotation(a, b recordingAnnotation) bool {
	return a.RecordingID == b.RecordingID && a.StartMS == b.StartMS && a.EndMS == b.EndMS && normalizeAnnotationType(a.Type) == normalizeAnnotationType(b.Type) && strings.TrimSpace(a.Label) == strings.TrimSpace(b.Label) && strings.TrimSpace(a.Notes) == strings.TrimSpace(b.Notes)
}

func (s *recordingStore) importAnnotations(bundle annotationTransferBundle) (annotationImportResult, error) {
	if bundle.Version != annotationTransferVersion {
		return annotationImportResult{}, errors.New("unsupported annotation transfer version")
	}
	if len(bundle.Items) > 10000 {
		return annotationImportResult{}, errors.New("annotation import exceeds 10000 items")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	byID := make(map[string]recording, len(s.data.Recordings))
	bySHA := make(map[string][]recording)
	for _, rec := range s.data.Recordings {
		byID[rec.ID] = rec
		hash := strings.ToLower(strings.TrimSpace(rec.SHA256))
		if hash != "" {
			bySHA[hash] = append(bySHA[hash], rec)
		}
	}

	result := annotationImportResult{}
	pending := make([]recordingAnnotation, 0, len(bundle.Items))
	for _, item := range bundle.Items {
		var target recording
		matched := false
		hash := strings.ToLower(strings.TrimSpace(item.Recording.SHA256))
		if hash != "" {
			matches := bySHA[hash]
			if len(matches) == 1 {
				target = matches[0]
				matched = true
			}
		} else if candidate, ok := byID[item.Recording.ID]; ok && candidate.Path == item.Recording.Path {
			target = candidate
			matched = true
		}
		if !matched {
			result.Unmatched++
			continue
		}

		annotation := item.Annotation
		annotation.ID = ""
		annotation.RecordingID = target.ID
		annotation.Type = normalizeAnnotationType(annotation.Type)
		annotation.Label = strings.TrimSpace(annotation.Label)
		annotation.Notes = strings.TrimSpace(annotation.Notes)
		if err := validateRecordingAnnotation(target, annotation); err != nil {
			return annotationImportResult{}, fmt.Errorf("invalid annotation for recording %s: %w", target.ID, err)
		}
		duplicate := false
		for _, existing := range s.data.Annotations {
			if sameAnnotation(existing, annotation) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			for _, existing := range pending {
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
		pending = append(pending, annotation)
	}

	if len(pending) == 0 {
		return result, nil
	}
	s.data.Annotations = append(s.data.Annotations, pending...)
	if err := s.save(); err != nil {
		s.data.Annotations = s.data.Annotations[:len(s.data.Annotations)-len(pending)]
		return annotationImportResult{}, err
	}
	result.Imported = len(pending)
	return result, nil
}

func marshalAnnotationBundle(bundle annotationTransferBundle) ([]byte, error) {
	return json.MarshalIndent(bundle, "", "  ")
}
