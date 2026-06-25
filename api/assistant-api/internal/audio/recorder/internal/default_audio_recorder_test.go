// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_recorder

import (
	"context"
	"encoding/binary"
	"testing"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

func newTestRecorder(t *testing.T) *conversationRecordingExecutor {
	t.Helper()
	rec, err := New(
		WithContextID("ctx-recording"),
		WithOnPacket(func(context.Context, ...internal_type.Packet) error {
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("failed to create recorder: %v", err)
	}
	return rec.(*conversationRecordingExecutor)
}

func testMediaTimestamp() time.Time {
	return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
}

func pcm(fillByte byte, byteLength int) []byte {
	pcm16Audio := make([]byte, byteLength)
	for audioByteIndex := range pcm16Audio {
		pcm16Audio[audioByteIndex] = fillByte
	}
	return pcm16Audio
}

func wavPCMData(wav []byte) []byte { return wav[wavHeaderSize:] }

func wavChannels(wav []byte) uint16 { return binary.LittleEndian.Uint16(wav[22:24]) }

func TestFirstPacketAnchorsTimeline(t *testing.T) {
	rec := newTestRecorder(t)

	err := rec.Execute(context.Background(), internal_type.RecordUserAudioPacket{
		Audio:     pcm(0x11, 100),
		Timestamp: testMediaTimestamp().Add(100 * time.Millisecond),
	})
	if err != nil {
		t.Fatalf("Record error: %v", err)
	}

	if len(rec.timelineChunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(rec.timelineChunks))
	}
	if rec.timelineChunks[0].Track != userRecordingTrack {
		t.Fatalf("expected user track, got %d", rec.timelineChunks[0].Track)
	}
	if rec.timelineChunks[0].TimelineByteOffset != 0 {
		t.Fatalf("expected first chunk offset 0, got %d", rec.timelineChunks[0].TimelineByteOffset)
	}
}

func TestRecordUserAudioUsesTimestampAfterAnchor(t *testing.T) {
	rec := newTestRecorder(t)
	ctx := context.Background()
	mediaStart := testMediaTimestamp()

	rec.Execute(ctx, internal_type.RecordUserAudioPacket{
		Audio:     pcm(0x11, 100),
		Timestamp: mediaStart,
	})
	err := rec.Execute(ctx, internal_type.RecordUserAudioPacket{
		Audio:     pcm(0x12, 100),
		Timestamp: mediaStart.Add(100 * time.Millisecond),
	})
	if err != nil {
		t.Fatalf("Record error: %v", err)
	}

	if rec.timelineChunks[1].TimelineByteOffset != 3200 {
		t.Fatalf("expected timestamp offset, got %d", rec.timelineChunks[1].TimelineByteOffset)
	}
}

func TestRecordAssistantAudioUsesTimestampAfterAnchor(t *testing.T) {
	rec := newTestRecorder(t)
	ctx := context.Background()
	mediaStart := testMediaTimestamp()

	rec.Execute(ctx, internal_type.RecordAssistantAudioPacket{
		Audio:     pcm(0x21, 100),
		Timestamp: mediaStart,
	})
	err := rec.Execute(ctx, internal_type.RecordAssistantAudioPacket{
		Audio:     pcm(0x22, 100),
		Timestamp: mediaStart.Add(250 * time.Millisecond),
	})
	if err != nil {
		t.Fatalf("Record error: %v", err)
	}

	if len(rec.timelineChunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(rec.timelineChunks))
	}
	if rec.timelineChunks[1].Track != assistantRecordingTrack {
		t.Fatalf("expected assistant track, got %d", rec.timelineChunks[1].Track)
	}
	if rec.timelineChunks[1].TimelineByteOffset != 8000 {
		t.Fatalf("expected timestamp offset, got %d", rec.timelineChunks[1].TimelineByteOffset)
	}
}

func TestRecordRequiresTimestamp(t *testing.T) {
	rec := newTestRecorder(t)
	ctx := context.Background()

	if err := rec.Execute(ctx, internal_type.RecordUserAudioPacket{Audio: pcm(0x11, 100)}); err == nil {
		t.Fatal("expected user audio without timestamp to fail")
	}
	if err := rec.Execute(ctx, internal_type.RecordAssistantAudioPacket{Audio: pcm(0x22, 100)}); err == nil {
		t.Fatal("expected assistant audio without timestamp to fail")
	}
	if len(rec.timelineChunks) != 0 {
		t.Fatalf("expected no chunks, got %d", len(rec.timelineChunks))
	}
}

func TestRecordEmptyDataIsIgnored(t *testing.T) {
	rec := newTestRecorder(t)
	ctx := context.Background()

	if err := rec.Execute(ctx, internal_type.RecordUserAudioPacket{}); err != nil {
		t.Fatalf("empty user audio should be ignored: %v", err)
	}
	if err := rec.Execute(ctx, internal_type.RecordAssistantAudioPacket{}); err != nil {
		t.Fatalf("empty assistant audio should be ignored: %v", err)
	}
	if len(rec.timelineChunks) != 0 {
		t.Fatalf("expected 0 chunks, got %d", len(rec.timelineChunks))
	}
}

func TestTimestampBeforeAnchorClampsToAnchor(t *testing.T) {
	rec := newTestRecorder(t)
	ctx := context.Background()
	mediaStart := testMediaTimestamp()

	rec.Execute(ctx, internal_type.RecordUserAudioPacket{
		Audio:     pcm(0x11, 100),
		Timestamp: mediaStart,
	})
	err := rec.Execute(ctx, internal_type.RecordAssistantAudioPacket{
		Audio:     pcm(0x11, 100),
		Timestamp: mediaStart.Add(-100 * time.Millisecond),
	})
	if err != nil {
		t.Fatalf("Record error: %v", err)
	}
	if rec.timelineChunks[1].TimelineByteOffset != 0 {
		t.Fatalf("expected timestamp before anchor to clamp to 0, got %d", rec.timelineChunks[1].TimelineByteOffset)
	}
}

func TestSameTrackCursorPreventsOverlap(t *testing.T) {
	rec := newTestRecorder(t)
	ctx := context.Background()
	timestamp := testMediaTimestamp().Add(100 * time.Millisecond)

	rec.Execute(ctx, internal_type.RecordAssistantAudioPacket{Audio: pcm(0xA1, 100), Timestamp: timestamp})
	rec.Execute(ctx, internal_type.RecordAssistantAudioPacket{Audio: pcm(0xA2, 100), Timestamp: timestamp})

	if len(rec.timelineChunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(rec.timelineChunks))
	}
	firstOffset := 0
	if rec.timelineChunks[0].TimelineByteOffset != firstOffset {
		t.Fatalf("first chunk offset = %d, want %d", rec.timelineChunks[0].TimelineByteOffset, firstOffset)
	}
	if rec.timelineChunks[1].TimelineByteOffset != firstOffset+100 {
		t.Fatalf("second chunk offset = %d, want %d", rec.timelineChunks[1].TimelineByteOffset, firstOffset+100)
	}
}

func TestPersistProducesUserAssistantAndMixedAudio(t *testing.T) {
	rec := newTestRecorder(t)
	ctx := context.Background()
	mediaStart := testMediaTimestamp()
	var recordingAudio internal_type.ConversationRecordingAudio
	rec.onPacket = func(_ context.Context, packets ...internal_type.Packet) error {
		recordingAudio = packets[0].(internal_type.ConversationRecordingCompletedPacket).Audio
		return nil
	}

	rec.Execute(ctx, internal_type.RecordUserAudioPacket{
		Audio:     pcm(0x11, 4),
		Timestamp: mediaStart,
	})
	rec.Execute(ctx, internal_type.RecordAssistantAudioPacket{
		Audio:     pcm(0x22, 4),
		Timestamp: mediaStart.Add(10 * time.Millisecond),
	})

	if err := rec.Close(ctx); err != nil {
		t.Fatalf("Persist error: %v", err)
	}
	if got := wavChannels(recordingAudio.UserAudio); got != 1 {
		t.Fatalf("user WAV channels = %d, want 1", got)
	}
	if got := wavChannels(recordingAudio.AssistantAudio); got != 1 {
		t.Fatalf("assistant WAV channels = %d, want 1", got)
	}
	if got := wavChannels(recordingAudio.MixedAudio); got != 2 {
		t.Fatalf("conversation WAV channels = %d, want 2", got)
	}

	userPCM := wavPCMData(recordingAudio.UserAudio)
	assistantPCM := wavPCMData(recordingAudio.AssistantAudio)
	conversationPCM := wavPCMData(recordingAudio.MixedAudio)
	assistantOffset := 320

	if userPCM[0] != 0x11 || userPCM[3] != 0x11 {
		t.Fatal("user track missing user audio")
	}
	if assistantPCM[assistantOffset] != 0x22 || assistantPCM[assistantOffset+3] != 0x22 {
		t.Fatal("assistant track missing assistant audio at timestamp")
	}

	assistantStereoOffset := assistantOffset / linear16BytesPerSample * linear16BytesPerSample * 2
	if conversationPCM[0] != 0x11 || conversationPCM[1] != 0x11 {
		t.Fatalf("conversation left channel start = % x, want user audio", conversationPCM[0:2])
	}
	if conversationPCM[2] != 0x00 || conversationPCM[3] != 0x00 {
		t.Fatalf("conversation right channel start = % x, want silence", conversationPCM[2:4])
	}
	if conversationPCM[assistantStereoOffset] != 0x00 || conversationPCM[assistantStereoOffset+1] != 0x00 {
		t.Fatalf("conversation left channel at assistant offset = % x, want silence", conversationPCM[assistantStereoOffset:assistantStereoOffset+2])
	}
	if conversationPCM[assistantStereoOffset+2] != 0x22 || conversationPCM[assistantStereoOffset+3] != 0x22 {
		t.Fatalf("conversation right channel at assistant offset = % x, want assistant audio", conversationPCM[assistantStereoOffset+2:assistantStereoOffset+4])
	}
}

func TestPersistUsesRecordedTimelineLength(t *testing.T) {
	rec := newTestRecorder(t)
	ctx := context.Background()
	mediaStart := testMediaTimestamp()
	var recordingAudio internal_type.ConversationRecordingAudio
	rec.onPacket = func(_ context.Context, packets ...internal_type.Packet) error {
		recordingAudio = packets[0].(internal_type.ConversationRecordingCompletedPacket).Audio
		return nil
	}

	rec.Execute(ctx, internal_type.RecordUserAudioPacket{Audio: pcm(0x11, 100), Timestamp: mediaStart})
	rec.Execute(ctx, internal_type.RecordAssistantAudioPacket{Audio: pcm(0x22, 200), Timestamp: mediaStart})

	if err := rec.Close(ctx); err != nil {
		t.Fatalf("Persist error: %v", err)
	}
	timelineBytes := 200
	if len(wavPCMData(recordingAudio.UserAudio)) != timelineBytes {
		t.Fatalf("user PCM length = %d, want %d", len(wavPCMData(recordingAudio.UserAudio)), timelineBytes)
	}
	if len(wavPCMData(recordingAudio.AssistantAudio)) != timelineBytes {
		t.Fatalf("assistant PCM length = %d, want %d", len(wavPCMData(recordingAudio.AssistantAudio)), timelineBytes)
	}
}

func TestPersistEmptyReturnsError(t *testing.T) {
	rec := newTestRecorder(t)
	if err := rec.Close(context.Background()); err == nil {
		t.Fatal("expected error for empty recorder")
	}
}

func TestPushCopiesData(t *testing.T) {
	rec := newTestRecorder(t)
	data := pcm(0xFF, 100)

	err := rec.Execute(context.Background(), internal_type.RecordUserAudioPacket{
		Audio:     data,
		Timestamp: testMediaTimestamp(),
	})
	if err != nil {
		t.Fatalf("Record error: %v", err)
	}
	data[0] = 0x00

	if rec.timelineChunks[0].PCM16[0] != 0xFF {
		t.Fatal("recorder must copy caller audio buffers")
	}
}

func TestPersistProducesValidWAVHeader(t *testing.T) {
	rec := newTestRecorder(t)
	var recordingAudio internal_type.ConversationRecordingAudio
	rec.onPacket = func(_ context.Context, packets ...internal_type.Packet) error {
		recordingAudio = packets[0].(internal_type.ConversationRecordingCompletedPacket).Audio
		return nil
	}
	rec.Execute(context.Background(), internal_type.RecordUserAudioPacket{
		Audio:     pcm(0x01, 320),
		Timestamp: testMediaTimestamp(),
	})

	if err := rec.Close(context.Background()); err != nil {
		t.Fatalf("Persist error: %v", err)
	}
	for name, wav := range map[string][]byte{"user": recordingAudio.UserAudio, "assistant": recordingAudio.AssistantAudio} {
		if len(wav) < wavHeaderSize {
			t.Fatalf("%s WAV too short", name)
		}
		if string(wav[0:4]) != "RIFF" || string(wav[8:12]) != "WAVE" {
			t.Fatalf("%s WAV missing RIFF/WAVE header", name)
		}
		if sampleRate := binary.LittleEndian.Uint32(wav[24:28]); sampleRate != recorderAudioConfig.SampleRate {
			t.Fatalf("%s sample rate = %d, want %d", name, sampleRate, recorderAudioConfig.SampleRate)
		}
	}
}

func TestCloseEmitsConversationRecordingCompletedPacket(t *testing.T) {
	var completedPacket internal_type.ConversationRecordingCompletedPacket
	rec := &conversationRecordingExecutor{
		contextID: "ctx-recording",
		onPacket: func(_ context.Context, packets ...internal_type.Packet) error {
			if len(packets) != 1 {
				t.Fatalf("expected 1 packet, got %d", len(packets))
			}
			var ok bool
			completedPacket, ok = packets[0].(internal_type.ConversationRecordingCompletedPacket)
			if !ok {
				t.Fatalf("expected ConversationRecordingCompletedPacket, got %T", packets[0])
			}
			return nil
		},
	}

	err := rec.Execute(context.Background(), internal_type.RecordUserAudioPacket{
		Audio:     pcm(0x01, 320),
		Timestamp: testMediaTimestamp(),
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if err := rec.Close(context.Background()); err != nil {
		t.Fatalf("Close error: %v", err)
	}

	if completedPacket.ContextID != "ctx-recording" {
		t.Fatalf("context id = %q, want ctx-recording", completedPacket.ContextID)
	}
	if len(completedPacket.Audio.UserAudio) == 0 {
		t.Fatal("expected user audio in completed recording")
	}
	if len(completedPacket.Audio.AssistantAudio) == 0 {
		t.Fatal("expected assistant audio WAV in completed recording")
	}
	if len(completedPacket.Audio.MixedAudio) == 0 {
		t.Fatal("expected mixed audio in completed recording")
	}
}

func TestCloseEmitsCompletedPacketOnce(t *testing.T) {
	emittedPackets := 0
	rec := &conversationRecordingExecutor{
		contextID: "ctx-recording",
		onPacket: func(_ context.Context, packets ...internal_type.Packet) error {
			emittedPackets += len(packets)
			return nil
		},
	}

	if err := rec.Execute(context.Background(), internal_type.RecordUserAudioPacket{
		Audio:     pcm(0x01, 320),
		Timestamp: testMediaTimestamp(),
	}); err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if err := rec.Close(context.Background()); err != nil {
		t.Fatalf("first Close error: %v", err)
	}
	if err := rec.Close(context.Background()); err != nil {
		t.Fatalf("second Close error: %v", err)
	}
	if emittedPackets != 1 {
		t.Fatalf("emitted packets = %d, want 1", emittedPackets)
	}
}
