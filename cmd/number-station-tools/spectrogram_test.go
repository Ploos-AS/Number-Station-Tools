package main

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func testToneWAV(sampleRate int, durationMS int, frequencyHz float64) []byte {
	frames := sampleRate * durationMS / 1000
	data := make([]byte, frames*2)
	for i := 0; i < frames; i++ {
		v := int16(math.Sin(2*math.Pi*frequencyHz*float64(i)/float64(sampleRate)) * 28000)
		binary.LittleEndian.PutUint16(data[i*2:i*2+2], uint16(v))
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

func TestWAVSpectrogramPreview(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tone.wav")
	if err := os.WriteFile(path, testToneWAV(8000, 1000, 1000), 0o600); err != nil {
		t.Fatal(err)
	}
	preview, err := buildSpectrogramPreview(path, "wav")
	if err != nil {
		t.Fatal(err)
	}
	if preview.TimeBins != spectrogramTimeBins || preview.FrequencyBins != spectrogramFrequencyBins {
		t.Fatalf("dimensions = %dx%d", preview.TimeBins, preview.FrequencyBins)
	}
	if len(preview.Bins) != spectrogramTimeBins*spectrogramFrequencyBins {
		t.Fatalf("spectrogram bins = %d", len(preview.Bins))
	}
	if preview.MaxHz != 4000 {
		t.Fatalf("max Hz = %d", preview.MaxHz)
	}

	for timeBin := 0; timeBin < preview.TimeBins; timeBin++ {
		row := preview.Bins[timeBin*preview.FrequencyBins : (timeBin+1)*preview.FrequencyBins]
		maxIndex := 0
		for i := range row {
			if row[i] > row[maxIndex] {
				maxIndex = i
			}
		}
		binHz := float64(maxIndex+1) * float64(preview.MaxHz) / float64(preview.FrequencyBins)
		if math.Abs(binHz-1000) > 250 {
			t.Fatalf("time bin %d dominant frequency %.0f Hz", timeBin, binHz)
		}
	}
}

func TestFLACSpectrogramPreviewIsOptional(t *testing.T) {
	preview, err := buildSpectrogramPreview("unused", "flac")
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Bins) != 0 {
		t.Fatal("FLAC should not expose spectrogram data without decoding")
	}
}
