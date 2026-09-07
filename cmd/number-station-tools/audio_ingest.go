package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const maxAudioUploadBytes int64 = 256 << 20

func registerAudioHandlers(mux *http.ServeMux, db *store, rs *recordingStore, audioDir string) {
	mux.HandleFunc("POST /api/audio", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxAudioUploadBytes)
		if err := r.ParseMultipartForm(maxAudioUploadBytes); err != nil {
			http.Error(w, "invalid or oversized multipart upload", http.StatusBadRequest)
			return
		}
		observationID := strings.TrimSpace(r.FormValue("observation_id"))
		if observationID == "" || !observationExists(db, observationID) {
			http.Error(w, "observation does not exist", http.StatusBadRequest)
			return
		}
		src, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "file is required", http.StatusBadRequest)
			return
		}
		defer src.Close()
		format, ext, err := audioFormatFromFilename(header.Filename)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := os.MkdirAll(audioDir, 0o750); err != nil {
			http.Error(w, "cannot create audio storage", http.StatusInternalServerError)
			return
		}
		recordingID := id()
		filename := recordingID + ext
		finalPath := filepath.Join(audioDir, filename)
		tmpPath := finalPath + ".tmp"
		dst, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			http.Error(w, "cannot create audio file", http.StatusInternalServerError)
			return
		}
		cleanup := true
		defer func() {
			_ = dst.Close()
			if cleanup {
				_ = os.Remove(tmpPath)
			}
		}()
		h := sha256.New()
		size, err := io.Copy(io.MultiWriter(dst, h), src)
		if err != nil {
			http.Error(w, "cannot store audio file", http.StatusInternalServerError)
			return
		}
		if err := dst.Sync(); err != nil {
			http.Error(w, "cannot sync audio file", http.StatusInternalServerError)
			return
		}
		if err := dst.Close(); err != nil {
			http.Error(w, "cannot close audio file", http.StatusInternalServerError)
			return
		}
		if err := validateAudioMagic(tmpPath, format); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		metadata, err := extractAudioMetadata(tmpPath, format)
		if err != nil {
			http.Error(w, "cannot extract audio metadata: "+err.Error(), http.StatusBadRequest)
			return
		}
		waveform, err := buildWaveformPreview(tmpPath, format)
		if err != nil {
			http.Error(w, "cannot build waveform preview: "+err.Error(), http.StatusBadRequest)
			return
		}
		frequency, err := buildFrequencyPreview(tmpPath, format)
		if err != nil {
			http.Error(w, "cannot build frequency preview: "+err.Error(), http.StatusBadRequest)
			return
		}
		spectrogram, err := buildSpectrogramPreview(tmpPath, format)
		if err != nil {
			http.Error(w, "cannot build spectrogram preview: "+err.Error(), http.StatusBadRequest)
			return
		}
		fingerprint := buildSignalFingerprint(frequency.Bins, spectrogram.Data, spectrogram.TimeBins, spectrogram.FrequencyBins)
		if err := os.Rename(tmpPath, finalPath); err != nil {
			http.Error(w, "cannot finalize audio file", http.StatusInternalServerError)
			return
		}
		cleanup = false
		rec := recording{
			ID: recordingID, ObservationID: observationID, Path: filepath.ToSlash(filepath.Join("audio", filename)), Format: format,
			SizeBytes: size, DurationMS: metadata.DurationMS, SampleRateHz: metadata.SampleRateHz, Channels: metadata.Channels,
			SHA256: hex.EncodeToString(h.Sum(nil)), Notes: strings.TrimSpace(r.FormValue("notes")), Managed: true, OriginalName: filepath.Base(header.Filename),
			Waveform: waveform, Frequency: frequency.Bins, FrequencyMaxHz: frequency.MaxHz,
			Spectrogram: spectrogram.Data, SpectrogramTimeBins: spectrogram.TimeBins, SpectrogramFreqBins: spectrogram.FrequencyBins, SpectrogramMaxHz: spectrogram.MaxHz,
			Fingerprint: fingerprint,
		}
		if err := rs.add(db, rec); err != nil {
			_ = os.Remove(finalPath)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusCreated, rec)
	})
	mux.HandleFunc("GET /api/recordings/{id}/file", func(w http.ResponseWriter, r *http.Request) {
		rec, ok := rs.byID(r.PathValue("id"))
		if !ok {
			http.Error(w, "recording does not exist", http.StatusNotFound)
			return
		}
		if !rec.Managed {
			http.Error(w, "recording is external metadata only", http.StatusConflict)
			return
		}
		path := filepath.Join(audioDir, filepath.Base(rec.Path))
		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				http.Error(w, "managed audio file is missing", http.StatusNotFound)
				return
			}
			http.Error(w, "cannot read managed audio file", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", safeDownloadName(rec)))
		http.ServeFile(w, r, path)
	})
}

func audioFormatFromFilename(name string) (format string, ext string, err error) {
	ext = strings.ToLower(filepath.Ext(filepath.Base(name)))
	switch ext {
	case ".wav":
		return "wav", ".wav", nil
	case ".flac":
		return "flac", ".flac", nil
	default:
		return "", "", errors.New("audio file must use .wav or .flac")
	}
}

func validateAudioMagic(path, format string) error {
	f, err := os.Open(path)
	if err != nil {
		return errors.New("cannot inspect audio file")
	}
	defer f.Close()
	buf := make([]byte, 12)
	n, err := io.ReadFull(f, buf)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return errors.New("cannot inspect audio file")
	}
	buf = buf[:n]
	switch format {
	case "wav":
		if len(buf) < 12 || !bytes.Equal(buf[:4], []byte("RIFF")) || !bytes.Equal(buf[8:12], []byte("WAVE")) {
			return errors.New("file does not contain a WAV signature")
		}
	case "flac":
		if len(buf) < 4 || !bytes.Equal(buf[:4], []byte("fLaC")) {
			return errors.New("file does not contain a FLAC signature")
		}
	default:
		return errors.New("unsupported audio format")
	}
	return nil
}

func safeDownloadName(rec recording) string {
	name := filepath.Base(strings.TrimSpace(rec.OriginalName))
	if name == "." || name == "" {
		name = filepath.Base(rec.Path)
	}
	return name
}
