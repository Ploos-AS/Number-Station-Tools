package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const portableArchiveVersion = 1

type portableArchiveSummary struct {
	Version        int `json:"version"`
	ManagedFiles   int `json:"managed_files"`
	RecordingItems int `json:"recording_items"`
}

type verifiedArchiveFile struct {
	path        string
	archivePath string
	info        os.FileInfo
}

type portableArchivePlan struct {
	manifest       []byte
	files          []verifiedArchiveFile
	recordingItems int
}

func managedArchivePath(rec recording) (string, error) {
	if !rec.Managed {
		return "", nil
	}
	base := filepath.Base(strings.TrimSpace(rec.Path))
	if base == "." || base == "" {
		return "", errors.New("managed recording path is empty")
	}
	want := filepath.ToSlash(filepath.Join("audio", base))
	if filepath.ToSlash(rec.Path) != want {
		return "", errors.New("managed recording path is outside audio directory")
	}
	return want, nil
}

func verifyManagedArchiveFile(audioDir string, rec recording) (string, os.FileInfo, error) {
	archivePath, err := managedArchivePath(rec)
	if err != nil {
		return "", nil, err
	}
	path := filepath.Join(audioDir, filepath.Base(archivePath))
	info, err := os.Lstat(path)
	if err != nil {
		return "", nil, fmt.Errorf("managed audio %s unavailable: %w", rec.ID, err)
	}
	if !info.Mode().IsRegular() {
		return "", nil, fmt.Errorf("managed audio %s is not a regular file", rec.ID)
	}
	if rec.SizeBytes > 0 && info.Size() != rec.SizeBytes {
		return "", nil, fmt.Errorf("managed audio %s size mismatch", rec.ID)
	}

	wantHash := strings.ToLower(strings.TrimSpace(rec.SHA256))
	if len(wantHash) != 64 {
		return "", nil, fmt.Errorf("managed audio %s has no valid SHA-256", rec.ID)
	}
	f, err := os.Open(path)
	if err != nil {
		return "", nil, fmt.Errorf("managed audio %s cannot be opened: %w", rec.ID, err)
	}
	h := sha256.New()
	_, copyErr := io.Copy(h, f)
	closeErr := f.Close()
	if copyErr != nil {
		return "", nil, fmt.Errorf("managed audio %s cannot be hashed: %w", rec.ID, copyErr)
	}
	if closeErr != nil {
		return "", nil, fmt.Errorf("managed audio %s cannot be closed: %w", rec.ID, closeErr)
	}
	if hex.EncodeToString(h.Sum(nil)) != wantHash {
		return "", nil, fmt.Errorf("managed audio %s SHA-256 mismatch", rec.ID)
	}
	return path, info, nil
}

func preparePortableArchive(rs *recordingStore, audioDir string) (portableArchivePlan, error) {
	bundle := rs.exportRecordingBundle()
	manifest, err := marshalRecordingBundle(bundle)
	if err != nil {
		return portableArchivePlan{}, err
	}
	verified := make([]verifiedArchiveFile, 0)
	for _, item := range bundle.Items {
		if !item.Recording.Managed {
			continue
		}
		archivePath, err := managedArchivePath(item.Recording)
		if err != nil {
			return portableArchivePlan{}, fmt.Errorf("recording %s: %w", item.Recording.ID, err)
		}
		path, info, err := verifyManagedArchiveFile(audioDir, item.Recording)
		if err != nil {
			return portableArchivePlan{}, err
		}
		verified = append(verified, verifiedArchiveFile{path: path, archivePath: archivePath, info: info})
	}
	return portableArchivePlan{manifest: manifest, files: verified, recordingItems: len(bundle.Items)}, nil
}

func writePreparedPortableArchive(dst io.Writer, plan portableArchivePlan) (portableArchiveSummary, error) {
	gz := gzip.NewWriter(dst)
	tw := tar.NewWriter(gz)
	closeWithError := func(err error) (portableArchiveSummary, error) {
		_ = tw.Close()
		_ = gz.Close()
		return portableArchiveSummary{}, err
	}

	zeroTime := time.Unix(0, 0).UTC()
	if err := tw.WriteHeader(&tar.Header{Name: "manifest.json", Mode: 0o644, Size: int64(len(plan.manifest)), ModTime: zeroTime}); err != nil {
		return closeWithError(err)
	}
	if _, err := tw.Write(plan.manifest); err != nil {
		return closeWithError(err)
	}

	for _, file := range plan.files {
		header := &tar.Header{Name: file.archivePath, Mode: 0o644, Size: file.info.Size(), ModTime: zeroTime}
		if err := tw.WriteHeader(header); err != nil {
			return closeWithError(err)
		}
		f, err := os.Open(file.path)
		if err != nil {
			return closeWithError(err)
		}
		_, copyErr := io.Copy(tw, f)
		closeErr := f.Close()
		if copyErr != nil {
			return closeWithError(copyErr)
		}
		if closeErr != nil {
			return closeWithError(closeErr)
		}
	}
	if err := tw.Close(); err != nil {
		_ = gz.Close()
		return portableArchiveSummary{}, err
	}
	if err := gz.Close(); err != nil {
		return portableArchiveSummary{}, err
	}
	return portableArchiveSummary{Version: portableArchiveVersion, ManagedFiles: len(plan.files), RecordingItems: plan.recordingItems}, nil
}

func writePortableArchive(dst io.Writer, rs *recordingStore, audioDir string) (portableArchiveSummary, error) {
	plan, err := preparePortableArchive(rs, audioDir)
	if err != nil {
		return portableArchiveSummary{}, err
	}
	return writePreparedPortableArchive(dst, plan)
}
