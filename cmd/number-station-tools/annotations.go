package main

import (
	"errors"
	"sort"
	"strings"
)

const (
	annotationTypeCallUp    = "call-up"
	annotationTypeStationID = "station ID"
	annotationTypeMessage   = "message"
	annotationTypeTone      = "tone"
	annotationTypeNoise     = "noise"
	annotationTypeFade      = "fade"
	annotationTypeOther     = "other"
)

var annotationTypes = map[string]struct{}{
	annotationTypeCallUp:    {},
	annotationTypeStationID: {},
	annotationTypeMessage:   {},
	annotationTypeTone:      {},
	annotationTypeNoise:     {},
	annotationTypeFade:      {},
	annotationTypeOther:     {},
}

type recordingAnnotation struct {
	ID          string `json:"id"`
	RecordingID string `json:"recording_id"`
	StartMS     int64  `json:"start_ms"`
	EndMS       int64  `json:"end_ms,omitempty"`
	Type        string `json:"type,omitempty"`
	Label       string `json:"label"`
	Notes       string `json:"notes,omitempty"`
}

func normalizeAnnotationType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return annotationTypeOther
	}
	return value
}

func validateRecordingAnnotation(rec recording, annotation recordingAnnotation) error {
	annotation.Type = normalizeAnnotationType(annotation.Type)
	annotation.Label = strings.TrimSpace(annotation.Label)
	annotation.Notes = strings.TrimSpace(annotation.Notes)
	if _, ok := annotationTypes[annotation.Type]; !ok {
		return errors.New("type must be call-up, station ID, message, tone, noise, fade, or other")
	}
	if annotation.Label == "" {
		return errors.New("label is required")
	}
	if len(annotation.Label) > 120 {
		return errors.New("label is too long")
	}
	if len(annotation.Notes) > 2000 {
		return errors.New("notes are too long")
	}
	if annotation.StartMS < 0 || annotation.EndMS < 0 {
		return errors.New("annotation timestamps must not be negative")
	}
	if annotation.EndMS > 0 && annotation.EndMS <= annotation.StartMS {
		return errors.New("end_ms must be greater than start_ms")
	}
	if rec.DurationMS > 0 {
		if annotation.StartMS > rec.DurationMS {
			return errors.New("start_ms exceeds recording duration")
		}
		if annotation.EndMS > rec.DurationMS {
			return errors.New("end_ms exceeds recording duration")
		}
	}
	return nil
}

func (s *recordingStore) listAnnotations(recordingID string) ([]recordingAnnotation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	found := false
	for _, rec := range s.data.Recordings {
		if rec.ID == recordingID {
			found = true
			break
		}
	}
	if !found {
		return nil, errors.New("recording does not exist")
	}
	out := make([]recordingAnnotation, 0)
	for _, annotation := range s.data.Annotations {
		if annotation.RecordingID == recordingID {
			annotation.Type = normalizeAnnotationType(annotation.Type)
			out = append(out, annotation)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].StartMS == out[j].StartMS {
			return out[i].ID < out[j].ID
		}
		return out[i].StartMS < out[j].StartMS
	})
	return out, nil
}

func (s *recordingStore) addAnnotation(recordingID string, annotation recordingAnnotation) (recordingAnnotation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var rec *recording
	for i := range s.data.Recordings {
		if s.data.Recordings[i].ID == recordingID {
			rec = &s.data.Recordings[i]
			break
		}
	}
	if rec == nil {
		return recordingAnnotation{}, errors.New("recording does not exist")
	}
	annotation.ID = id()
	annotation.RecordingID = recordingID
	annotation.Type = normalizeAnnotationType(annotation.Type)
	annotation.Label = strings.TrimSpace(annotation.Label)
	annotation.Notes = strings.TrimSpace(annotation.Notes)
	if err := validateRecordingAnnotation(*rec, annotation); err != nil {
		return recordingAnnotation{}, err
	}
	s.data.Annotations = append(s.data.Annotations, annotation)
	if err := s.save(); err != nil {
		return recordingAnnotation{}, err
	}
	return annotation, nil
}

func (s *recordingStore) updateAnnotation(recordingID, annotationID string, annotation recordingAnnotation) (recordingAnnotation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var rec *recording
	for i := range s.data.Recordings {
		if s.data.Recordings[i].ID == recordingID {
			rec = &s.data.Recordings[i]
			break
		}
	}
	if rec == nil {
		return recordingAnnotation{}, errors.New("recording does not exist")
	}
	annotation.ID = annotationID
	annotation.RecordingID = recordingID
	annotation.Type = normalizeAnnotationType(annotation.Type)
	annotation.Label = strings.TrimSpace(annotation.Label)
	annotation.Notes = strings.TrimSpace(annotation.Notes)
	if err := validateRecordingAnnotation(*rec, annotation); err != nil {
		return recordingAnnotation{}, err
	}
	for i := range s.data.Annotations {
		if s.data.Annotations[i].ID == annotationID && s.data.Annotations[i].RecordingID == recordingID {
			s.data.Annotations[i] = annotation
			if err := s.save(); err != nil {
				return recordingAnnotation{}, err
			}
			return annotation, nil
		}
	}
	return recordingAnnotation{}, errors.New("annotation does not exist")
}

func (s *recordingStore) deleteAnnotation(recordingID, annotationID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Annotations {
		if s.data.Annotations[i].ID == annotationID && s.data.Annotations[i].RecordingID == recordingID {
			s.data.Annotations = append(s.data.Annotations[:i], s.data.Annotations[i+1:]...)
			return s.save()
		}
	}
	return errors.New("annotation does not exist")
}
