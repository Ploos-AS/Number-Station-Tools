package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func testWaveWAV() []byte {
	data := make([]byte, 256)
	for i := 0; i < len(data)/2; i++ {
		value := int16((i%32)*1800 - 28000)
		binary.LittleEndian.PutUint16(data[i*2:i*2+2], uint16(value))
	}
	buf := new(bytes.Buffer)
	buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(36+len(data)))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint32(8000))
	_ = binary.Write(buf, binary.LittleEndian, uint32(16000))
	_ = binary.Write(buf, binary.LittleEndian, uint16(2))
	_ = binary.Write(buf, binary.LittleEndian, uint16(16))
	buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, uint32(len(data)))
	buf.Write(data)
	return buf.Bytes()
}

func TestWAVWaveformPreview(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preview.wav")
	if err := osWriteFile(path, testWaveWAV()); err != nil {
		t.Fatal(err)
	}
	preview, err := buildWaveformPreview(path, "wav")
	if err != nil {
		t.Fatal(err)
	}
	if len(preview) != waveformBins {
		t.Fatalf("waveform bins = %d, want %d", len(preview), waveformBins)
	}
	nonzero := false
	for _, value := range preview {
		if value > 0 {
			nonzero = true
			break
		}
	}
	if !nonzero {
		t.Fatal("waveform preview contains no signal")
	}
}

func TestManagedUploadPersistsWaveform(t *testing.T) {
	db, _ := openStore(filepath.Join(t.TempDir(), "data.json"))
	_ = db.addStation(station{ID: "s1", Name: "Test"})
	_ = db.addObservation(observation{ID: "o1", StationID: "s1", HeardAt: time.Now().UTC(), FrequencyHz: 1000})
	rs, _ := openRecordingStore(filepath.Join(t.TempDir(), "recordings.json"))
	mux := http.NewServeMux()
	registerAudioHandlers(mux, db, rs, filepath.Join(t.TempDir(), "audio"))

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("observation_id", "o1")
	part, _ := mw.CreateFormFile("file", "preview.wav")
	_, _ = part.Write(testWaveWAV())
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/audio", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("upload status = %d body=%s", w.Code, w.Body.String())
	}
	var rec recording
	if err := json.Unmarshal(w.Body.Bytes(), &rec); err != nil {
		t.Fatal(err)
	}
	if len(rec.Waveform) != waveformBins {
		t.Fatalf("response waveform bins = %d", len(rec.Waveform))
	}
	persisted, ok := rs.byID(rec.ID)
	if !ok || len(persisted.Waveform) != waveformBins {
		t.Fatal("waveform preview was not persisted")
	}
}

func TestFLACWaveformPreviewIsOptional(t *testing.T) {
	preview, err := buildWaveformPreview("unused", "flac")
	if err != nil {
		t.Fatal(err)
	}
	if preview != nil {
		t.Fatal("compressed FLAC should not expose a synthetic waveform without decoding")
	}
}

func osWriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o600)
}
