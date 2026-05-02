// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_asterisk

import (
	"context"
	"fmt"
	"sync"
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

const (
	chunkDuration        = 20 * time.Millisecond
	defaultFrameSize     = 160
	inputBufferThreshold = 32 * 60
)

// AudioChunk represents a processed audio chunk ready for streaming.
type AudioChunk struct {
	Data     []byte
	Duration time.Duration
}

// AudioProcessorConfig parameterizes the shared AudioProcessor for different
// Asterisk transports (AudioSocket SLIN vs WebSocket mu-law).
type AudioProcessorConfig struct {
	AsteriskConfig   *protos.AudioConfig // provider-side format
	DownstreamConfig *protos.AudioConfig // internal format (linear16 16kHz)
	SilenceByte      byte                // 0xFF for mu-law, 0x00 for SLIN
	FrameSize        int                 // optimal frame size (bytes per 20ms)
	Ambient          *internal_ambient.Config
}

// AudioProcessor handles audio conversion between Asterisk and downstream
// formats. It is parameterized by AudioProcessorConfig so both the AudioSocket
// (SLIN 8kHz) and WebSocket (mu-law 8kHz) transports share a single
// implementation.
type AudioProcessor struct {
	logger           commons.Logger
	resampler        internal_type.AudioResampler
	asteriskConfig   *protos.AudioConfig
	downstreamConfig *protos.AudioConfig
	silenceByte      byte
	optimalFrameSize int
	stateMu          sync.RWMutex

	inputBuffer  internal_channel_input.InputBuffer
	outputBuffer internal_telephony_output.FrameBuffer

	onInputAudio  func(audio []byte)
	onOutputChunk func(chunk *AudioChunk) error
	silenceChunk  *AudioChunk
	ambientMixer  internal_ambient.Mixer
	adapter       internal_telephony_output.AudioAdapter

	xoffActive bool
	xoffMu     sync.Mutex

	outputSenderRunning atomic.Bool
	outputHealth        *internal_telephony_output.HealthStats
}

func NewAudioProcessor(logger commons.Logger, cfg AudioProcessorConfig) (*AudioProcessor, error) {
	resampler, err := internal_audio_resampler.GetResampler(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create resampler: %w", err)
	}

	frameSize := cfg.FrameSize
	if frameSize <= 0 {
		frameSize = defaultFrameSize
	}

	p := &AudioProcessor{
		logger:           logger,
		resampler:        resampler,
		asteriskConfig:   cfg.AsteriskConfig,
		downstreamConfig: cfg.DownstreamConfig,
		silenceByte:      cfg.SilenceByte,
		optimalFrameSize: frameSize,
		inputBuffer:      internal_channel_input.NewBytesInputBuffer(inputBufferThreshold * 2),
		outputBuffer:     internal_telephony_output.NewBytesFrameBuffer(frameSize * 8),
		outputHealth:     internal_telephony_output.NewHealthStats(),
	}
	p.adapter = newAudioAdapter(p.resampler, p.downstreamConfig, p.asteriskConfig, frameSize, p.silenceByte)
	p.silenceChunk = p.createSilenceChunk(frameSize, p.adapter.SilenceByte())
	if cfg.AsteriskConfig != nil {
		switch cfg.AsteriskConfig.GetAudioFormat() {
		case protos.AudioConfig_MuLaw8:
			ambientMixer, err := internal_ambient.NewLoopMixer(internal_ambient.MixerSpec{
				Resampler:         p.resampler,
				TargetAudioConfig: internal_audio.NewLinear8khzMonoAudioConfig(),
				FrameBytes:        frameSize * 2,
			})
			if err == nil {
				p.ambientMixer = ambientMixer
			}
		case protos.AudioConfig_LINEAR16:
			ambientMixer, err := internal_ambient.NewLoopMixer(internal_ambient.MixerSpec{
				Resampler:         p.resampler,
				TargetAudioConfig: cfg.AsteriskConfig,
				FrameBytes:        frameSize,
			})
			if err == nil {
				p.ambientMixer = ambientMixer
			}
		}
	}
	if cfg.Ambient != nil {
		_ = p.ConfigureAmbient(*cfg.Ambient)
	}
	return p, nil
}

func (p *AudioProcessor) SetInputAudioCallback(callback func(audio []byte)) {
	p.onInputAudio = callback
}

func (p *AudioProcessor) SetOutputChunkCallback(callback func(chunk *AudioChunk) error) {
	p.onOutputChunk = callback
}

func (p *AudioProcessor) ConfigureAmbient(cfg internal_ambient.Config) error {
	if p.ambientMixer == nil {
		return nil
	}
	return p.ambientMixer.Configure(cfg)
}

func (p *AudioProcessor) SetOptimalFrameSize(size int) {
	if size > 0 {
		p.stateMu.Lock()
		p.optimalFrameSize = size
		p.adapter = newAudioAdapter(p.resampler, p.downstreamConfig, p.asteriskConfig, size, p.silenceByte)
		p.silenceChunk = p.createSilenceChunk(size, p.adapter.SilenceByte())
		p.stateMu.Unlock()
	}
}

func (p *AudioProcessor) GetOptimalFrameSize() int {
	p.stateMu.RLock()
	defer p.stateMu.RUnlock()
	return p.optimalFrameSize
}

func (p *AudioProcessor) GetDownstreamConfig() *protos.AudioConfig {
	return p.downstreamConfig
}

func (p *AudioProcessor) ProcessInputAudio(audio []byte) error {
	if len(audio) == 0 {
		return nil
	}

	converted, err := p.resampler.Resample(audio, p.asteriskConfig, p.downstreamConfig)
	if err != nil {
		return fmt.Errorf("audio conversion from asterisk format to downstream failed: %w", err)
	}

	p.bufferAndSendInput(converted)
	return nil
}

func (p *AudioProcessor) bufferAndSendInput(audio []byte) {
	p.inputBuffer.Write(audio)
	audioData, ok := p.inputBuffer.DrainIfReady(inputBufferThreshold)
	if !ok {
		return
	}

	if p.onInputAudio != nil {
		p.onInputAudio(audioData)
	}
}

func (p *AudioProcessor) ClearInputBuffer() {
	p.inputBuffer.Clear()
}

func (p *AudioProcessor) ProcessOutputAudio(audio []byte) error {
	if len(audio) == 0 {
		return nil
	}

	adapter := p.getAdapter()
	converted, err := adapter.ConvertOutput(audio)
	if err != nil {
		return fmt.Errorf("audio conversion from downstream to asterisk format failed: %w", err)
	}

	p.outputBuffer.Write(converted)

	return nil
}

// Complete flushes buffered trailing bytes by padding to a full frame.
func (p *AudioProcessor) Complete() {
	p.outputBuffer.Complete(p.getFrameSize(), p.getAdapter().SilenceByte())
}

func (p *AudioProcessor) GetNextChunk() *AudioChunk {
	chunkSize := p.getFrameSize()
	chunk, ok := p.outputBuffer.Next(chunkSize)
	if !ok {
		return nil
	}

	return &AudioChunk{
		Data:     chunk,
		Duration: chunkDuration,
	}
}

func (p *AudioProcessor) ClearOutputBuffer() {
	p.outputBuffer.Clear()
}

func (p *AudioProcessor) createSilenceChunk(chunkSize int, silenceByte byte) *AudioChunk {
	if chunkSize <= 0 {
		chunkSize = defaultFrameSize
	}
	chunk := make([]byte, chunkSize)
	for i := range chunk {
		chunk[i] = silenceByte
	}

	return &AudioChunk{
		Data:     chunk,
		Duration: chunkDuration,
	}
}

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
		FrameDuration: chunkDuration,
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
	return p.getAdapter().MixAmbient(chunk, p.ambientMixer)
}

func (p *AudioProcessor) NextFrame() []byte {
	if p.IsXOFF() {
		return nil
	}
	chunk := p.GetNextChunk()
	if chunk == nil {
		return nil
	}
	return p.applyAmbient(chunk.Data)
}

func (p *AudioProcessor) IdleFrame() []byte {
	if p.IsXOFF() {
		return nil
	}
	frame := p.applyAmbient(nil)
	if len(frame) > 0 {
		return frame
	}
	p.stateMu.RLock()
	defer p.stateMu.RUnlock()
	return append([]byte(nil), p.silenceChunk.Data...)
}

func (p *AudioProcessor) ConsumeFrame(frame []byte) error {
	return p.onOutputChunk(&AudioChunk{
		Data:     frame,
		Duration: chunkDuration,
	})
}

func (p *AudioProcessor) SetXOFF() {
	p.xoffMu.Lock()
	p.xoffActive = true
	p.xoffMu.Unlock()
}

func (p *AudioProcessor) SetXON() {
	p.xoffMu.Lock()
	p.xoffActive = false
	p.xoffMu.Unlock()
}

func (p *AudioProcessor) IsXOFF() bool {
	p.xoffMu.Lock()
	defer p.xoffMu.Unlock()
	return p.xoffActive
}

func (p *AudioProcessor) getFrameSize() int {
	p.stateMu.RLock()
	defer p.stateMu.RUnlock()
	if p.optimalFrameSize <= 0 {
		return defaultFrameSize
	}
	return p.optimalFrameSize
}

func (p *AudioProcessor) getAdapter() internal_telephony_output.AudioAdapter {
	p.stateMu.RLock()
	defer p.stateMu.RUnlock()
	return p.adapter
}
