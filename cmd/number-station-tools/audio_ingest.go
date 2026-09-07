package main

import (
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
		if err := os.Rename(tmpPath, finalPath); err != nil {
			http.Error(w, "cannot finalize audio file", http.StatusInternalServerError)
			return
		}
		cleanup = false

		rec := recording{
			ID:            recordingID,
			ObservationID: observationID,
			Path:          filepath.ToSlash(filepath.Join("audio", filename)),
			Format:        format,
			SizeBytes:     size,
			SHA256:        hex.EncodeToString(h.Sum(nil)),
			Notes:         strings.TrimSpace(r.FormValue("notes")),
			Managed:       true,
			OriginalName:  filepath.Base(header.Filename),
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
	ext := strings.ToLower(filepath.Ext(filepath.Base(name)))
	switch ext {
	case ".wav":
		return "wav", ".wav", nil
	case ".flac":
		return "flac", ".flac", nil
	default:
		return "", "", errors.New("audio file must use .wav or .flac")
	}
}

func safeDownloadName(rec recording) string {
	name := filepath.Base(strings.TrimSpace(rec.OriginalName))
	if name == "." || name == "" {
		name = filepath.Base(rec.Path)
	}
	return name
}
