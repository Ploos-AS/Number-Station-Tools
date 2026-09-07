package main

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"os"
)

const frequencyBins = 64
const frequencyWindowFrames = 256
const frequencyWindows = 4

type frequencyPreview struct {
	Bins  []uint8
	MaxHz int
}

func buildFrequencyPreview(path, format string) (frequencyPreview, error) {
	switch format {
	case "wav":
		return wavFrequencyPreview(path)
	case "flac":
		return frequencyPreview{}, nil
	default:
		return frequencyPreview{}, errors.New("unsupported audio format")
	}
}

func wavFrequencyPreview(path string) (frequencyPreview, error) {
	f, err := os.Open(path)
	if err != nil {
		return frequencyPreview{}, err
	}
	defer f.Close()

	header := make([]byte, 12)
	if _, err := io.ReadFull(f, header); err != nil {
		return frequencyPreview{}, errors.New("invalid WAV header")
	}
	if string(header[:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		return frequencyPreview{}, errors.New("invalid WAV signature")
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
			return frequencyPreview{}, err
		}
		size := binary.LittleEndian.Uint32(chunkHeader[4:8])
		chunk := make([]byte, size)
		if _, err := io.ReadFull(f, chunk); err != nil {
			return frequencyPreview{}, errors.New("truncated WAV chunk")
		}
		if size%2 == 1 {
			_, _ = f.Seek(1, io.SeekCurrent)
		}
		switch string(chunkHeader[:4]) {
		case "fmt ":
			if len(chunk) < 16 {
				return frequencyPreview{}, errors.New("invalid WAV fmt chunk")
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
		return frequencyPreview{}, errors.New("WAV PCM metadata or data missing")
	}
	if audioFormat != 1 {
		return frequencyPreview{}, nil
	}
	bytesPerSample := int(bits / 8)
	if bytesPerSample != 1 && bytesPerSample != 2 {
		return frequencyPreview{}, nil
	}
	frameSize := bytesPerSample * int(channels)
	frames := len(data) / frameSize
	if frames < 2 {
		return frequencyPreview{}, errors.New("WAV PCM data is too short")
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

	windowSize := frequencyWindowFrames
	if frames < windowSize {
		windowSize = frames
	}
	spectrum := make([]float64, frequencyBins)
	usedWindows := 0
	for w := 0; w < frequencyWindows; w++ {
		start := 0
		if frames > windowSize {
			start = w * (frames - windowSize) / (frequencyWindows - 1)
		}
		for bin := 0; bin < frequencyBins; bin++ {
			k := 1 + bin*(windowSize/2-1)/frequencyBins
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
			magnitude := math.Hypot(re, im)
			if magnitude > spectrum[bin] {
				spectrum[bin] = magnitude
			}
		}
		usedWindows++
		if frames <= windowSize {
			break
		}
	}
	_ = usedWindows

	maxMagnitude := 0.0
	for _, magnitude := range spectrum {
		if magnitude > maxMagnitude {
			maxMagnitude = magnitude
		}
	}
	out := make([]uint8, frequencyBins)
	if maxMagnitude > 0 {
		for i, magnitude := range spectrum {
			normalized := math.Sqrt(magnitude / maxMagnitude)
			if normalized > 1 {
				normalized = 1
			}
			out[i] = uint8(math.Round(normalized * 255))
		}
	}
	return frequencyPreview{Bins: out, MaxHz: int(sampleRate / 2)}, nil
}
