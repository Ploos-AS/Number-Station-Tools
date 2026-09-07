package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func testFLAC(sampleRate int, channels int, totalSamples uint64) []byte {
	buf := new(bytes.Buffer)
	buf.WriteString("fLaC")
	buf.Write([]byte{0x80, 0x00, 0x00, 34})
	streamInfo := make([]byte, 34)
	binary.BigEndian.PutUint16(streamInfo[0:2], 4096)
	binary.BigEndian.PutUint16(streamInfo[2:4], 4096)
	packed := uint64(sampleRate)<<44 |
		uint64(channels-1)<<41 |
		uint64(15)<<36 |
		(totalSamples & ((uint64(1) << 36) - 1))
	binary.BigEndian.PutUint64(streamInfo[10:18], packed)
	buf.Write(streamInfo)
	return buf.Bytes()
}

func TestExtractFLACMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.flac")
	if err := os.WriteFile(path, testFLAC(44100, 2, 88200), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := extractAudioMetadata(path, "flac")
	if err != nil {
		t.Fatal(err)
	}
	if got.SampleRateHz != 44100 || got.Channels != 2 || got.DurationMS != 2000 {
		t.Fatalf("metadata = %#v", got)
	}
}

func TestExtractWAVMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.wav")
	if err := os.WriteFile(path, testWAV(48000, 2, 1500), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := extractAudioMetadata(path, "wav")
	if err != nil {
		t.Fatal(err)
	}
	if got.SampleRateHz != 48000 || got.Channels != 2 || got.DurationMS != 1500 {
		t.Fatalf("metadata = %#v", got)
	}
}
