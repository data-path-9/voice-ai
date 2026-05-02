// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_twilio

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	internal_audio_resampler "github.com/rapidaai/api/assistant-api/internal/audio/resampler"
	internal_channel_input "github.com/rapidaai/api/assistant-api/internal/channel/input"
	internal_telephony_output "github.com/rapidaai/api/assistant-api/internal/channel/output"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

// Twilio audio constants (mulaw 8kHz)
const (
	// Standard chunk duration for telephony (20ms)
	ChunkDuration = 20 * time.Millisecond

	// Mulaw 8kHz: 8 bytes per ms (8-bit mono, 8000 samples/sec)
	MulawBytesPerMs = 8

	// Output chunk size: 20ms at 8kHz mulaw = 160 bytes
	OutputChunkSize = MulawBytesPerMs * 20

	// Input buffer threshold: 60ms at 16kHz linear16 = 1920 bytes
	InputBufferThreshold = 32 * 60

	// Mulaw silence value (0x7F or 0xFF represents silence)
	MulawSilence = 0xFF
)

// AudioChunk represents a processed audio chunk ready for streaming
type AudioChunk struct {
	Data     []byte
	Duration time.Duration
}

// AudioProcessor handles audio conversion for Twilio (mulaw 8kHz <-> linear16 16kHz)
type AudioProcessor struct {
	logger commons.Logger

	// Resampler for format and sample rate conversion
	resampler internal_type.AudioResampler

	// Audio configs
	twilioConfig     *protos.AudioConfig // mulaw 8kHz for Twilio
	downstreamConfig *protos.AudioConfig // linear16 16kHz for STT/TTS

	// Input buffer for accumulating incoming audio (converted to 16kHz)
	inputBuffer internal_channel_input.InputBuffer

	// Output buffer for audio to be sent to Twilio (converted to mulaw 8kHz)
	outputBuffer internal_telephony_output.FrameBuffer

	// Callback for processed input audio (to send to downstream)
	onInputAudio func(audio []byte)

	// Callback for sending audio chunk to Twilio
	onOutputChunk func(chunk *AudioChunk) error

	// Pre-created silence chunk
	silenceChunk *AudioChunk

	ambientMixer internal_ambient.Mixer
	adapter      internal_telephony_output.AudioAdapter

	outputSenderRunning atomic.Bool
	outputHealth        *internal_telephony_output.HealthStats
}

// NewAudioProcessor creates a new Twilio audio processor
func NewAudioProcessor(logger commons.Logger) (*AudioProcessor, error) {
	resampler, err := internal_audio_resampler.GetResampler(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create resampler: %w", err)
	}

	p := &AudioProcessor{
		logger:           logger,
		resampler:        resampler,
		twilioConfig:     internal_audio.NewMulaw8khzMonoAudioConfig(),
		downstreamConfig: internal_audio.NewLinear16khzMonoAudioConfig(),
		inputBuffer:      internal_channel_input.NewBytesInputBuffer(InputBufferThreshold * 2),
		outputBuffer:     internal_telephony_output.NewBytesFrameBuffer(OutputChunkSize * 8),
		outputHealth:     internal_telephony_output.NewHealthStats(),
	}
	p.adapter = newAudioAdapter(p.resampler, p.downstreamConfig, p.twilioConfig, OutputChunkSize, MulawSilence)

	// Pre-create silence chunk
	p.silenceChunk = p.createSilenceChunk()
	ambientMixer, err := internal_ambient.NewLoopMixer(internal_ambient.MixerSpec{
		Resampler:         p.resampler,
		TargetAudioConfig: internal_audio.NewLinear8khzMonoAudioConfig(),
		FrameBytes:        OutputChunkSize * 2,
	})
	if err == nil {
		p.ambientMixer = ambientMixer
	}

	return p, nil
}

func (p *AudioProcessor) ConfigureAmbient(cfg internal_ambient.Config) error {
	if p.ambientMixer == nil {
		return nil
	}
	return p.ambientMixer.Configure(cfg)
}

// SetInputAudioCallback sets the callback for processed input audio
func (p *AudioProcessor) SetInputAudioCallback(callback func(audio []byte)) {
	p.onInputAudio = callback
}

// SetOutputChunkCallback sets the callback for sending audio chunks to Twilio
func (p *AudioProcessor) SetOutputChunkCallback(callback func(chunk *AudioChunk) error) {
	p.onOutputChunk = callback
}

// ============================================================================
// Input Audio Processing (from Twilio mulaw 8kHz -> downstream linear16 16kHz)
// ============================================================================

// ProcessInputAudio converts incoming mulaw 8kHz audio to linear16 16kHz
func (p *AudioProcessor) ProcessInputAudio(audio []byte) error {
	if len(audio) == 0 {
		return nil
	}

	// Convert from mulaw 8kHz to linear16 16kHz
	converted, err := p.resampler.Resample(audio, p.twilioConfig, p.downstreamConfig)
	if err != nil {
		return fmt.Errorf("audio conversion to 16kHz linear16 failed: %w", err)
	}

	// Buffer and send when threshold reached
	p.bufferAndSendInput(converted)
	return nil
}

// bufferAndSendInput buffers input audio and sends when threshold is reached
func (p *AudioProcessor) bufferAndSendInput(audio []byte) {
	p.inputBuffer.Write(audio)
	audioData, ok := p.inputBuffer.DrainIfReady(InputBufferThreshold)
	if !ok {
		return
	}

	if p.onInputAudio != nil {
		p.onInputAudio(audioData)
	}
}

// ============================================================================
// Output Audio Processing (from downstream linear16 16kHz -> Twilio mulaw 8kHz)
// ============================================================================

// ProcessOutputAudio converts outgoing linear16 16kHz audio to mulaw 8kHz
func (p *AudioProcessor) ProcessOutputAudio(audio []byte) error {
	if len(audio) == 0 {
		return nil
	}

	// Convert from linear16 16kHz to mulaw 8kHz
	converted, err := p.adapter.ConvertOutput(audio)
	if err != nil {
		return fmt.Errorf("audio conversion to mulaw 8kHz failed: %w", err)
	}

	p.outputBuffer.Write(converted)

	return nil
}

// Complete flushes buffered trailing bytes by padding to a full frame.
func (p *AudioProcessor) Complete() {
	p.outputBuffer.Complete(p.adapter.FrameSize(), p.adapter.SilenceByte())
}

// GetNextChunk retrieves the next audio chunk from the output buffer
func (p *AudioProcessor) GetNextChunk() *AudioChunk {
	chunk, ok := p.outputBuffer.Next(p.adapter.FrameSize())
	if !ok {
		return nil
	}

	return &AudioChunk{
		Data:     chunk,
		Duration: ChunkDuration,
	}
}

// createSilenceChunk creates a mulaw silence chunk
func (p *AudioProcessor) createSilenceChunk() *AudioChunk {
	chunk := make([]byte, p.adapter.FrameSize())
	for i := range chunk {
		chunk[i] = p.adapter.SilenceByte()
	}
	return &AudioChunk{
		Data:     chunk,
		Duration: ChunkDuration,
	}
}

// RunOutputSender continuously sends audio chunks at consistent 20ms intervals
func (p *AudioProcessor) RunOutputSender(ctx context.Context) {
	if p.onOutputChunk == nil {
		p.logger.Error("RunOutputSender called without output callback set")
		return
	}
	if !p.outputSenderRunning.CompareAndSwap(false, true) {
		return
	}
	defer p.outputSenderRunning.Store(false)
	(&internal_telephony_output.Pacer{
		Logger:        p.logger,
		FrameDuration: ChunkDuration,
		Provider:      p,
		Consumer:      p,
		Health:        p.outputHealth,
	}).Run(ctx)
}

func (p *AudioProcessor) OutputHealthSnapshot() internal_telephony_output.HealthSnapshot {
	if p.outputHealth == nil {
		return internal_telephony_output.HealthSnapshot{}
	}
	return p.outputHealth.Snapshot()
}

func (p *AudioProcessor) applyAmbient(chunk []byte) []byte {
	return p.adapter.MixAmbient(chunk, p.ambientMixer)
}

func (p *AudioProcessor) NextFrame() []byte {
	chunk := p.GetNextChunk()
	if chunk == nil {
		return nil
	}
	return p.applyAmbient(chunk.Data)
}

func (p *AudioProcessor) IdleFrame() []byte {
	frame := p.applyAmbient(nil)
	if len(frame) > 0 {
		return frame
	}
	return append([]byte(nil), p.silenceChunk.Data...)
}

func (p *AudioProcessor) ConsumeFrame(frame []byte) error {
	return p.onOutputChunk(&AudioChunk{
		Data:     frame,
		Duration: ChunkDuration,
	})
}

// ClearOutputBuffer clears the output audio buffer
func (p *AudioProcessor) ClearOutputBuffer() {
	p.outputBuffer.Clear()
}
