package main

import "testing"

func TestRecordingClusters(t *testing.T) {
	s := &recordingStore{data: recordingData{Recordings: []recording{
		{ID: "a", ObservationID: "o1", Path: "a.wav", SHA256: "same", Fingerprint: []uint8{255, 200, 100, 20}},
		{ID: "b", ObservationID: "o2", Path: "b.wav", SHA256: "same", Fingerprint: []uint8{255, 198, 102, 20}},
		{ID: "c", ObservationID: "o3", Path: "c.wav", Fingerprint: []uint8{20, 100, 200, 255}},
	}}}

	clusters := s.clusters(99)
	if len(clusters) != 1 {
		t.Fatalf("expected one cluster, got %d", len(clusters))
	}
	cluster := clusters[0]
	if len(cluster.Members) != 2 {
		t.Fatalf("expected two members, got %d", len(cluster.Members))
	}
	if !cluster.ExactDuplicate {
		t.Fatal("expected duplicate SHA-256 to mark exact duplicate")
	}
	if cluster.Members[0].RecordingID != "a" || cluster.Members[1].RecordingID != "b" {
		t.Fatalf("unexpected cluster members: %#v", cluster.Members)
	}
	if cluster.MinimumScore < 99 {
		t.Fatalf("expected minimum score >= 99, got %.2f", cluster.MinimumScore)
	}
}

func TestRecordingClustersDefaultThresholdAndNoSingletons(t *testing.T) {
	s := &recordingStore{data: recordingData{Recordings: []recording{
		{ID: "a", Fingerprint: []uint8{255, 200, 100, 20}},
		{ID: "b", Fingerprint: []uint8{255, 198, 102, 20}},
		{ID: "c", Fingerprint: []uint8{20, 100, 200, 255}},
		{ID: "none"},
	}}}

	clusters := s.clusters(0)
	if len(clusters) != 1 {
		t.Fatalf("expected one cluster at default threshold, got %d", len(clusters))
	}
	if len(clusters[0].Members) != 2 {
		t.Fatalf("expected singleton exclusion, got %d members", len(clusters[0].Members))
	}
}
