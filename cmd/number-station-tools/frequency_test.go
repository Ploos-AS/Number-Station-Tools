package main

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func sineWAV(sampleRate int, frequencyHz float64, durationMS int) []byte {
	frames := sampleRate * durationMS / 1000
	data := make([]byte, frames*2)
	for i := 0; i < frames; i++ {
		sample := int16(math.Sin(2*math.Pi*frequencyHz*float64(i)/float64(sampleRate)) * 28000)
		binary.LittleEndian.PutUint16(data[i*2:i*2+2], uint16(sample))
	}
	buf := new(bytes.Buffer)
	buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(36+len(data)))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate*2))
	_ = binary.Write(buf, binary.LittleEndian, uint16(2))
	_ = binary.Write(buf, binary.LittleEndian, uint16(16))
	buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, uint32(len(data)))
	buf.Write(data)
	return buf.Bytes()
}

func TestWAVFrequencyPreviewFindsTone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tone.wav")
	if err := os.WriteFile(path, sineWAV(8000, 1000, 500), 0o600); err != nil {
		t.Fatal(err)
	}
	preview, err := buildFrequencyPreview(path, "wav")
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Bins) != frequencyBins || preview.MaxHz != 4000 {
		t.Fatalf("preview shape = %d bins max=%d", len(preview.Bins), preview.MaxHz)
	}
	peakBin := 0
	for i := range preview.Bins {
		if preview.Bins[i] > preview.Bins[peakBin] {
			peakBin = i
		}
	}
	peakHz := float64(peakBin) * float64(preview.MaxHz) / float64(frequencyBins)
	if math.Abs(peakHz-1000) > 180 {
		t.Fatalf("dominant frequency ~= %.1f Hz, want about 1000 Hz (bin %d)", peakHz, peakBin)
	}
	if preview.Bins[peakBin] < 200 {
		t.Fatalf("dominant bin too weak: %d", preview.Bins[peakBin])
	}
}

func TestFLACFrequencyPreviewIsOptional(t *testing.T) {
	preview, err := buildFrequencyPreview("unused", "flac")
	if err != nil {
		t.Fatal(err)
	}
	if preview.Bins != nil || preview.MaxHz != 0 {
		t.Fatal("compressed FLAC should not expose synthetic frequency data without decoding")
	}
}
