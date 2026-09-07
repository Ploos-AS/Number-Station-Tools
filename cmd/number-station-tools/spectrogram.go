package main

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"os"
)

const spectrogramTimeBins = 24
const spectrogramFrequencyBins = 32
const spectrogramWindowFrames = 256

type spectrogramPreview struct {
	Bins          []uint8
	TimeBins      int
	FrequencyBins int
	MaxHz         int
}

func buildSpectrogramPreview(path, format string) (spectrogramPreview, error) {
	switch format {
	case "wav":
		return wavSpectrogramPreview(path)
	case "flac":
		return spectrogramPreview{}, nil
	default:
		return spectrogramPreview{}, errors.New("unsupported audio format")
	}
}

func wavSpectrogramPreview(path string) (spectrogramPreview, error) {
	f, err := os.Open(path)
	if err != nil {
		return spectrogramPreview{}, err
	}
	defer f.Close()

	header := make([]byte, 12)
	if _, err := io.ReadFull(f, header); err != nil {
		return spectrogramPreview{}, errors.New("invalid WAV header")
	}
	if string(header[:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		return spectrogramPreview{}, errors.New("invalid WAV signature")
	}

	var audioFormat, channels, bits uint16
	var sampleRate uint32
	var data []byte
	for {
		chunkHeader := make([]byte, 8)
		if _, err := io.ReadFull(f, chunkHeader); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return spectrogramPreview{}, err
		}
		size := binary.LittleEndian.Uint32(chunkHeader[4:8])
		chunk := make([]byte, size)
		if _, err := io.ReadFull(f, chunk); err != nil {
			return spectrogramPreview{}, errors.New("truncated WAV chunk")
		}
		if size%2 == 1 {
			_, _ = f.Seek(1, io.SeekCurrent)
		}
		switch string(chunkHeader[:4]) {
		case "fmt ":
			if len(chunk) < 16 {
				return spectrogramPreview{}, errors.New("invalid WAV fmt chunk")
			}
			audioFormat = binary.LittleEndian.Uint16(chunk[0:2])
			channels = binary.LittleEndian.Uint16(chunk[2:4])
			sampleRate = binary.LittleEndian.Uint32(chunk[4:8])
			bits = binary.LittleEndian.Uint16(chunk[14:16])
		case "data":
			data = chunk
		}
	}
	if len(data) == 0 || channels == 0 || sampleRate == 0 {
		return spectrogramPreview{}, errors.New("WAV PCM metadata or data missing")
	}
	if audioFormat != 1 {
		return spectrogramPreview{}, nil
	}
	bytesPerSample := int(bits / 8)
	if bytesPerSample != 1 && bytesPerSample != 2 {
		return spectrogramPreview{}, nil
	}
	frameSize := bytesPerSample * int(channels)
	frames := len(data) / frameSize
	if frames < 2 {
		return spectrogramPreview{}, errors.New("WAV PCM data is too short")
	}

	mono := func(frame int) float64 {
		base := frame * frameSize
		var sum float64
		for ch := 0; ch < int(channels); ch++ {
			off := base + ch*bytesPerSample
			if bytesPerSample == 1 {
				sum += float64(int(data[off])-128) / 128.0
			} else {
				s := int16(binary.LittleEndian.Uint16(data[off : off+2]))
				sum += float64(s) / 32768.0
			}
		}
		return sum / float64(channels)
	}

	windowSize := spectrogramWindowFrames
	if frames < windowSize {
		windowSize = frames
	}
	out := make([]uint8, spectrogramTimeBins*spectrogramFrequencyBins)
	peak := 0.0
	magnitudes := make([]float64, len(out))
	for t := 0; t < spectrogramTimeBins; t++ {
		start := 0
		if frames > windowSize && spectrogramTimeBins > 1 {
			start = t * (frames - windowSize) / (spectrogramTimeBins - 1)
		}
		for b := 0; b < spectrogramFrequencyBins; b++ {
			k := 1 + b*(windowSize/2-1)/spectrogramFrequencyBins
			var re, im float64
			for n := 0; n < windowSize; n++ {
				weight := 1.0
				if windowSize > 1 {
					weight = 0.5 - 0.5*math.Cos(2*math.Pi*float64(n)/float64(windowSize-1))
				}
				sample := mono(start+n) * weight
				angle := 2 * math.Pi * float64(k*n) / float64(windowSize)
				re += sample * math.Cos(angle)
				im -= sample * math.Sin(angle)
			}
			m := math.Hypot(re, im)
			idx := t*spectrogramFrequencyBins + b
			magnitudes[idx] = m
			if m > peak {
				peak = m
			}
		}
	}
	if peak > 0 {
		for i, m := range magnitudes {
			normalized := math.Sqrt(m / peak)
			if normalized > 1 {
				normalized = 1
			}
			out[i] = uint8(math.Round(normalized * 255))
		}
	}
	return spectrogramPreview{Bins: out, TimeBins: spectrogramTimeBins, FrequencyBins: spectrogramFrequencyBins, MaxHz: int(sampleRate / 2)}, nil
}
