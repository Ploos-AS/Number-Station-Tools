package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
)

const (
	maxPortableArchiveUploadBytes int64 = 1 << 30
	maxPortableArchiveExpanded   int64 = 2 << 30
	maxPortableArchiveEntries          = 10002
	maxPortableManifestBytes     int64 = 16 << 20
)

type portableRestoreResult struct {
	ImportedRecords    int `json:"imported_recordings"`
	ImportedAnnotations int `json:"imported_annotations"`
	Duplicates         int `json:"duplicates"`
	Conflicts          int `json:"conflicts"`
	Unmatched          int `json:"unmatched"`
}

type stagedRestoreFile struct {
	archivePath string
	path        string
	size        int64
	sha256      string
}

type stagedPortableArchive struct {
	dir      string
	manifest recordingBundleManifest
	files    map[string]stagedRestoreFile
}

func readPortableArchive(src io.Reader, audioDir string) (stagedPortableArchive, error) {
	if err := os.MkdirAll(audioDir, 0o750); err != nil {
		return stagedPortableArchive{}, errors.New("cannot create audio storage")
	}
	stageDir, err := os.MkdirTemp(audioDir, ".restore-")
	if err != nil {
		return stagedPortableArchive{}, errors.New("cannot create restore staging directory")
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(stageDir)
		}
	}()

	gz, err := gzip.NewReader(src)
	if err != nil {
		return stagedPortableArchive{}, errors.New("invalid gzip archive")
	}
	defer gz.Close()
	tr := tar.NewReader(gz)

	files := make(map[string]stagedRestoreFile)
	var manifestBytes []byte
	var expanded int64
	entries := 0
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return stagedPortableArchive{}, errors.New("invalid tar archive")
		}
		entries++
		if entries > maxPortableArchiveEntries {
			return stagedPortableArchive{}, errors.New("archive contains too many entries")
		}
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
			return stagedPortableArchive{}, fmt.Errorf("archive entry %q is not a regular file", hdr.Name)
		}
		name := path.Clean(strings.TrimSpace(hdr.Name))
		if name != hdr.Name || name == "." || strings.HasPrefix(name, "/") || strings.HasPrefix(name, "../") {
			return stagedPortableArchive{}, fmt.Errorf("unsafe archive path %q", hdr.Name)
		}
		if hdr.Size < 0 || expanded > maxPortableArchiveExpanded-hdr.Size {
			return stagedPortableArchive{}, errors.New("archive expanded size exceeds limit")
		}
		expanded += hdr.Size

		if name == "manifest.json" {
			if manifestBytes != nil {
				return stagedPortableArchive{}, errors.New("archive contains duplicate manifest.json")
			}
			if hdr.Size > maxPortableManifestBytes {
				return stagedPortableArchive{}, errors.New("manifest.json exceeds size limit")
			}
			manifestBytes = make([]byte, hdr.Size)
			if _, err := io.ReadFull(tr, manifestBytes); err != nil {
				return stagedPortableArchive{}, errors.New("cannot read manifest.json")
			}
			continue
		}

		if !strings.HasPrefix(name, "audio/") || path.Base(name) == "." || name != "audio/"+path.Base(name) {
			return stagedPortableArchive{}, fmt.Errorf("unexpected archive entry %q", name)
		}
		if _, exists := files[name]; exists {
			return stagedPortableArchive{}, fmt.Errorf("duplicate archive entry %q", name)
		}
		if hdr.Size > maxAudioUploadBytes {
			return stagedPortableArchive{}, fmt.Errorf("audio entry %q exceeds size limit", name)
		}
		tmpPath := filepath.Join(stageDir, path.Base(name))
		f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return stagedPortableArchive{}, errors.New("cannot create staged audio file")
		}
		h := sha256.New()
		written, copyErr := io.CopyN(io.MultiWriter(f, h), tr, hdr.Size)
		closeErr := f.Close()
		if copyErr != nil || written != hdr.Size {
			return stagedPortableArchive{}, fmt.Errorf("cannot extract %q", name)
		}
		if closeErr != nil {
			return stagedPortableArchive{}, fmt.Errorf("cannot close staged %q", name)
		}
		files[name] = stagedRestoreFile{archivePath: name, path: tmpPath, size: written, sha256: hex.EncodeToString(h.Sum(nil))}
	}
	if len(manifestBytes) == 0 {
		return stagedPortableArchive{}, errors.New("archive is missing manifest.json")
	}
	var manifest recordingBundleManifest
	decoder := json.NewDecoder(strings.NewReader(string(manifestBytes)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return stagedPortableArchive{}, errors.New("invalid manifest.json")
	}
	if manifest.Version != recordingBundleVersion {
		return stagedPortableArchive{}, errors.New("unsupported recording bundle version")
	}
	if len(manifest.Items) > 10000 {
		return stagedPortableArchive{}, errors.New("recording bundle exceeds 10000 items")
	}
	cleanup = false
	return stagedPortableArchive{dir: stageDir, manifest: manifest, files: files}, nil
}

func samePortableRecording(a, b recording) bool {
	a = portableRecording(a)
	b = portableRecording(b)
	a.SHA256 = strings.ToLower(strings.TrimSpace(a.SHA256))
	b.SHA256 = strings.ToLower(strings.TrimSpace(b.SHA256))
	return reflect.DeepEqual(a, b)
}

func prepareRestoredRecording(rec recording, staged stagedRestoreFile) (recording, error) {
	rec = portableRecording(rec)
	rec.Format = strings.ToLower(strings.TrimSpace(rec.Format))
	rec.SHA256 = strings.ToLower(strings.TrimSpace(rec.SHA256))
	if err := validateRecording(rec); err != nil {
		return recording{}, err
	}
	if staged.size != rec.SizeBytes {
		return recording{}, errors.New("audio size does not match manifest")
	}
	if staged.sha256 != rec.SHA256 {
		return recording{}, errors.New("audio SHA-256 does not match manifest")
	}
	if err := validateAudioMagic(staged.path, rec.Format); err != nil {
		return recording{}, err
	}
	metadata, err := extractAudioMetadata(staged.path, rec.Format)
	if err != nil {
		return recording{}, fmt.Errorf("cannot extract restored audio metadata: %w", err)
	}
	waveform, err := buildWaveformPreview(staged.path, rec.Format)
	if err != nil {
		return recording{}, fmt.Errorf("cannot rebuild waveform preview: %w", err)
	}
	frequency, err := buildFrequencyPreview(staged.path, rec.Format)
	if err != nil {
		return recording{}, fmt.Errorf("cannot rebuild frequency preview: %w", err)
	}
	spectrogram, err := buildSpectrogramPreview(staged.path, rec.Format)
	if err != nil {
		return recording{}, fmt.Errorf("cannot rebuild spectrogram preview: %w", err)
	}
	rec.DurationMS = metadata.DurationMS
	rec.SampleRateHz = metadata.SampleRateHz
	rec.Channels = metadata.Channels
	rec.Waveform = waveform
	rec.Frequency = frequency.Bins
	rec.FrequencyMaxHz = frequency.MaxHz
	rec.Spectrogram = spectrogram.Bins
	rec.SpectrogramTimeBins = spectrogram.TimeBins
	rec.SpectrogramFreqBins = spectrogram.FrequencyBins
	rec.SpectrogramMaxHz = spectrogram.MaxHz
	rec.Fingerprint = buildSignalFingerprint(frequency.Bins, spectrogram.Bins, spectrogram.TimeBins, spectrogram.FrequencyBins)
	return rec, nil
}

func restorePortableArchive(src io.Reader, db *store, rs *recordingStore, audioDir string) (portableRestoreResult, error) {
	staged, err := readPortableArchive(src, audioDir)
	if err != nil {
		return portableRestoreResult{}, err
	}
	defer os.RemoveAll(staged.dir)

	result := portableRestoreResult{}
	seenIDs := make(map[string]struct{}, len(staged.manifest.Items))
	requiredFiles := make(map[string]struct{})
	type preparedItem struct {
		recording   recording
		annotations []recordingAnnotation
		staged      *stagedRestoreFile
		duplicate   bool
	}
	prepared := make([]preparedItem, 0, len(staged.manifest.Items))

	for _, item := range staged.manifest.Items {
		rec := portableRecording(item.Recording)
		rec.ID = strings.TrimSpace(rec.ID)
		if rec.ID == "" {
			return portableRestoreResult{}, errors.New("manifest contains recording without id")
		}
		if _, exists := seenIDs[rec.ID]; exists {
			return portableRestoreResult{}, fmt.Errorf("manifest contains duplicate recording id %s", rec.ID)
		}
		seenIDs[rec.ID] = struct{}{}
		if !observationExists(db, rec.ObservationID) {
			result.Unmatched++
			continue
		}
		var stagedFile *stagedRestoreFile
		if rec.Managed {
			archivePath, err := managedArchivePath(rec)
			if err != nil {
				return portableRestoreResult{}, fmt.Errorf("recording %s: %w", rec.ID, err)
			}
			file, ok := staged.files[archivePath]
			if !ok {
				return portableRestoreResult{}, fmt.Errorf("managed recording %s is missing %s", rec.ID, archivePath)
			}
			requiredFiles[archivePath] = struct{}{}
			rebuilt, err := prepareRestoredRecording(rec, file)
			if err != nil {
				return portableRestoreResult{}, fmt.Errorf("recording %s: %w", rec.ID, err)
			}
			rec = rebuilt
			copy := file
			stagedFile = &copy
		} else if err := validateRecording(rec); err != nil {
			return portableRestoreResult{}, fmt.Errorf("recording %s: %w", rec.ID, err)
		}
		prepared = append(prepared, preparedItem{recording: rec, annotations: item.Annotations, staged: stagedFile})
	}
	for name := range staged.files {
		if _, ok := requiredFiles[name]; !ok {
			return portableRestoreResult{}, fmt.Errorf("archive contains unreferenced audio entry %q", name)
		}
	}

	rs.mu.Lock()
	defer rs.mu.Unlock()
	existingByID := make(map[string]recording, len(rs.data.Recordings))
	for _, rec := range rs.data.Recordings {
		existingByID[rec.ID] = rec
	}
	pendingRecordings := make([]recording, 0)
	pendingAnnotations := make([]recordingAnnotation, 0)
	type move struct{ from, to string }
	moves := make([]move, 0)

	for i := range prepared {
		entry := &prepared[i]
		target := entry.recording
		if existing, ok := existingByID[target.ID]; ok {
			if samePortableRecording(existing, target) {
				entry.duplicate = true
				result.Duplicates++
				target = existing
			} else {
				result.Conflicts++
				continue
			}
		} else {
			if entry.staged != nil {
				dest := filepath.Join(audioDir, filepath.Base(entry.staged.archivePath))
				if _, err := os.Lstat(dest); err == nil {
					result.Conflicts++
					continue
				} else if !errors.Is(err, os.ErrNotExist) {
					return portableRestoreResult{}, fmt.Errorf("cannot inspect destination for %s", target.ID)
				}
				moves = append(moves, move{from: entry.staged.path, to: dest})
			}
			pendingRecordings = append(pendingRecordings, target)
			existingByID[target.ID] = target
			result.ImportedRecords++
		}

		for _, incoming := range entry.annotations {
			annotation := incoming
			annotation.ID = ""
			annotation.RecordingID = target.ID
			annotation.Type = normalizeAnnotationType(annotation.Type)
			annotation.Label = strings.TrimSpace(annotation.Label)
			annotation.Notes = strings.TrimSpace(annotation.Notes)
			if err := validateRecordingAnnotation(target, annotation); err != nil {
				return portableRestoreResult{}, fmt.Errorf("invalid annotation for recording %s: %w", target.ID, err)
			}
			duplicate := false
			for _, existing := range rs.data.Annotations {
				if sameAnnotation(existing, annotation) {
					duplicate = true
					break
				}
			}
			if !duplicate {
				for _, existing := range pendingAnnotations {
					if sameAnnotation(existing, annotation) {
						duplicate = true
						break
					}
				}
			}
			if duplicate {
				result.Duplicates++
				continue
			}
			annotation.ID = id()
			pendingAnnotations = append(pendingAnnotations, annotation)
			result.ImportedAnnotations++
		}
	}

	moved := make([]move, 0, len(moves))
	for _, mv := range moves {
		if err := os.Rename(mv.from, mv.to); err != nil {
			for j := len(moved) - 1; j >= 0; j-- {
				_ = os.Rename(moved[j].to, moved[j].from)
			}
			return portableRestoreResult{}, errors.New("cannot finalize restored audio files")
		}
		moved = append(moved, mv)
	}
	oldRecordings := len(rs.data.Recordings)
	oldAnnotations := len(rs.data.Annotations)
	rs.data.Recordings = append(rs.data.Recordings, pendingRecordings...)
	rs.data.Annotations = append(rs.data.Annotations, pendingAnnotations...)
	if err := rs.save(); err != nil {
		rs.data.Recordings = rs.data.Recordings[:oldRecordings]
		rs.data.Annotations = rs.data.Annotations[:oldAnnotations]
		for j := len(moved) - 1; j >= 0; j-- {
			_ = os.Rename(moved[j].to, moved[j].from)
		}
		return portableRestoreResult{}, err
	}
	return result, nil
}
