package main

import (
	"sort"
)

const defaultClusterThreshold = 98.0

type clusterMember struct {
	RecordingID   string `json:"recording_id"`
	ObservationID string `json:"observation_id"`
	Path          string `json:"path"`
	SHA256        string `json:"sha256,omitempty"`
}

type recordingCluster struct {
	ID             string          `json:"id"`
	Members        []clusterMember `json:"members"`
	MinimumScore   float64         `json:"minimum_score"`
	ExactDuplicate bool            `json:"exact_duplicate"`
}

func (s *recordingStore) clusters(threshold float64) []recordingCluster {
	if threshold <= 0 || threshold > 100 {
		threshold = defaultClusterThreshold
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	recs := make([]recording, 0)
	for _, rec := range s.data.Recordings {
		if len(rec.Fingerprint) > 0 {
			recs = append(recs, rec)
		}
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].ID < recs[j].ID })
	parent := make([]int, len(recs))
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[rb] = ra
		}
	}
	for i := 0; i < len(recs); i++ {
		for j := i + 1; j < len(recs); j++ {
			if fingerprintSimilarity(recs[i].Fingerprint, recs[j].Fingerprint) >= threshold {
				union(i, j)
			}
		}
	}

	groups := map[int][]int{}
	for i := range recs {
		root := find(i)
		groups[root] = append(groups[root], i)
	}
	out := make([]recordingCluster, 0)
	for _, indexes := range groups {
		if len(indexes) < 2 {
			continue
		}
		members := make([]clusterMember, 0, len(indexes))
		minimum := 100.0
		exact := false
		seenHash := map[string]bool{}
		for _, idx := range indexes {
			rec := recs[idx]
			members = append(members, clusterMember{RecordingID: rec.ID, ObservationID: rec.ObservationID, Path: rec.Path, SHA256: rec.SHA256})
			if rec.SHA256 != "" {
				if seenHash[rec.SHA256] {
					exact = true
				}
				seenHash[rec.SHA256] = true
			}
		}
		for a := 0; a < len(indexes); a++ {
			for b := a + 1; b < len(indexes); b++ {
				score := fingerprintSimilarity(recs[indexes[a]].Fingerprint, recs[indexes[b]].Fingerprint)
				if score < minimum {
					minimum = score
				}
			}
		}
		out = append(out, recordingCluster{ID: members[0].RecordingID, Members: members, MinimumScore: minimum, ExactDuplicate: exact})
	}
	sort.Slice(out, func(i, j int) bool {
		if len(out[i].Members) != len(out[j].Members) {
			return len(out[i].Members) > len(out[j].Members)
		}
		return out[i].ID < out[j].ID
	})
	return out
}
