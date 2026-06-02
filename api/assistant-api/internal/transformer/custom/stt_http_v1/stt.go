// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_stt_http_v1

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_audio_resampler "github.com/rapidaai/api/assistant-api/internal/audio/resampler"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type speechToText struct {
	config *Config
	engine *dslEngine

	ctx        context.Context
	cancel     context.CancelFunc
	httpClient *http.Client

	logger   commons.Logger
	onPacket func(pkt ...internal_type.Packet) error

	mu                 sync.Mutex
	contextID          string
	connectedAt        time.Time
	speechStartedAt    time.Time
	userSpeaking       bool
	speechAudioBuffer  bytes.Buffer
	activeRequestCount int

	resampler         internal_type.AudioResampler
	sourceAudioConfig *protos.AudioConfig
	targetAudioConfig *protos.AudioConfig
}

func NewSpeechToText(
	ctx context.Context,
	logger commons.Logger,
	credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option,
) (internal_type.SpeechToTextTransformer, error) {
	config, err := NewConfig(credential, opts)
	if err != nil {
		return nil, err
	}
	resampler, err := internal_audio_resampler.GetResampler(logger)
	if err != nil {
		return nil, fmt.Errorf("custom-stt http_v1: failed to initialize audio resampler: %w", err)
	}
	transformerContext, cancel := context.WithCancel(ctx)
	return &speechToText{
		config:            config,
		engine:            config.newEngine(),
		ctx:               transformerContext,
		cancel:            cancel,
		httpClient:        &http.Client{Timeout: 60 * time.Second},
		logger:            logger,
		onPacket:          onPacket,
		resampler:         resampler,
		sourceAudioConfig: internal_audio.RAPIDA_INTERNAL_AUDIO_CONFIG,
		targetAudioConfig: &protos.AudioConfig{
			SampleRate:  uint32(config.SampleRate),
			AudioFormat: protos.AudioConfig_LINEAR16,
			Channels:    1,
		},
	}, nil
}

func (*speechToText) Name() string {
	return "custom-stt-http-v1"
}

func (transformer *speechToText) Initialize() error {
	start := time.Now()
	transformer.mu.Lock()
	transformer.connectedAt = time.Now()
	contextID := transformer.contextID
	transformer.mu.Unlock()

	transformer.emitPackets(internal_type.ConversationEventPacket{
		ContextID: contextID,
		Name:      "stt",
		Data: map[string]string{
			"type":     "initialized",
			"provider": transformer.Name(),
			"init_ms":  fmt.Sprintf("%d", time.Since(start).Milliseconds()),
		},
		Time: time.Now(),
	})
	return nil
}

func (transformer *speechToText) Transform(_ context.Context, in internal_type.Packet) error {
	switch input := in.(type) {
	case internal_type.TurnChangePacket:
		transformer.mu.Lock()
		transformer.contextID = input.ContextID
		transformer.mu.Unlock()
		return nil
	case internal_type.SpeechToTextEndPacket:
		transformer.flushBufferedSpeech(input.ContextID)
		return nil
	case internal_type.SpeechToTextStartPacket:
		transformer.mu.Lock()
		if input.ContextID != "" {
			transformer.contextID = input.ContextID
		}
		transformer.userSpeaking = true
		transformer.speechStartedAt = time.Now()
		transformer.speechAudioBuffer.Reset()
		transformer.mu.Unlock()
		return nil
	case internal_type.SpeechToTextAudioPacket:
		if len(input.Audio) == 0 {
			return nil
		}
		chunk, err := transformer.prepareAudioChunk(input.Audio)
		if err != nil {
			transformer.emitPackets(internal_type.SpeechToTextErrorPacket{
				ContextID: transformer.currentContextID(),
				Error:     err,
				Type:      internal_type.STTInvalidInput,
			})
			return nil
		}
		transformer.mu.Lock()
		if input.ContextID != "" {
			transformer.contextID = input.ContextID
		}
		if transformer.userSpeaking {
			_, _ = transformer.speechAudioBuffer.Write(chunk)
		}
		transformer.mu.Unlock()
		return nil
	default:
		return nil
	}
}

func (transformer *speechToText) Close(_ context.Context) error {
	transformer.cancel()

	transformer.mu.Lock()
	contextID := transformer.contextID
	connectedAt := transformer.connectedAt
	transformer.contextID = ""
	transformer.connectedAt = time.Time{}
	transformer.speechStartedAt = time.Time{}
	transformer.userSpeaking = false
	transformer.speechAudioBuffer.Reset()
	transformer.mu.Unlock()

	if !connectedAt.IsZero() {
		transformer.emitPackets(
			internal_type.ConversationEventPacket{
				ContextID: contextID,
				Name:      "stt",
				Data: map[string]string{
					"type":     "closed",
					"provider": transformer.Name(),
				},
				Time: time.Now(),
			},
			internal_type.ConversationMetricPacket{
				ContextID: 0,
				Metrics: []*protos.Metric{{
					Name:        type_enums.CONVERSATION_STT_DURATION.String(),
					Value:       fmt.Sprintf("%d", time.Since(connectedAt).Nanoseconds()),
					Description: "Total STT connection duration in nanoseconds",
				}},
			},
		)
	}

	return nil
}

func (transformer *speechToText) flushBufferedSpeech(contextID string) {
	transformer.mu.Lock()
	if contextID != "" {
		transformer.contextID = contextID
	}
	effectiveContextID := transformer.contextID
	startedAt := transformer.speechStartedAt
	audioData := make([]byte, transformer.speechAudioBuffer.Len())
	copy(audioData, transformer.speechAudioBuffer.Bytes())
	transformer.speechAudioBuffer.Reset()
	transformer.userSpeaking = false
	transformer.speechStartedAt = time.Time{}
	if len(audioData) > 0 {
		transformer.activeRequestCount++
	}
	transformer.mu.Unlock()
	if len(audioData) == 0 {
		return
	}

	// VAD end is the HTTP flush boundary; the provider receives one WAV per user turn.
	utils.Go(transformer.ctx, func() {
		defer func() {
			transformer.mu.Lock()
			transformer.activeRequestCount--
			transformer.mu.Unlock()
		}()
		transformer.transcribe(effectiveContextID, audioData, startedAt)
	})
}

func (transformer *speechToText) transcribe(contextID string, pcmAudio []byte, startedAt time.Time) {
	wavAudio, err := createPCM16MonoWAV(pcmAudio, transformer.config.SampleRate)
	if err != nil {
		transformer.emitPackets(internal_type.SpeechToTextErrorPacket{
			ContextID: contextID,
			Error:     err,
			Type:      internal_type.STTInvalidInput,
		})
		return
	}

	requestURL, err := transformer.engine.BuildRequestURL(transformer.config.newQueryScope())
	if err != nil {
		transformer.emitPackets(internal_type.SpeechToTextErrorPacket{
			ContextID: contextID,
			Error:     err,
			Type:      internal_type.STTInvalidInput,
		})
		return
	}

	requests, err := transformer.engine.EvaluateRequestRules(
		requestPacketAudio,
		transformer.config.newRequestScope(contextID, pcmAudio, wavAudio),
	)
	if err != nil {
		transformer.emitPackets(internal_type.SpeechToTextErrorPacket{
			ContextID: contextID,
			Error:     fmt.Errorf("custom-stt http_v1: failed to evaluate request rules: %w", err),
			Type:      internal_type.STTInvalidInput,
		})
		return
	}
	if len(requests) == 0 {
		transformer.emitPackets(internal_type.SpeechToTextErrorPacket{
			ContextID: contextID,
			Error:     fmt.Errorf("custom-stt http_v1: request rules produced no audio request"),
			Type:      internal_type.STTInvalidInput,
		})
		return
	}
	requestBody := requests[0].Body
	if requests[0].Frame != frameTypeJSON {
		transformer.emitPackets(internal_type.SpeechToTextErrorPacket{
			ContextID: contextID,
			Error:     fmt.Errorf("custom-stt http_v1: audio request rule send.frame must be json"),
			Type:      internal_type.STTInvalidInput,
		})
		return
	}
	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		transformer.emitPackets(internal_type.SpeechToTextErrorPacket{
			ContextID: contextID,
			Error:     fmt.Errorf("custom-stt http_v1: failed to marshal request body: %w", err),
			Type:      internal_type.STTInvalidInput,
		})
		return
	}

	request, err := http.NewRequestWithContext(transformer.ctx, http.MethodPost, requestURL, bytes.NewReader(requestBodyBytes))
	if err != nil {
		transformer.emitPackets(internal_type.SpeechToTextErrorPacket{
			ContextID: contextID,
			Error:     fmt.Errorf("custom-stt http_v1: failed to create request: %w", err),
			Type:      internal_type.STTNetworkTimeout,
		})
		return
	}
	for key, value := range transformer.config.Headers {
		request.Header.Set(key, value)
	}
	if request.Header.Get("Content-Type") == "" {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := transformer.httpClient.Do(request)
	if err != nil {
		if transformer.ctx.Err() != nil || errors.Is(err, context.Canceled) {
			return
		}
		transformer.emitPackets(internal_type.SpeechToTextErrorPacket{
			ContextID: contextID,
			Error:     fmt.Errorf("custom-stt http_v1: request failed: %w", err),
			Type:      internal_type.STTNetworkTimeout,
		})
		return
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	transformer.logger.Debugf("************* %s", string(responseBody))
	if err != nil {
		transformer.emitPackets(internal_type.SpeechToTextErrorPacket{
			ContextID: contextID,
			Error:     fmt.Errorf("custom-stt http_v1: failed to read response: %w", err),
			Type:      internal_type.STTNetworkTimeout,
		})
		return
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		transformer.emitPackets(internal_type.SpeechToTextErrorPacket{
			ContextID: contextID,
			Error:     fmt.Errorf("custom-stt http_v1: status %d: %s", response.StatusCode, string(responseBody)),
			Type:      classifyHTTPStatus(response.StatusCode),
		})
		return
	}

	frame, err := transformer.engine.ParseHTTPResponse(responseBody)
	if err != nil {
		transformer.emitPackets(internal_type.SpeechToTextErrorPacket{
			ContextID: contextID,
			Error:     err,
			Type:      internal_type.STTSystemPanic,
		})
		return
	}
	outcome, err := transformer.engine.EvaluateResponse(frame)
	if err != nil {
		transformer.emitPackets(internal_type.SpeechToTextErrorPacket{
			ContextID: contextID,
			Error:     err,
			Type:      internal_type.STTSystemPanic,
		})
		return
	}
	if !outcome.Matched {
		return
	}
	if strings.TrimSpace(outcome.ErrorText) != "" {
		transformer.emitPackets(internal_type.SpeechToTextErrorPacket{
			ContextID: contextID,
			Error:     errors.New(strings.TrimSpace(outcome.ErrorText)),
			Type:      internal_type.STTSystemPanic,
		})
		return
	}
	if strings.TrimSpace(outcome.Script) == "" {
		return
	}

	transformer.emitTranscript(contextID, outcome, startedAt)
}

func (transformer *speechToText) emitTranscript(contextID string, outcome responseOutcome, startedAt time.Time) {
	now := time.Now()
	language := strings.TrimSpace(outcome.Language)
	if language == "" {
		language = transformer.config.Language
	}

	eventData := map[string]string{
		"type":       "completed",
		"script":     outcome.Script,
		"confidence": fmt.Sprintf("%.4f", outcome.Confidence),
		"word_count": fmt.Sprintf("%d", len(strings.Fields(outcome.Script))),
		"char_count": fmt.Sprintf("%d", len(outcome.Script)),
	}
	if language != "" {
		eventData["language"] = language
	}

	packets := []internal_type.Packet{
		internal_type.InterruptionDetectedPacket{
			ContextID: contextID,
			Source:    internal_type.InterruptionSourceWord,
		},
		internal_type.SpeechToTextPacket{
			ContextID:  contextID,
			Script:     outcome.Script,
			Concat:     utils.Ptr(""),
			Confidence: outcome.Confidence,
			Language:   language,
			Interim:    false,
		},
		internal_type.ConversationEventPacket{
			ContextID: contextID,
			Name:      "stt",
			Data:      eventData,
			Time:      now,
		},
	}
	if !startedAt.IsZero() {
		packets = append(packets, internal_type.UserMessageMetricPacket{
			ContextID: contextID,
			Metrics: []*protos.Metric{{
				Name:  "stt_latency_ms",
				Value: fmt.Sprintf("%d", now.Sub(startedAt).Milliseconds()),
			}},
		})
	}

	transformer.emitPackets(packets...)
}

func (transformer *speechToText) prepareAudioChunk(audio []byte) ([]byte, error) {
	chunk := audio
	if transformer.resampler != nil {
		resampled, err := transformer.resampler.Resample(chunk, transformer.sourceAudioConfig, transformer.targetAudioConfig)
		if err != nil {
			return nil, fmt.Errorf("custom-stt http_v1: failed to resample audio: %w", err)
		}
		chunk = resampled
	}
	return chunk, nil
}

func (transformer *speechToText) currentContextID() string {
	transformer.mu.Lock()
	defer transformer.mu.Unlock()
	return transformer.contextID
}

func (transformer *speechToText) emitPackets(packets ...internal_type.Packet) {
	if err := transformer.onPacket(packets...); err != nil {
		transformer.logger.Errorf("custom-stt http_v1: onPacket failed: %v", err)
	}
}

func classifyHTTPStatus(statusCode int) internal_type.STTErrorType {
	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return internal_type.STTAuthentication
	case statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError:
		return internal_type.STTInvalidInput
	default:
		return internal_type.STTNetworkTimeout
	}
}

func createPCM16MonoWAV(pcmAudio []byte, sampleRate int) ([]byte, error) {
	if sampleRate <= 0 {
		return nil, fmt.Errorf("custom-stt http_v1: sample rate must be positive")
	}
	if len(pcmAudio)%2 != 0 {
		return nil, fmt.Errorf("custom-stt http_v1: LINEAR16 audio must contain complete samples")
	}

	dataSize := len(pcmAudio)
	byteRate := sampleRate * 2
	totalSize := 36 + dataSize

	header := make([]byte, 44)
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], uint32(totalSize))
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16)
	binary.LittleEndian.PutUint16(header[20:22], 1)
	binary.LittleEndian.PutUint16(header[22:24], 1)
	binary.LittleEndian.PutUint32(header[24:28], uint32(sampleRate))
	binary.LittleEndian.PutUint32(header[28:32], uint32(byteRate))
	binary.LittleEndian.PutUint16(header[32:34], 2)
	binary.LittleEndian.PutUint16(header[34:36], 16)
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], uint32(dataSize))

	wavAudio := make([]byte, 0, len(header)+len(pcmAudio))
	wavAudio = append(wavAudio, header...)
	wavAudio = append(wavAudio, pcmAudio...)
	return wavAudio, nil
}

func parseAudioEncoding(encoding string) protos.AudioConfig_AudioFormat {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "mulaw", "mu-law", "mulaw8", "mu_law", "ulaw", "u-law", "pcmu", "g711_ulaw":
		return protos.AudioConfig_MuLaw8
	default:
		return protos.AudioConfig_LINEAR16
	}
}

func cloneAudioConfig(config *protos.AudioConfig) *protos.AudioConfig {
	if config == nil {
		return &protos.AudioConfig{SampleRate: 16000, AudioFormat: protos.AudioConfig_LINEAR16, Channels: 1}
	}
	return &protos.AudioConfig{
		SampleRate:  config.GetSampleRate(),
		AudioFormat: config.GetAudioFormat(),
		Channels:    config.GetChannels(),
	}
}

func isSameAudioConfig(left, right *protos.AudioConfig) bool {
	if left == nil || right == nil {
		return false
	}
	return left.GetSampleRate() == right.GetSampleRate() &&
		left.GetAudioFormat() == right.GetAudioFormat() &&
		left.GetChannels() == right.GetChannels()
}
