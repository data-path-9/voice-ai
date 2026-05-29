// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package webrtc_internal

import (
	"encoding/binary"
	"fmt"
	"sync"

	"gopkg.in/hraban/opus.v2"
)

// OpusCodec handles Opus audio encoding/decoding for WebRTC (48kHz mono)
type OpusCodec struct {
	codecLock sync.Mutex

	encoder *opus.Encoder
	decoder *opus.Decoder

	encodeSamples []int16
	encodeOutput  []byte
	decodeSamples []int16
	decodePCM     []byte
}

func NewOpusDecoder() (*OpusCodec, error) {
	dec, err := opus.NewDecoder(OpusSampleRate, OpusVoiceChannels)
	if err != nil {
		return nil, fmt.Errorf("failed to create Opus decoder: %w", err)
	}
	return &OpusCodec{decoder: dec}, nil
}

// NewOpusCodec creates a new Opus codec optimized for voice
func NewOpusCodec() (*OpusCodec, error) {
	enc, err := opus.NewEncoder(OpusSampleRate, OpusVoiceChannels, opus.AppVoIP)
	if err != nil {
		return nil, fmt.Errorf("failed to create Opus encoder: %w", err)
	}

	enc.SetBitrate(OpusVoiceBitrate)
	enc.SetComplexity(OpusVoiceComplexity)
	enc.SetInBandFEC(true)
	enc.SetPacketLossPerc(OpusExpectedPacketLossPercent)

	dec, err := opus.NewDecoder(OpusSampleRate, OpusVoiceChannels)
	if err != nil {
		return nil, fmt.Errorf("failed to create Opus decoder: %w", err)
	}

	return &OpusCodec{encoder: enc, decoder: dec}, nil
}

// Encode encodes PCM16 bytes (48kHz mono, little-endian) to Opus
func (c *OpusCodec) Encode(pcm []byte) ([]byte, error) {
	if c == nil || c.encoder == nil {
		return nil, fmt.Errorf("Opus encoder is not initialized")
	}
	if len(pcm) == 0 {
		return nil, nil
	}

	c.codecLock.Lock()
	defer c.codecLock.Unlock()

	numSamples := len(pcm) / OpusPCMBytesPerSample
	if cap(c.encodeSamples) < numSamples {
		c.encodeSamples = make([]int16, numSamples)
	}
	samples := c.encodeSamples[:numSamples]
	for i := 0; i < numSamples; i++ {
		sampleOffset := i * OpusPCMBytesPerSample
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[sampleOffset : sampleOffset+OpusPCMBytesPerSample]))
	}
	if cap(c.encodeOutput) < OpusEncoderOutputMaxBytes {
		c.encodeOutput = make([]byte, OpusEncoderOutputMaxBytes)
	}
	output := c.encodeOutput[:OpusEncoderOutputMaxBytes]
	n, err := c.encoder.Encode(samples, output)
	if err != nil {
		return nil, fmt.Errorf("Opus encode failed: %w", err)
	}
	encoded := make([]byte, n)
	copy(encoded, output[:n])
	return encoded, nil
}

// Decode decodes Opus to PCM16 bytes (48kHz mono, little-endian).
// The decode buffer is sized for the maximum Opus frame (120ms) so that
// any valid frame duration (2.5ms, 5ms, 10ms, 20ms, 40ms, 60ms, or 120ms
// via CELT) can be decoded without "buffer too small" errors.
func (c *OpusCodec) Decode(encoded []byte) ([]byte, error) {
	if c == nil || c.decoder == nil {
		return nil, fmt.Errorf("Opus decoder is not initialized")
	}
	if len(encoded) == 0 {
		return nil, nil
	}

	c.codecLock.Lock()
	defer c.codecLock.Unlock()

	if cap(c.decodeSamples) < OpusMaxFrameSamples {
		c.decodeSamples = make([]int16, OpusMaxFrameSamples)
	}
	samples := c.decodeSamples[:OpusMaxFrameSamples]
	n, err := c.decoder.Decode(encoded, samples)
	if err != nil {
		return nil, fmt.Errorf("Opus decode failed (payload=%d bytes): %w", len(encoded), err)
	}

	pcmLen := n * OpusPCMBytesPerSample
	if cap(c.decodePCM) < pcmLen {
		c.decodePCM = make([]byte, pcmLen)
	}
	pcm := c.decodePCM[:pcmLen]
	for i := 0; i < n; i++ {
		sampleOffset := i * OpusPCMBytesPerSample
		binary.LittleEndian.PutUint16(pcm[sampleOffset:sampleOffset+OpusPCMBytesPerSample], uint16(samples[i]))
	}

	out := make([]byte, pcmLen)
	copy(out, pcm)
	return out, nil
}
