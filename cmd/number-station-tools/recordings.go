package main

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strings"
	"sync"
)

type recording struct {
	ID            string `json:"id"`
	ObservationID string `json:"observation_id"`
	Path          string `json:"path"`
	Format        string `json:"format"`
	SizeBytes     int64  `json:"size_bytes,omitempty"`
	DurationMS    int64  `json:"duration_ms,omitempty"`
	SampleRateHz  int    `json:"sample_rate_hz,omitempty"`
	Channels      int    `json:"channels,omitempty"`
	SHA256        string `json:"sha256,omitempty"`
	Notes         string `json:"notes,omitempty"`
}

type recordingData struct {
	Recordings []recording `json:"recordings"`
}

type recordingStore struct {
	mu   sync.Mutex
	path string
	data recordingData
}

func openRecordingStore(path string) (*recordingStore, error) {
	s := &recordingStore{path: path, data: recordingData{Recordings: []recording{}}}
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
	if s.data.Recordings == nil {
		s.data.Recordings = []recording{}
	}
	return s, nil
}

func (s *recordingStore) save() error {
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func validateRecording(v recording) error {
	if strings.TrimSpace(v.ObservationID) == "" {
		return errors.New("observation_id is required")
	}
	if strings.TrimSpace(v.Path) == "" {
		return errors.New("path is required")
	}
	format := strings.ToLower(strings.TrimSpace(v.Format))
	if format != "wav" && format != "flac" {
		return errors.New("format must be wav or flac")
	}
	if v.SizeBytes < 0 || v.DurationMS < 0 || v.SampleRateHz < 0 || v.Channels < 0 {
		return errors.New("numeric metadata must not be negative")
	}
	if v.SHA256 != "" {
		hash := strings.ToLower(strings.TrimSpace(v.SHA256))
		if len(hash) != 64 {
			return errors.New("sha256 must contain 64 hexadecimal characters")
		}
		for _, c := range hash {
			if !strings.ContainsRune("0123456789abcdef", c) {
				return errors.New("sha256 must contain 64 hexadecimal characters")
			}
		}
	}
	return nil
}

func observationExists(db *store, observationID string) bool {
	db.mu.Lock()
	defer db.mu.Unlock()
	for _, obs := range db.data.Observations {
		if obs.ID == observationID {
			return true
		}
	}
	return false
}

func (s *recordingStore) add(db *store, v recording) error {
	if err := validateRecording(v); err != nil {
		return err
	}
	if !observationExists(db, v.ObservationID) {
		return errors.New("observation does not exist")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v.Format = strings.ToLower(strings.TrimSpace(v.Format))
	v.SHA256 = strings.ToLower(strings.TrimSpace(v.SHA256))
	s.data.Recordings = append(s.data.Recordings, v)
	return s.save()
}

func (s *recordingStore) update(db *store, recordingID string, v recording) error {
	if err := validateRecording(v); err != nil {
		return err
	}
	if !observationExists(db, v.ObservationID) {
		return errors.New("observation does not exist")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Recordings {
		if s.data.Recordings[i].ID == recordingID {
			v.ID = recordingID
			v.Format = strings.ToLower(strings.TrimSpace(v.Format))
			v.SHA256 = strings.ToLower(strings.TrimSpace(v.SHA256))
			s.data.Recordings[i] = v
			return s.save()
		}
	}
	return errors.New("recording does not exist")
}

func (s *recordingStore) delete(recordingID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Recordings {
		if s.data.Recordings[i].ID == recordingID {
			s.data.Recordings = append(s.data.Recordings[:i], s.data.Recordings[i+1:]...)
			return s.save()
		}
	}
	return errors.New("recording does not exist")
}

func (s *recordingStore) list() []recording {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]recording(nil), s.data.Recordings...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out
}

func (s *recordingStore) listForObservation(observationID string) []recording {
	all := s.list()
	out := make([]recording, 0)
	for _, rec := range all {
		if rec.ObservationID == observationID {
			out = append(out, rec)
		}
	}
	return out
}

func (s *recordingStore) observationReferenced(observationID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rec := range s.data.Recordings {
		if rec.ObservationID == observationID {
			return true
		}
	}
	return false
}
