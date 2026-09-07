package main

import (
	"errors"
	"math"
	"sort"
)

const fingerprintBins = 32

type similarityResult struct {
	RecordingID   string  `json:"recording_id"`
	ObservationID string  `json:"observation_id"`
	Path          string  `json:"path"`
	Score         float64 `json:"score"`
}

func buildSignalFingerprint(frequency []uint8, spectrogram []uint8, timeBins, freqBins int) []uint8 {
	out := make([]uint8, fingerprintBins)
	if len(spectrogram) > 0 && timeBins > 0 && freqBins > 0 && len(spectrogram) == timeBins*freqBins {
		for dst := 0; dst < fingerprintBins; dst++ {
			start := dst * freqBins / fingerprintBins
			end := (dst + 1) * freqBins / fingerprintBins
			if end <= start {
				end = start + 1
			}
			var sum, count int
			for t := 0; t < timeBins; t++ {
				for f := start; f < end && f < freqBins; f++ {
					sum += int(spectrogram[t*freqBins+f])
					count++
				}
			}
			if count > 0 {
				out[dst] = uint8(sum / count)
			}
		}
		return normalizeFingerprint(out)
	}
	if len(frequency) == 0 {
		return nil
	}
	for dst := 0; dst < fingerprintBins; dst++ {
		start := dst * len(frequency) / fingerprintBins
		end := (dst + 1) * len(frequency) / fingerprintBins
		if end <= start {
			end = start + 1
		}
		var sum, count int
		for i := start; i < end && i < len(frequency); i++ {
			sum += int(frequency[i])
			count++
		}
		if count > 0 {
			out[dst] = uint8(sum / count)
		}
	}
	return normalizeFingerprint(out)
}

func normalizeFingerprint(v []uint8) []uint8 {
	max := uint8(0)
	for _, x := range v {
		if x > max {
			max = x
		}
	}
	if max == 0 {
		return v
	}
	for i := range v {
		v[i] = uint8(math.Round(float64(v[i]) * 255 / float64(max)))
	}
	return v
}

func fingerprintSimilarity(a, b []uint8) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, aa, bb float64
	for i := range a {
		x, y := float64(a[i]), float64(b[i])
		dot += x * y
		aa += x * x
		bb += y * y
	}
	if aa == 0 || bb == 0 {
		return 0
	}
	return math.Round((dot/math.Sqrt(aa*bb))*10000) / 100
}

func (s *recordingStore) similar(recordingID string, limit int) ([]similarityResult, error) {
	if limit < 1 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var target *recording
	for i := range s.data.Recordings {
		if s.data.Recordings[i].ID == recordingID {
			target = &s.data.Recordings[i]
			break
		}
	}
	if target == nil {
		return nil, errors.New("recording does not exist")
	}
	if len(target.Fingerprint) == 0 {
		return []similarityResult{}, nil
	}
	out := make([]similarityResult, 0)
	for _, rec := range s.data.Recordings {
		if rec.ID == recordingID || len(rec.Fingerprint) == 0 {
			continue
		}
		out = append(out, similarityResult{RecordingID: rec.ID, ObservationID: rec.ObservationID, Path: rec.Path, Score: fingerprintSimilarity(target.Fingerprint, rec.Fingerprint)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
