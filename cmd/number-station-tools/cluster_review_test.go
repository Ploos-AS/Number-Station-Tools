package main

import (
	"path/filepath"
	"testing"
)

func TestClusterReviewPersistsAndAttaches(t *testing.T) {
	path := filepath.Join(t.TempDir(), "recordings.json")
	s := &recordingStore{path: path, data: recordingData{Recordings: []recording{
		{ID: "a", ObservationID: "o1", Path: "a.wav", Fingerprint: []uint8{255, 200, 100, 20}},
		{ID: "b", ObservationID: "o2", Path: "b.wav", Fingerprint: []uint8{255, 198, 102, 20}},
	}, ClusterReviews: []clusterReview{}}}

	clusters := s.clusters(99)
	if len(clusters) != 1 {
		t.Fatalf("expected one cluster, got %d", len(clusters))
	}
	review, err := s.setClusterReview(clusters[0], clusterReviewSameTransmission, "same call-up and groups")
	if err != nil {
		t.Fatal(err)
	}
	if review.ClusterID != clusters[0].ID || len(review.RecordingIDs) != 2 || review.ReviewedAt == "" {
		t.Fatalf("unexpected review: %#v", review)
	}

	reopened, err := openRecordingStore(path)
	if err != nil {
		t.Fatal(err)
	}
	reloaded := reopened.clusters(99)
	if len(reloaded) != 1 || reloaded[0].Review == nil {
		t.Fatalf("expected persisted review, got %#v", reloaded)
	}
	if reloaded[0].Review.Classification != clusterReviewSameTransmission || reloaded[0].Review.Notes != "same call-up and groups" {
		t.Fatalf("unexpected persisted review: %#v", reloaded[0].Review)
	}
}

func TestClusterReviewValidation(t *testing.T) {
	s := &recordingStore{path: filepath.Join(t.TempDir(), "recordings.json"), data: recordingData{ClusterReviews: []clusterReview{}}}
	cluster := recordingCluster{ID: "cluster", Members: []clusterMember{{RecordingID: "a"}, {RecordingID: "b"}}}
	if _, err := s.setClusterReview(cluster, "not a class", ""); err == nil {
		t.Fatal("expected invalid classification to fail")
	}
}

func TestStableClusterIDTracksMembership(t *testing.T) {
	a := []clusterMember{{RecordingID: "b"}, {RecordingID: "a"}}
	b := []clusterMember{{RecordingID: "a"}, {RecordingID: "b"}}
	c := []clusterMember{{RecordingID: "a"}, {RecordingID: "b"}, {RecordingID: "c"}}
	if stableClusterID(a) != stableClusterID(b) {
		t.Fatal("cluster ID must be independent of member order")
	}
	if stableClusterID(a) == stableClusterID(c) {
		t.Fatal("cluster ID must change when membership changes")
	}
}

func TestDeleteClusterReview(t *testing.T) {
	path := filepath.Join(t.TempDir(), "recordings.json")
	s := &recordingStore{path: path, data: recordingData{ClusterReviews: []clusterReview{{ClusterID: "cluster", Classification: clusterReviewFalsePositive}}}}
	if err := s.deleteClusterReview("cluster"); err != nil {
		t.Fatal(err)
	}
	if len(s.data.ClusterReviews) != 0 {
		t.Fatalf("expected review deletion, got %#v", s.data.ClusterReviews)
	}
}
