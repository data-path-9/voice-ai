// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_sip_telephony

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	internal_telephony_output "github.com/rapidaai/api/assistant-api/internal/channel/output"
	"github.com/rapidaai/api/assistant-api/internal/observe"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	audioChSize    = 100
	chunkDuration  = 20 * time.Millisecond
	mulawFrameSize = 160 // 20ms at 8kHz, 1 byte/sample
)

var (
	rapida16kConfig = internal_audio.NewLinear16khzMonoAudioConfig()
	mulaw8kConfig   = internal_audio.NewMulaw8khzMonoAudioConfig()
	linear8kConfig  = internal_audio.NewLinear8khzMonoAudioConfig()
)

type bridgeState struct {
	outRTP    *sip_infra.RTPHandler
	transcode func([]byte) []byte
}

// AudioProcessor handles all RTP audio for a SIP streamer:
//   - Input: AudioIn → codec normalize → resample 8k→16k → onInputAudio callback
//   - Output: 16k LINEAR16 → resample 8k → codec convert → 20ms pacing → AudioOut
//   - Bridge: forward caller↔target + queue for recording via Talk pipeline
//   - Ringback: generate tone → AudioOut
type AudioProcessor struct {
	resampler  internal_type.AudioResampler
	rtpHandler *sip_infra.RTPHandler
	pushInput  func(internal_type.Stream)
	onInput    func([]byte)

	// Output buffering (AI TTS → RTP)
	outputBuffer internal_telephony_output.FrameBuffer
	flushCh      chan struct{}

	// Bridge state.
	// `bridge` is held in an atomic pointer so any goroutine can do a
	// lock-free `bridge.Load() != nil` check (IsBridgeActive, ProcessOutputAudio).
	// `bridgeMu` serializes ForwardUserAudio (the writer) with
	// ClearBridgeTarget so that Clear cannot return — and the outbound RTP
	// channel cannot be closed — while a send is in flight.
	bridgeMu         sync.Mutex
	bridge           atomic.Pointer[bridgeState]
	bridgeUserCh     chan []byte
	bridgeOperatorCh chan []byte

	// Ringback
	ringtoneMu sync.RWMutex
	ringtone   []byte

	ambientMixer internal_ambient.Mixer
	adapter      internal_telephony_output.AudioAdapter

	outputSenderRunning atomic.Bool
	outputHealth        *internal_telephony_output.HealthStats
	droppedOutputFrames atomic.Uint64
	transferActive      atomic.Bool
}

type AudioProcessorConfig struct {
	RTPHandler *sip_infra.RTPHandler
	Resampler  internal_type.AudioResampler
	PushInput  func(internal_type.Stream)
	Ringtone   string
	Ambient    *internal_ambient.Config
}

func NewAudioProcessor(cfg AudioProcessorConfig) *AudioProcessor {
	p := &AudioProcessor{
		resampler:        cfg.Resampler,
		rtpHandler:       cfg.RTPHandler,
		pushInput:        cfg.PushInput,
		outputBuffer:     internal_telephony_output.NewBytesFrameBuffer(mulawFrameSize * 8),
		flushCh:          make(chan struct{}, 1),
		bridgeUserCh:     make(chan []byte, audioChSize),
		bridgeOperatorCh: make(chan []byte, audioChSize),
		outputHealth:     internal_telephony_output.NewHealthStats(),
	}
	p.adapter = newAudioAdapter(cfg.Resampler, cfg.RTPHandler.GetCodec)
	p.SetRingtone(cfg.Ringtone)
	ambientMixer, err := internal_ambient.NewLoopMixer(internal_ambient.MixerSpec{
		Resampler:         cfg.Resampler,
		TargetAudioConfig: linear8kConfig,
		FrameBytes:        mulawFrameSize * 2,
	})
	if err == nil {
		p.ambientMixer = ambientMixer
		if cfg.Ambient != nil {
			_ = p.ambientMixer.Configure(*cfg.Ambient)
		}
	}
	return p
}

func (p *AudioProcessor) SetRingtone(name string) {
	audio := LoadRingtoneBytes(name)
	p.ringtoneMu.Lock()
	p.ringtone = audio
	p.ringtoneMu.Unlock()
}

func (p *AudioProcessor) ConfigureAmbient(cfg internal_ambient.Config) error {
	if p.ambientMixer == nil {
		return nil
	}
	return p.ambientMixer.Configure(cfg)
}

func (p *AudioProcessor) ringtoneBytes() []byte {
	p.ringtoneMu.RLock()
	defer p.ringtoneMu.RUnlock()
	return p.ringtone
}

func (p *AudioProcessor) SetInputAudioCallback(callback func(audio []byte)) {
	p.onInput = callback
}

func (p *AudioProcessor) convertInputAudio(audioData []byte) []byte {
	codec := p.rtpHandler.GetCodec()
	if codec != nil && codec.Name == "PCMA" {
		audioData = internal_audio.AlawToUlaw(audioData)
	}
	resampled, err := p.resampler.Resample(audioData, mulaw8kConfig, rapida16kConfig)
	if err != nil {
		return nil
	}
	return resampled
}

// ProcessInputAudio normalizes codec and resamples RTP audio to 16kHz LINEAR16.
// It forwards converted bytes to the registered input callback.
func (p *AudioProcessor) ProcessInputAudio(audioData []byte) error {
	resampled := p.convertInputAudio(audioData)
	if len(resampled) == 0 {
		return nil
	}
	if p.onInput != nil {
		p.onInput(resampled)
	}
	return nil
}

// ProcessOutputAudio resamples 16kHz LINEAR16 to 8kHz µ-law and buffers for pacing.
// Discards audio when a bridge is active — the caller hears the operator, not the AI.
func (p *AudioProcessor) ProcessOutputAudio(audioData []byte) error {
	if p.bridge.Load() != nil || p.transferActive.Load() {
		return nil
	}
	outData, err := p.adapter.ConvertOutput(audioData)
	if err != nil {
		return err
	}
	p.outputBuffer.Write(outData)
	return nil
}

func (p *AudioProcessor) Complete() {
	p.outputBuffer.Complete(mulawFrameSize, p.adapter.SilenceByte())
}

func (p *AudioProcessor) getNextChunk() []byte {
	chunk, ok := p.outputBuffer.Next(mulawFrameSize)
	if !ok {
		return nil
	}
	return chunk
}

func (p *AudioProcessor) ClearOutputBuffer() {
	p.outputBuffer.Clear()
	select {
	case p.flushCh <- struct{}{}:
	default:
	}
}

func (p *AudioProcessor) WaitOutputDrain(ctx context.Context) {
	ticker := time.NewTicker(chunkDuration)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			empty := p.outputBuffer.Len() == 0
			if empty {
				return
			}
		}
	}
}

// RunOutputSender paces 20ms audio frames to the RTP handler. Blocks until ctx is cancelled.
func (p *AudioProcessor) RunOutputSender(ctx context.Context) {
	if !p.outputSenderRunning.CompareAndSwap(false, true) {
		return
	}
	defer p.outputSenderRunning.Store(false)

	go p.runFlushSignalWorker(ctx)
	(&internal_telephony_output.Pacer{
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

func (p *AudioProcessor) runFlushSignalWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.flushCh:
			p.rtpHandler.FlushAudioOut()
		}
	}
}

func (p *AudioProcessor) applyAmbient(chunk []byte) []byte {
	return p.adapter.MixAmbient(chunk, p.ambientMixer)
}

func (p *AudioProcessor) NextFrame() []byte {
	if p.transferActive.Load() {
		return nil
	}
	chunk := p.getNextChunk()
	if len(chunk) == 0 {
		return nil
	}
	return p.applyAmbient(chunk)
}

func (p *AudioProcessor) IdleFrame() []byte {
	if p.transferActive.Load() {
		return nil
	}
	return p.applyAmbient(nil)
}

func (p *AudioProcessor) SetTransferActive(active bool) {
	p.transferActive.Store(active)
}

func (p *AudioProcessor) ConsumeFrame(frame []byte) error {
	select {
	case p.rtpHandler.AudioOut() <- frame:
	default:
		dropped := p.droppedOutputFrames.Add(1)
		if dropped == 1 || dropped%100 == 0 {
			if p.pushInput != nil {
				p.pushInput(&protos.ConversationEvent{
					Name: observe.ComponentTelephony,
					Data: map[string]string{
						"type":                 "output_send_error",
						"provider":             "sip",
						"reason":               "audio_out_full",
						"dropped_frames_total": fmt.Sprintf("%d", dropped),
					},
					Time: timestamppb.Now(),
				})
			}
		}
	}
	return nil
}

// =============================================================================
// Bridge
// =============================================================================

func (p *AudioProcessor) IsBridgeActive() bool {
	return p.bridge.Load() != nil
}

func (p *AudioProcessor) SetBridgeTarget(rtp *sip_infra.RTPHandler, inCodec, outCodec *sip_infra.Codec) {
	if rtp == nil {
		return
	}
	bs := &bridgeState{outRTP: rtp}
	if inCodec != nil && outCodec != nil && inCodec.Name != outCodec.Name {
		if inCodec.Name == "PCMA" && outCodec.Name == "PCMU" {
			bs.transcode = internal_audio.AlawToUlaw
		} else if inCodec.Name == "PCMU" && outCodec.Name == "PCMA" {
			bs.transcode = internal_audio.UlawToAlaw
		}
	}
	p.bridgeMu.Lock()
	p.bridge.Store(bs)
	p.bridgeMu.Unlock()
}

// ClearBridgeTarget releases the bridge target. Blocks until any in-flight
// ForwardUserAudio call has finished writing — this is the load-bearing
// invariant: callers may close the outbound RTP channel only AFTER this
// returns, since no writer holds a reference to it past that point.
func (p *AudioProcessor) ClearBridgeTarget() {
	p.bridgeMu.Lock()
	p.bridge.Store(nil)
	p.bridgeMu.Unlock()
}

// ForwardUserAudio routes caller audio to the bridge target and queues it
// for recording. Returns true if bridge is active and audio was handled.
//
// Holds bridgeMu for the duration of the send so that ClearBridgeTarget
// cannot return — and therefore the outbound RTP channel cannot be closed —
// while a send is in flight.
func (p *AudioProcessor) ForwardUserAudio(audioData []byte) bool {
	p.bridgeMu.Lock()
	defer p.bridgeMu.Unlock()
	bs := p.bridge.Load()
	if bs == nil {
		return false
	}
	rawAudio := audioData
	if bs.transcode != nil {
		audioData = bs.transcode(audioData)
	}
	select {
	case bs.outRTP.AudioOut() <- audioData:
	default:
	}
	select {
	case p.bridgeUserCh <- rawAudio:
	default:
	}
	return true
}

// PushOperatorAudio queues transfer target audio for recording.
func (p *AudioProcessor) PushOperatorAudio(audio []byte) {
	select {
	case p.bridgeOperatorCh <- audio:
	default:
	}
}

// RunBridgeRecorder resamples queued bridge audio and pushes it into the Talk pipeline.
// Blocks until ctx is cancelled.
func (p *AudioProcessor) RunBridgeRecorder(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case audio := <-p.bridgeUserCh:
			if resampled, err := p.resampler.Resample(audio, mulaw8kConfig, rapida16kConfig); err == nil {
				p.pushInput(&protos.ConversationBridgeUserAudio{Audio: resampled})
			}
		case audio := <-p.bridgeOperatorCh:
			if resampled, err := p.resampler.Resample(audio, mulaw8kConfig, rapida16kConfig); err == nil {
				p.pushInput(&protos.ConversationBridgeOperatorAudio{Audio: resampled})
			}
		}
	}
}

// =============================================================================
// Ringback
// =============================================================================

// PlayRingback generates a standard ringback tone and writes directly to AudioOut.
// Blocks until ctx is cancelled.
func (p *AudioProcessor) PlayRingback(ctx context.Context) {
	codec := p.rtpHandler.GetCodec()
	ticker := time.NewTicker(chunkDuration)
	defer ticker.Stop()

	ringtone := p.ringtoneBytes()
	useFile := len(ringtone) >= mulawFrameSize
	fileOffset := 0
	offset := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var frame []byte
			if useFile {
				if len(ringtone) >= mulawFrameSize {
					end := fileOffset + mulawFrameSize
					if end > len(ringtone) {
						fileOffset = 0
						end = mulawFrameSize
					}
					if end <= len(ringtone) {
						frame = ringtone[fileOffset:end]
						fileOffset = end
					}
				}
				if len(frame) == 0 {
					frame, offset = internal_audio.GenerateRingbackMulawFrame(offset)
				}
			} else {
				frame, offset = internal_audio.GenerateRingbackMulawFrame(offset)
			}
			if codec != nil && codec.Name == "PCMA" {
				frame = internal_audio.UlawToAlaw(frame)
			}
			select {
			case p.rtpHandler.AudioOut() <- frame:
			case <-ctx.Done():
				return
			default:
			}
		}
	}
}
