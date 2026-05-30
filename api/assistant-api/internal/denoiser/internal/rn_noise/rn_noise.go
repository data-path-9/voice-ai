// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_denoiser_rnnoise

/*
#cgo CFLAGS: -I.
#cgo LDFLAGS: -L./models -lrnnoise
#include <rnnoise.h>
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"sync"
	"unsafe"
)

var frameSize int

const pcm16AmplitudeScale = float32(32768.0)

func init() {
	frameSize = int(C.rnnoise_get_frame_size())
}

// RNNoise wraps the RNNoise C library with thread-safe processing
type RNNoise struct {
	mu           sync.Mutex
	denoiseState *C.DenoiseState
	frameCount   int
}

// NewRNNoise creates a new RNNoise instance
func NewRNNoise() (*RNNoise, error) {
	state := C.rnnoise_create(nil)
	if state == nil {
		return nil, fmt.Errorf("failed to create rnnoise state")
	}

	return &RNNoise{
		denoiseState: state,
		frameCount:   0,
	}, nil
}

// SuppressNoise processes a single frame of audio and returns confidence score and cleaned audio
// Input must be exactly frameSize samples (typically 480 at 48kHz)
// Audio must be at 48kHz sample rate for proper noise suppression
func (st *RNNoise) SuppressNoise(input []float32) (float64, []float32, error) {
	if st.denoiseState == nil {
		return 0, nil, fmt.Errorf("rnnoise state is not initialized")
	}

	if len(input) != frameSize {
		return 0, nil, fmt.Errorf("input must be exactly %d samples, got %d", frameSize, len(input))
	}

	pcmAmplitudeInputFrame := make([]float32, frameSize)
	pcmAmplitudeOutputFrame := make([]float32, frameSize)
	// Match upstream rnnoise_demo.c: it copies int16 PCM samples directly into float before rnnoise_process_frame.
	// Reference: https://github.com/xiph/rnnoise/blob/main/examples/rnnoise_demo.c
	for i, sample := range input {
		pcmAmplitudeInputFrame[i] = sample * pcm16AmplitudeScale
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	inputPtr := (*C.float)(unsafe.Pointer(&pcmAmplitudeInputFrame[0]))
	outputPtr := (*C.float)(unsafe.Pointer(&pcmAmplitudeOutputFrame[0]))

	speechConfidence := C.rnnoise_process_frame(st.denoiseState, outputPtr, inputPtr)

	st.frameCount++

	output := make([]float32, frameSize)
	copyPCMAmplitudeSamplesToNormalized(output, pcmAmplitudeOutputFrame)

	return float64(speechConfidence), output, nil
}

// ProcessAudio processes multiple frames, preserving the exact input length
// while averaging the per-frame speech confidence returned by RNNoise.
func (st *RNNoise) ProcessAudio(input []float32) (float64, []float32, error) {
	if st.denoiseState == nil {
		return 0, nil, fmt.Errorf("rnnoise state is not initialized")
	}

	if len(input) == 0 {
		return 0, nil, fmt.Errorf("input audio is empty")
	}

	frameCount := (len(input) + frameSize - 1) / frameSize
	cleanedAudio := make([]float32, len(input))
	var totalConfidence float64

	st.mu.Lock()
	defer st.mu.Unlock()

	pcmAmplitudeInputFrame := make([]float32, frameSize)
	pcmAmplitudeOutputFrame := make([]float32, frameSize)

	for i := 0; i < len(input); i += frameSize {
		end := i + frameSize
		if end > len(input) {
			end = len(input)
		}

		chunk := input[i:end]
		outputChunk := cleanedAudio[i:end]

		clear(pcmAmplitudeInputFrame)
		clear(pcmAmplitudeOutputFrame)
		for j, sample := range chunk {
			pcmAmplitudeInputFrame[j] = sample * pcm16AmplitudeScale
		}

		inputPtr := (*C.float)(unsafe.Pointer(&pcmAmplitudeInputFrame[0]))
		outputPtr := (*C.float)(unsafe.Pointer(&pcmAmplitudeOutputFrame[0]))

		speechConfidence := C.rnnoise_process_frame(st.denoiseState, outputPtr, inputPtr)
		totalConfidence += float64(speechConfidence)

		copyPCMAmplitudeSamplesToNormalized(outputChunk, pcmAmplitudeOutputFrame[:len(chunk)])

		st.frameCount++
	}

	return totalConfidence / float64(frameCount), cleanedAudio, nil
}

func copyPCMAmplitudeSamplesToNormalized(dst, src []float32) {
	for i, sample := range src {
		normalized := sample / pcm16AmplitudeScale
		switch {
		case normalized > 1:
			dst[i] = 1
		case normalized < -1:
			dst[i] = -1
		default:
			dst[i] = normalized
		}
	}
}

// Close cleans up resources
func (st *RNNoise) Close() error {
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.denoiseState == nil {
		return fmt.Errorf("double-free attempt")
	}

	C.rnnoise_destroy(st.denoiseState)
	st.denoiseState = nil

	return nil
}
