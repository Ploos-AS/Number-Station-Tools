package main

import (
	"errors"
	"sort"
	"strings"
	"time"
)

const (
	clusterReviewSameTransmission = "same transmission"
	clusterReviewSameStation      = "same station"
	clusterReviewFalsePositive    = "false positive"
	clusterReviewDuplicateCapture = "duplicate capture"
)

type clusterReview struct {
	ClusterID          string   `json:"cluster_id"`
	Classification     string   `json:"classification"`
	Notes              string   `json:"notes,omitempty"`
	RecordingIDs       []string `json:"recording_ids"`
	ReviewedAt         string   `json:"reviewed_at"`
}

func validateClusterReviewClassification(v string) (string, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	switch v {
	case clusterReviewSameTransmission, clusterReviewSameStation, clusterReviewFalsePositive, clusterReviewDuplicateCapture:
		return v, nil
	default:
		return "", errors.New("classification must be same transmission, same station, false positive, or duplicate capture")
	}
}

func (s *recordingStore) setClusterReview(cluster recordingCluster, classification, notes string) (clusterReview, error) {
	classification, err := validateClusterReviewClassification(classification)
	if err != nil {
		return clusterReview{}, err
	}
	notes = strings.TrimSpace(notes)
	if len(notes) > 2000 {
		return clusterReview{}, errors.New("notes must not exceed 2000 characters")
	}
	ids := make([]string, 0, len(cluster.Members))
	for _, member := range cluster.Members {
		ids = append(ids, member.RecordingID)
	}
	sort.Strings(ids)
	review := clusterReview{
		ClusterID:      cluster.ID,
		Classification: classification,
		Notes:          notes,
		RecordingIDs:   ids,
		ReviewedAt:     time.Now().UTC().Format(time.RFC3339),
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.ClusterReviews {
		if s.data.ClusterReviews[i].ClusterID == cluster.ID {
			s.data.ClusterReviews[i] = review
			return review, s.save()
		}
	}
	s.data.ClusterReviews = append(s.data.ClusterReviews, review)
	return review, s.save()
}

func (s *recordingStore) deleteClusterReview(clusterID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.ClusterReviews {
		if s.data.ClusterReviews[i].ClusterID == clusterID {
			s.data.ClusterReviews = append(s.data.ClusterReviews[:i], s.data.ClusterReviews[i+1:]...)
			return s.save()
		}
	}
	return errors.New("cluster review does not exist")
}
