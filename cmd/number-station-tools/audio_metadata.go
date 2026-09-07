package main

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
)

type audioMetadata struct {
	DurationMS   int64
	SampleRateHz int
	Channels     int
}

func extractAudioMetadata(path, format string) (audioMetadata, error) {
	switch format {
	case "wav":
		return extractWAVMetadata(path)
	case "flac":
		return extractFLACMetadata(path)
	default:
		return audioMetadata{}, errors.New("unsupported audio format")
	}
}

func extractWAVMetadata(path string) (audioMetadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return audioMetadata{}, err
	}
	defer f.Close()

	header := make([]byte, 12)
	if _, err := io.ReadFull(f, header); err != nil {
		return audioMetadata{}, errors.New("invalid WAV header")
	}
	if string(header[:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		return audioMetadata{}, errors.New("invalid WAV signature")
	}

	var sampleRate uint32
	var byteRate uint32
	var channels uint16
	var dataBytes uint32
	for {
		chunkHeader := make([]byte, 8)
		if _, err := io.ReadFull(f, chunkHeader); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return audioMetadata{}, err
		}
		chunkID := string(chunkHeader[:4])
		chunkSize := binary.LittleEndian.Uint32(chunkHeader[4:8])
		switch chunkID {
		case "fmt ":
			if chunkSize < 16 {
				return audioMetadata{}, errors.New("invalid WAV fmt chunk")
			}
			buf := make([]byte, chunkSize)
			if _, err := io.ReadFull(f, buf); err != nil {
				return audioMetadata{}, errors.New("truncated WAV fmt chunk")
			}
			channels = binary.LittleEndian.Uint16(buf[2:4])
			sampleRate = binary.LittleEndian.Uint32(buf[4:8])
			byteRate = binary.LittleEndian.Uint32(buf[8:12])
		case "data":
			dataBytes = chunkSize
			if _, err := f.Seek(int64(chunkSize), io.SeekCurrent); err != nil {
				return audioMetadata{}, err
			}
		default:
			if _, err := f.Seek(int64(chunkSize), io.SeekCurrent); err != nil {
				return audioMetadata{}, err
			}
		}
		if chunkSize%2 == 1 {
			if _, err := f.Seek(1, io.SeekCurrent); err != nil {
				return audioMetadata{}, err
			}
		}
		if sampleRate > 0 && byteRate > 0 && channels > 0 && dataBytes > 0 {
			break
		}
	}
	if sampleRate == 0 || byteRate == 0 || channels == 0 || dataBytes == 0 {
		return audioMetadata{}, errors.New("WAV metadata is incomplete")
	}
	return audioMetadata{
		DurationMS:   int64(dataBytes) * 1000 / int64(byteRate),
		SampleRateHz: int(sampleRate),
		Channels:     int(channels),
	}, nil
}

func extractFLACMetadata(path string) (audioMetadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return audioMetadata{}, err
	}
	defer f.Close()

	signature := make([]byte, 4)
	if _, err := io.ReadFull(f, signature); err != nil || string(signature) != "fLaC" {
		return audioMetadata{}, errors.New("invalid FLAC signature")
	}
	for {
		header := make([]byte, 4)
		if _, err := io.ReadFull(f, header); err != nil {
			return audioMetadata{}, errors.New("missing FLAC STREAMINFO")
		}
		last := header[0]&0x80 != 0
		blockType := header[0] & 0x7f
		length := int(header[1])<<16 | int(header[2])<<8 | int(header[3])
		if blockType == 0 {
			if length != 34 {
				return audioMetadata{}, errors.New("invalid FLAC STREAMINFO length")
			}
			buf := make([]byte, 34)
			if _, err := io.ReadFull(f, buf); err != nil {
				return audioMetadata{}, errors.New("truncated FLAC STREAMINFO")
			}
			packed := binary.BigEndian.Uint64(buf[10:18])
			sampleRate := int((packed >> 44) & 0xfffff)
			channels := int((packed>>41)&0x7) + 1
			totalSamples := packed & ((uint64(1) << 36) - 1)
			if sampleRate <= 0 || channels <= 0 || totalSamples == 0 {
				return audioMetadata{}, errors.New("FLAC STREAMINFO is incomplete")
			}
			return audioMetadata{
				DurationMS:   int64(totalSamples) * 1000 / int64(sampleRate),
				SampleRateHz: sampleRate,
				Channels:     channels,
			}, nil
		}
		if _, err := f.Seek(int64(length), io.SeekCurrent); err != nil {
			return audioMetadata{}, err
		}
		if last {
			break
		}
	}
	return audioMetadata{}, errors.New("missing FLAC STREAMINFO")
}
