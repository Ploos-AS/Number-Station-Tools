package main

import (
	"encoding/json"
	"sort"
	"strings"
)

const recordingBundleVersion = 1

type recordingBundleItem struct {
	Recording   recording             `json:"recording"`
	Annotations []recordingAnnotation `json:"annotations"`
}

type recordingBundleManifest struct {
	Version int                   `json:"version"`
	Items   []recordingBundleItem `json:"items"`
}

func portableRecording(rec recording) recording {
	rec.Waveform = nil
	rec.Frequency = nil
	rec.Spectrogram = nil
	rec.Fingerprint = nil
	return rec
}

func (s *recordingStore) exportRecordingBundle() recordingBundleManifest {
	s.mu.Lock()
	defer s.mu.Unlock()

	annotationsByRecording := make(map[string][]recordingAnnotation)
	for _, annotation := range s.data.Annotations {
		annotation.Type = normalizeAnnotationType(annotation.Type)
		annotationsByRecording[annotation.RecordingID] = append(annotationsByRecording[annotation.RecordingID], annotation)
	}

	items := make([]recordingBundleItem, 0, len(s.data.Recordings))
	for _, rec := range s.data.Recordings {
		annotations := append([]recordingAnnotation(nil), annotationsByRecording[rec.ID]...)
		sort.Slice(annotations, func(i, j int) bool {
			if annotations[i].StartMS == annotations[j].StartMS {
				return annotations[i].ID < annotations[j].ID
			}
			return annotations[i].StartMS < annotations[j].StartMS
		})
		item := recordingBundleItem{Recording: portableRecording(rec), Annotations: annotations}
		item.Recording.SHA256 = strings.ToLower(strings.TrimSpace(item.Recording.SHA256))
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Recording.ID < items[j].Recording.ID })
	return recordingBundleManifest{Version: recordingBundleVersion, Items: items}
}

func marshalRecordingBundle(bundle recordingBundleManifest) ([]byte, error) {
	return json.MarshalIndent(bundle, "", "  ")
}
