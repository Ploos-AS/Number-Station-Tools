package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func portableArchiveTestStore(t *testing.T) (*recordingStore, string, []byte) {
	t.Helper()
	audioDir := filepath.Join(t.TempDir(), "audio")
	if err := os.MkdirAll(audioDir, 0o750); err != nil {
		t.Fatal(err)
	}
	audio := []byte("portable archive audio payload")
	path := filepath.Join(audioDir, "r1.wav")
	if err := os.WriteFile(path, audio, 0o600); err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(audio)
	rs, err := openRecordingStore(filepath.Join(t.TempDir(), "recordings.json"))
	if err != nil {
		t.Fatal(err)
	}
	rs.data.Recordings = []recording{{
		ID: "r1", ObservationID: "o1", Path: "audio/r1.wav", Format: "wav",
		SizeBytes: int64(len(audio)), SHA256: strings.ToUpper(hex.EncodeToString(h[:])), Managed: true,
		OriginalName: "capture.wav", Waveform: []uint8{1, 2}, Frequency: []uint8{3}, Spectrogram: []uint8{4}, Fingerprint: []uint8{5},
	}}
	rs.data.Annotations = []recordingAnnotation{{ID: "a1", RecordingID: "r1", StartMS: 100, Type: "", Label: "Call"}}
	return rs, audioDir, audio
}

func TestPortableArchiveContainsManifestAndVerifiedAudio(t *testing.T) {
	rs, audioDir, audio := portableArchiveTestStore(t)
	var buf bytes.Buffer
	summary, err := writePortableArchive(&buf, rs, audioDir)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Version != 1 || summary.ManagedFiles != 1 || summary.RecordingItems != 1 {
		t.Fatalf("summary = %#v", summary)
	}

	gz, err := gzip.NewReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	entries := map[string][]byte{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		entries[hdr.Name] = body
	}
	if !bytes.Equal(entries["audio/r1.wav"], audio) {
		t.Fatalf("archived audio = %q", entries["audio/r1.wav"])
	}
	var manifest recordingBundleManifest
	if err := json.Unmarshal(entries["manifest.json"], &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Items) != 1 || len(manifest.Items[0].Annotations) != 1 {
		t.Fatalf("manifest = %#v", manifest)
	}
	rec := manifest.Items[0].Recording
	if rec.Waveform != nil || rec.Frequency != nil || rec.Spectrogram != nil || rec.Fingerprint != nil {
		t.Fatalf("manifest contains derived payloads: %#v", rec)
	}
	if rec.SHA256 != strings.ToLower(rec.SHA256) || manifest.Items[0].Annotations[0].Type != annotationTypeOther {
		t.Fatalf("manifest normalization failed: %#v", manifest.Items[0])
	}
}

func TestPortableArchiveRejectsTamperedManagedAudio(t *testing.T) {
	rs, audioDir, _ := portableArchiveTestStore(t)
	if err := os.WriteFile(filepath.Join(audioDir, "r1.wav"), []byte("tampered archive audio payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := writePortableArchive(&buf, rs, audioDir); err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("expected integrity mismatch, got %v", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("archive wrote %d bytes before failed preflight", buf.Len())
	}
}

func TestPortableArchiveRejectsUnsafeManagedPath(t *testing.T) {
	rs, audioDir, _ := portableArchiveTestStore(t)
	rs.data.Recordings[0].Path = "../r1.wav"
	var buf bytes.Buffer
	if _, err := writePortableArchive(&buf, rs, audioDir); err == nil || !strings.Contains(err.Error(), "outside audio directory") {
		t.Fatalf("expected unsafe path error, got %v", err)
	}
}

func TestPortableArchiveAPI(t *testing.T) {
	rs, audioDir, _ := portableArchiveTestStore(t)
	mux := http.NewServeMux()
	registerRecordingBundleHandlers(mux, rs, audioDir)

	req := httptest.NewRequest(http.MethodGet, "/api/recording-archive", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Content-Type"); got != "application/gzip" {
		t.Fatalf("content type=%q", got)
	}
	if !strings.Contains(w.Header().Get("Content-Disposition"), "number-station-archive.tar.gz") {
		t.Fatalf("content disposition=%q", w.Header().Get("Content-Disposition"))
	}

	rs.data.Recordings[0].SHA256 = strings.Repeat("0", 64)
	req = httptest.NewRequest(http.MethodGet, "/api/recording-archive", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("tampered preflight status=%d body=%s", w.Code, w.Body.String())
	}
}
