package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecordingBundleExport(t *testing.T) {
	_, rs := annotationTestStore(t)
	rs.data.Recordings[0].SHA256 = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	rs.data.Recordings[0].Waveform = []uint8{1, 2, 3}
	rs.data.Recordings[0].Frequency = []uint8{4, 5}
	rs.data.Recordings[0].Spectrogram = []uint8{6, 7}
	rs.data.Recordings[0].Fingerprint = []uint8{8, 9}
	if _, err := rs.addAnnotation("r1", recordingAnnotation{StartMS: 1250, Type: annotationTypeStationID, Label: "ID"}); err != nil {
		t.Fatal(err)
	}

	bundle := rs.exportRecordingBundle()
	if bundle.Version != recordingBundleVersion || len(bundle.Items) != 1 {
		t.Fatalf("bundle = %#v", bundle)
	}
	item := bundle.Items[0]
	if item.Recording.ID != "r1" || item.Recording.SHA256 != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("recording = %#v", item.Recording)
	}
	if len(item.Recording.Waveform) != 0 || len(item.Recording.Frequency) != 0 || len(item.Recording.Spectrogram) != 0 || len(item.Recording.Fingerprint) != 0 {
		t.Fatalf("derived preview payload leaked into bundle: %#v", item.Recording)
	}
	if len(item.Annotations) != 1 || item.Annotations[0].Type != annotationTypeStationID {
		t.Fatalf("annotations = %#v", item.Annotations)
	}
}

func TestRecordingBundleAPI(t *testing.T) {
	db, rs := annotationTestStore(t)
	mux := http.NewServeMux()
	registerRecordingBundleHandlers(mux, db, rs, t.TempDir())
	req := httptest.NewRequest(http.MethodGet, "/api/recording-bundle", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Content-Disposition"); got == "" {
		t.Fatal("missing Content-Disposition")
	}
	var bundle recordingBundleManifest
	if err := json.Unmarshal(w.Body.Bytes(), &bundle); err != nil {
		t.Fatal(err)
	}
	if bundle.Version != recordingBundleVersion || len(bundle.Items) != 1 {
		t.Fatalf("bundle = %#v", bundle)
	}
}
