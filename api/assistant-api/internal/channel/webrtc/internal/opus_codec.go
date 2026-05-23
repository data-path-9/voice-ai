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

const (
	opusFrameSamples    = 960  // 20ms at 48kHz
	opusMaxFrameSamples = 5760 // 120ms at 48kHz — max Opus frame size per RFC 6716
)

// OpusCodec handles Opus audio encoding/decoding for WebRTC (48kHz mono)
type OpusCodec struct {
	mu sync.Mutex

	encoder *opus.Encoder
	decoder *opus.Decoder

	encodeSamples []int16
	encodeOutput  []byte
	decodeSamples []int16
	decodePCM     []byte
}

func NewOpusDecoder() (*OpusCodec, error) {
	dec, err := opus.NewDecoder(OpusSampleRate, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to create Opus decoder: %w", err)
	}
	return &OpusCodec{decoder: dec}, nil
}

// NewOpusCodec creates a new Opus codec optimized for voice
func NewOpusCodec() (*OpusCodec, error) {
	enc, err := opus.NewEncoder(OpusSampleRate, 1, opus.AppVoIP)
	if err != nil {
		return nil, fmt.Errorf("failed to create Opus encoder: %w", err)
	}

	enc.SetBitrate(32000)
	enc.SetComplexity(8)
	enc.SetInBandFEC(true)
	enc.SetPacketLossPerc(10)

	dec, err := opus.NewDecoder(OpusSampleRate, 1)
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

	c.mu.Lock()
	defer c.mu.Unlock()

	numSamples := len(pcm) / 2
	if cap(c.encodeSamples) < numSamples {
		c.encodeSamples = make([]int16, numSamples)
	}
	samples := c.encodeSamples[:numSamples]
	for i := 0; i < numSamples; i++ {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2 : i*2+2]))
	}
	if cap(c.encodeOutput) < 1000 {
		c.encodeOutput = make([]byte, 1000)
	}
	output := c.encodeOutput[:1000]
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

	c.mu.Lock()
	defer c.mu.Unlock()

	if cap(c.decodeSamples) < opusMaxFrameSamples {
		c.decodeSamples = make([]int16, opusMaxFrameSamples)
	}
	samples := c.decodeSamples[:opusMaxFrameSamples]
	n, err := c.decoder.Decode(encoded, samples)
	if err != nil {
		return nil, fmt.Errorf("Opus decode failed (payload=%d bytes): %w", len(encoded), err)
	}

	pcmLen := n * 2
	if cap(c.decodePCM) < pcmLen {
		c.decodePCM = make([]byte, pcmLen)
	}
	pcm := c.decodePCM[:pcmLen]
	for i := 0; i < n; i++ {
		binary.LittleEndian.PutUint16(pcm[i*2:i*2+2], uint16(samples[i]))
	}

	out := make([]byte, pcmLen)
	copy(out, pcm)
	return out, nil
}
