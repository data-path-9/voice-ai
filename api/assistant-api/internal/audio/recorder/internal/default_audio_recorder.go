// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_recorder

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
)

const (
	linear16BytesPerSample = 2
	linear16BitsPerSample  = 16
	wavPCMFormat           = 1
	wavHeaderSize          = 44
)

const recordingTrackCount = 2

type recordingTrack int

const (
	userRecordingTrack recordingTrack = iota
	assistantRecordingTrack
)

var recorderAudioConfig = internal_audio.RAPIDA_INTERNAL_AUDIO_CONFIG

type recordedAudioChunk struct {
	TimelineByteOffset int
	PCM16              []byte
	Track              recordingTrack
}

// conversationRecordingExecutor captures timestamped user and assistant audio on aligned tracks.
type conversationRecordingExecutor struct {
	contextID             string
	emitPacket            func(context.Context, ...internal_type.Packet) error
	recorderLock          sync.Mutex
	recordingClosed       bool
	timelineAnchorTime    time.Time
	timelineChunks        []recordedAudioChunk
	trackWriteCursorBytes [recordingTrackCount]int
}

func NewConversationRecordingExecutor(
	contextID string,
	emitPacket func(context.Context, ...internal_type.Packet) error,
) (internal_type.ConversationRecordingExecutor, error) {
	return &conversationRecordingExecutor{
		contextID:  contextID,
		emitPacket: emitPacket,
	}, nil
}

func (r *conversationRecordingExecutor) Name() string { return "conversation_recording" }

func (r *conversationRecordingExecutor) Options() utils.Option { return utils.Option{} }

func (r *conversationRecordingExecutor) Arguments() (map[string]string, error) {
	return map[string]string{}, nil
}

func (r *conversationRecordingExecutor) Execute(_ context.Context, packet internal_type.Packet) error {
	switch pkt := packet.(type) {
	case internal_type.RecordUserAudioPacket:
		return r.recordPCM16Chunk(pkt.Audio, userRecordingTrack, pkt.Timestamp)
	case internal_type.RecordAssistantAudioPacket:
		return r.recordPCM16Chunk(pkt.Audio, assistantRecordingTrack, pkt.Timestamp)
	}
	return nil
}

func (r *conversationRecordingExecutor) Close(ctx context.Context) error {
	r.recorderLock.Lock()
	if r.recordingClosed {
		r.recorderLock.Unlock()
		return nil
	}
	r.recordingClosed = true

	if len(r.timelineChunks) == 0 {
		r.recorderLock.Unlock()
		return fmt.Errorf("no audio chunks to persist")
	}

	timelinePCMByteLength := 0
	for _, recordedChunk := range r.timelineChunks {
		chunkEndByteOffset := recordedChunk.TimelineByteOffset + len(recordedChunk.PCM16)
		if chunkEndByteOffset > timelinePCMByteLength {
			timelinePCMByteLength = chunkEndByteOffset
		}
	}

	trackPCM16 := [recordingTrackCount][]byte{
		make([]byte, timelinePCMByteLength),
		make([]byte, timelinePCMByteLength),
	}

	for _, recordedChunk := range r.timelineChunks {
		copy(trackPCM16[recordedChunk.Track][recordedChunk.TimelineByteOffset:], recordedChunk.PCM16)
	}

	userWAV, err := encodeWAV(trackPCM16[userRecordingTrack], 1)
	if err != nil {
		r.recorderLock.Unlock()
		return fmt.Errorf("encoding user WAV: %w", err)
	}
	assistantWAV, err := encodeWAV(trackPCM16[assistantRecordingTrack], 1)
	if err != nil {
		r.recorderLock.Unlock()
		return fmt.Errorf("encoding assistant WAV: %w", err)
	}

	monoSampleCount := min(len(trackPCM16[userRecordingTrack]), len(trackPCM16[assistantRecordingTrack])) / linear16BytesPerSample
	mixedPCM16 := make([]byte, monoSampleCount*linear16BytesPerSample*2)
	for sampleIndex := 0; sampleIndex < monoSampleCount; sampleIndex++ {
		monoByteOffset := sampleIndex * linear16BytesPerSample
		stereoByteOffset := sampleIndex * linear16BytesPerSample * 2
		copy(mixedPCM16[stereoByteOffset:stereoByteOffset+linear16BytesPerSample], trackPCM16[userRecordingTrack][monoByteOffset:monoByteOffset+linear16BytesPerSample])
		copy(mixedPCM16[stereoByteOffset+linear16BytesPerSample:stereoByteOffset+linear16BytesPerSample*2], trackPCM16[assistantRecordingTrack][monoByteOffset:monoByteOffset+linear16BytesPerSample])
	}

	mixedWAV, err := encodeWAV(mixedPCM16, 2)
	if err != nil {
		r.recorderLock.Unlock()
		return fmt.Errorf("encoding conversation WAV: %w", err)
	}

	recordingAudio := internal_type.ConversationRecordingAudio{
		UserAudio:      userWAV,
		AssistantAudio: assistantWAV,
		MixedAudio:     mixedWAV,
	}
	r.recorderLock.Unlock()

	if r.emitPacket == nil {
		return nil
	}
	if err := r.emitPacket(ctx, internal_type.ConversationRecordingCompletedPacket{
		ContextID: r.contextID,
		Audio:     recordingAudio,
	}); err != nil {
		return err
	}
	return nil
}

func (r *conversationRecordingExecutor) recordPCM16Chunk(pcm16Audio []byte, track recordingTrack, mediaTimestamp time.Time) error {
	if len(pcm16Audio) == 0 {
		return nil
	}
	if mediaTimestamp.IsZero() {
		return fmt.Errorf("record audio missing timestamp")
	}

	r.recorderLock.Lock()
	defer r.recorderLock.Unlock()

	if r.recordingClosed {
		return nil
	}

	mediaTimestampOffsetBytes := 0
	if r.timelineAnchorTime.IsZero() {
		r.timelineAnchorTime = mediaTimestamp
	} else if mediaTimestamp.After(r.timelineAnchorTime) || mediaTimestamp.Equal(r.timelineAnchorTime) {
		rawOffsetBytes := int(mediaTimestamp.Sub(r.timelineAnchorTime).Seconds() * float64(internal_audio.BytesPerSecond(recorderAudioConfig)))
		frameBytes := internal_audio.FrameSize(recorderAudioConfig)
		mediaTimestampOffsetBytes = (rawOffsetBytes / frameBytes) * frameBytes
	}

	timelineByteOffset := mediaTimestampOffsetBytes
	if r.trackWriteCursorBytes[track] > timelineByteOffset {
		timelineByteOffset = r.trackWriteCursorBytes[track]
	}

	pcm16Copy := make([]byte, len(pcm16Audio))
	copy(pcm16Copy, pcm16Audio)

	r.timelineChunks = append(r.timelineChunks, recordedAudioChunk{
		TimelineByteOffset: timelineByteOffset,
		PCM16:              pcm16Copy,
		Track:              track,
	})

	r.trackWriteCursorBytes[track] = timelineByteOffset + len(pcm16Copy)
	return nil
}

// encodeWAV wraps raw PCM data in a canonical WAV (RIFF) container.
// Format: 16-bit LINEAR PCM at the configured sample rate and channel count.
func encodeWAV(pcm16Audio []byte, channelCount int) ([]byte, error) {
	sampleRate := recorderAudioConfig.SampleRate
	blockAlign := uint16(channelCount) * uint16(linear16BytesPerSample)
	byteRate := uint32(sampleRate) * uint32(blockAlign)

	wavBuffer := bytes.NewBuffer(make([]byte, 0, wavHeaderSize+len(pcm16Audio)))

	wavBuffer.Write([]byte("RIFF"))
	if err := binary.Write(wavBuffer, binary.LittleEndian, uint32(36+len(pcm16Audio))); err != nil {
		return nil, fmt.Errorf("writing RIFF size: %w", err)
	}
	wavBuffer.Write([]byte("WAVE"))

	wavBuffer.Write([]byte("fmt "))
	if err := binary.Write(wavBuffer, binary.LittleEndian, uint32(16)); err != nil {
		return nil, fmt.Errorf("writing fmt chunk size: %w", err)
	}
	if err := binary.Write(wavBuffer, binary.LittleEndian, uint16(wavPCMFormat)); err != nil {
		return nil, fmt.Errorf("writing audio format: %w", err)
	}
	if err := binary.Write(wavBuffer, binary.LittleEndian, uint16(channelCount)); err != nil {
		return nil, fmt.Errorf("writing channels: %w", err)
	}
	if err := binary.Write(wavBuffer, binary.LittleEndian, uint32(sampleRate)); err != nil {
		return nil, fmt.Errorf("writing sample rate: %w", err)
	}
	if err := binary.Write(wavBuffer, binary.LittleEndian, byteRate); err != nil {
		return nil, fmt.Errorf("writing byte rate: %w", err)
	}
	if err := binary.Write(wavBuffer, binary.LittleEndian, blockAlign); err != nil {
		return nil, fmt.Errorf("writing block align: %w", err)
	}
	if err := binary.Write(wavBuffer, binary.LittleEndian, uint16(linear16BitsPerSample)); err != nil {
		return nil, fmt.Errorf("writing bits per sample: %w", err)
	}

	wavBuffer.Write([]byte("data"))
	if err := binary.Write(wavBuffer, binary.LittleEndian, uint32(len(pcm16Audio))); err != nil {
		return nil, fmt.Errorf("writing data chunk size: %w", err)
	}
	wavBuffer.Write(pcm16Audio)

	return wavBuffer.Bytes(), nil
}
