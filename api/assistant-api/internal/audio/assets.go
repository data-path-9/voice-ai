// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_audio

import (
	"embed"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/rapidaai/protos"
)

//go:embed assets/ambient/* assets/ringtone/*
var embeddedAudioAssets embed.FS

// LoadAmbientPCM16LE loads an ambient loop by enum name from shared assets.
// Supported formats:
//   - assets/ambient/<name>.wav (PCM16 LE, mono)
//   - assets/ambient/<name>.pcm (raw PCM16 LE mono)
func LoadAmbientPCM16LE(name string) ([]byte, *protos.AudioConfig, error) {
	n := strings.TrimSpace(strings.ToLower(name))
	if n == "" {
		return nil, nil, fmt.Errorf("ambient name is empty")
	}

	if wavBytes, err := embeddedAudioAssets.ReadFile("assets/ambient/" + n + ".wav"); err == nil {
		return decodePCM16WAV(wavBytes)
	}
	if pcmBytes, err := embeddedAudioAssets.ReadFile("assets/ambient/" + n + ".pcm"); err == nil {
		if len(pcmBytes) < 2 {
			return nil, nil, fmt.Errorf("ambient pcm too short: %s", n)
		}
		if len(pcmBytes)%2 != 0 {
			pcmBytes = pcmBytes[:len(pcmBytes)-1]
		}
		// Raw PCM defaults to 48kHz mono LINEAR16 for WebRTC-friendly assets.
		return pcmBytes, NewLinear48khzMonoAudioConfig(), nil
	}
	if _, err := embeddedAudioAssets.ReadFile("assets/ambient/" + n + ".mp3"); err == nil {
		return nil, nil, fmt.Errorf("ambient mp3 is not supported in server mixer path: %s", n)
	}
	return nil, nil, fmt.Errorf("ambient asset not found: %s", n)
}

func decodePCM16WAV(wav []byte) ([]byte, *protos.AudioConfig, error) {
	if len(wav) < 44 {
		return nil, nil, fmt.Errorf("invalid wav: too short")
	}
	if string(wav[0:4]) != "RIFF" || string(wav[8:12]) != "WAVE" {
		return nil, nil, fmt.Errorf("invalid wav header")
	}

	off := 12
	var data []byte
	var audioFormat uint16
	var channels uint16
	var sampleRate uint32
	var bitsPerSample uint16
	foundFmt := false
	foundData := false

	for off+8 <= len(wav) {
		chunkID := string(wav[off : off+4])
		chunkSize := int(binary.LittleEndian.Uint32(wav[off+4 : off+8]))
		off += 8
		if off+chunkSize > len(wav) {
			break
		}

		switch chunkID {
		case "fmt ":
			if chunkSize < 16 {
				return nil, nil, fmt.Errorf("invalid wav fmt chunk")
			}
			audioFormat = binary.LittleEndian.Uint16(wav[off : off+2])
			channels = binary.LittleEndian.Uint16(wav[off+2 : off+4])
			sampleRate = binary.LittleEndian.Uint32(wav[off+4 : off+8])
			bitsPerSample = binary.LittleEndian.Uint16(wav[off+14 : off+16])
			foundFmt = true
		case "data":
			data = wav[off : off+chunkSize]
			foundData = true
		}

		off += chunkSize
		if off%2 == 1 {
			off++
		}
	}

	if !foundFmt || !foundData {
		return nil, nil, fmt.Errorf("invalid wav: missing fmt/data")
	}
	if audioFormat != 1 {
		return nil, nil, fmt.Errorf("unsupported wav format: %d", audioFormat)
	}
	if channels == 0 {
		return nil, nil, fmt.Errorf("invalid wav channels: %d", channels)
	}
	if bitsPerSample != 16 {
		return nil, nil, fmt.Errorf("unsupported wav bits per sample: %d", bitsPerSample)
	}
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	return data, &protos.AudioConfig{
		SampleRate:  sampleRate,
		AudioFormat: protos.AudioConfig_LINEAR16,
		Channels:    uint32(channels),
	}, nil
}
