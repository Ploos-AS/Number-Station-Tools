package main

import "testing"

func TestFingerprintSimilarityRanksSimilarSignals(t *testing.T) {
	base := make([]uint8, 64)
	near := make([]uint8, 64)
	far := make([]uint8, 64)
	for i := 0; i < 64; i++ {
		if i >= 10 && i <= 14 {
			base[i] = 220
			near[i] = 210
		}
		if i >= 40 && i <= 44 {
			far[i] = 220
		}
	}
	a := buildSignalFingerprint(base, nil, 0, 0)
	b := buildSignalFingerprint(near, nil, 0, 0)
	c := buildSignalFingerprint(far, nil, 0, 0)
	if len(a) != fingerprintBins {
		t.Fatalf("fingerprint length=%d want %d", len(a), fingerprintBins)
	}
	ab := fingerprintSimilarity(a, b)
	ac := fingerprintSimilarity(a, c)
	if ab < 99 {
		t.Fatalf("similar signal score %.2f, want >=99", ab)
	}
	if ac >= ab {
		t.Fatalf("dissimilar score %.2f should be below similar %.2f", ac, ab)
	}
}

func TestRecordingStoreSimilarSortsDescending(t *testing.T) {
	rs := &recordingStore{data: recordingData{Recordings: []recording{
		{ID: "a", ObservationID: "o1", Path: "a.wav", Fingerprint: []uint8{255, 255, 0, 0}},
		{ID: "b", ObservationID: "o2", Path: "b.wav", Fingerprint: []uint8{250, 250, 0, 0}},
		{ID: "c", ObservationID: "o3", Path: "c.wav", Fingerprint: []uint8{0, 0, 255, 255}},
	}}}
	results, err := rs.similar("a", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].RecordingID != "b" {
		t.Fatalf("unexpected ranking: %+v", results)
	}
	if results[0].Score <= results[1].Score {
		t.Fatalf("scores not descending: %+v", results)
	}
}
