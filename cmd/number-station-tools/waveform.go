package main

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"os"
)

const waveformBins = 128

func buildWaveformPreview(path, format string) ([]uint8, error) {
	switch format {
	case "wav":
		return wavWaveform(path, waveformBins)
	case "flac":
		return nil, nil
	default:
		return nil, errors.New("unsupported audio format")
	}
}

func wavWaveform(path string, bins int) ([]uint8, error) {
	if bins < 1 {
		return nil, errors.New("waveform bins must be positive")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	header := make([]byte, 12)
	if _, err := io.ReadFull(f, header); err != nil {
		return nil, errors.New("invalid WAV header")
	}
	if string(header[:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		return nil, errors.New("invalid WAV signature")
	}

	var audioFormat, channels, bits uint16
	var data []byte
	for {
		chunkHeader := make([]byte, 8)
		if _, err := io.ReadFull(f, chunkHeader); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return nil, err
		}
		size := binary.LittleEndian.Uint32(chunkHeader[4:8])
		chunk := make([]byte, size)
		if _, err := io.ReadFull(f, chunk); err != nil {
			return nil, errors.New("truncated WAV chunk")
		}
		if size%2 == 1 {
			_, _ = f.Seek(1, io.SeekCurrent)
		}
		switch string(chunkHeader[:4]) {
		case "fmt ":
			if len(chunk) < 16 {
				return nil, errors.New("invalid WAV fmt chunk")
			}
			audioFormat = binary.LittleEndian.Uint16(chunk[0:2])
			channels = binary.LittleEndian.Uint16(chunk[2:4])
			bits = binary.LittleEndian.Uint16(chunk[14:16])
		case "data":
			data = chunk
		}
	}
	if len(data) == 0 || channels == 0 {
		return nil, errors.New("WAV data chunk missing")
	}
	if audioFormat != 1 {
		return nil, nil
	}
	bytesPerSample := int(bits / 8)
	if bytesPerSample != 1 && bytesPerSample != 2 {
		return nil, nil
	}
	frameSize := bytesPerSample * int(channels)
	if frameSize <= 0 || len(data) < frameSize {
		return nil, errors.New("WAV PCM data is empty")
	}
	frames := len(data) / frameSize
	out := make([]uint8, bins)
	for frame := 0; frame < frames; frame++ {
		bin := frame * bins / frames
		if bin >= bins {
			bin = bins - 1
		}
		peak := float64(out[bin]) / 255.0
		base := frame * frameSize
		for ch := 0; ch < int(channels); ch++ {
			off := base + ch*bytesPerSample
			var amp float64
			if bytesPerSample == 1 {
				amp = math.Abs(float64(int(data[off])-128)) / 128.0
			} else {
				s := int16(binary.LittleEndian.Uint16(data[off : off+2]))
				amp = math.Abs(float64(s)) / 32768.0
			}
			if amp > peak {
				peak = amp
			}
		}
		if peak > 1 {
			peak = 1
		}
		out[bin] = uint8(math.Round(peak * 255))
	}
	return out, nil
}
