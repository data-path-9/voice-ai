// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_asterisk

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	internal_channel_input "github.com/rapidaai/api/assistant-api/internal/channel/input"
	internal_channel_output "github.com/rapidaai/api/assistant-api/internal/channel/output"
	"github.com/rapidaai/protos"
)

// mockResampler is a test double that returns input unchanged (identity conversion).
type mockResampler struct{}

func (m *mockResampler) Resample(data []byte, _, _ *protos.AudioConfig) ([]byte, error) {
	out := make([]byte, len(data))
	copy(out, data)
	return out, nil
}

// newTestProcessor builds an AudioProcessor wired with a mock resampler.
func newTestProcessor(t *testing.T, silenceByte byte, frameSize int) *AudioProcessor {
	t.Helper()
	p := &AudioProcessor{
		logger:           nil, // not used in unit tests
		resampler:        &mockResampler{},
		asteriskConfig:   &protos.AudioConfig{},
		downstreamConfig: &protos.AudioConfig{},
		silenceByte:      silenceByte,
		optimalFrameSize: frameSize,
		inputBuffer:      internal_channel_input.NewBytesInputBuffer(inputBufferThreshold * 2),
		outputBuffer:     internal_channel_output.NewBytesFrameBuffer(frameSize * 8),
		outputHealth:     internal_channel_output.NewHealthStats(),
	}
	p.adapter = newAudioAdapter(p.resampler, p.downstreamConfig, p.asteriskConfig, frameSize, silenceByte)
	p.silenceChunk = p.createSilenceChunk(frameSize, silenceByte)
	return p
}

func TestNewAudioProcessor_HappyPath(t *testing.T) {
	p := newTestProcessor(t, 0x00, 320)
	if p == nil {
		t.Fatal("expected non-nil processor")
	}
	if p.GetOptimalFrameSize() != 320 {
		t.Errorf("expected frame size 320, got %d", p.GetOptimalFrameSize())
	}
	if p.GetDownstreamConfig() == nil {
		t.Fatal("expected non-nil downstream config")
	}
}

func TestSetOptimalFrameSize(t *testing.T) {
	p := newTestProcessor(t, 0xFF, 160)

	p.SetOptimalFrameSize(256)
	if p.GetOptimalFrameSize() != 256 {
		t.Errorf("expected 256, got %d", p.GetOptimalFrameSize())
	}

	// Zero should be ignored
	p.SetOptimalFrameSize(0)
	if p.GetOptimalFrameSize() != 256 {
		t.Errorf("expected 256 (unchanged), got %d", p.GetOptimalFrameSize())
	}

	// Negative should be ignored
	p.SetOptimalFrameSize(-1)
	if p.GetOptimalFrameSize() != 256 {
		t.Errorf("expected 256 (unchanged), got %d", p.GetOptimalFrameSize())
	}
}

func TestProcessInputAudio_EmptyInput(t *testing.T) {
	p := newTestProcessor(t, 0x00, 320)
	called := false
	p.SetInputAudioCallback(func(_ []byte) { called = true })

	if err := p.ProcessInputAudio(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("callback should not be called for empty input")
	}

	if err := p.ProcessInputAudio([]byte{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("callback should not be called for zero-length input")
	}
}

func TestProcessInputAudio_BufferThreshold(t *testing.T) {
	p := newTestProcessor(t, 0x00, 320)

	var received []byte
	p.SetInputAudioCallback(func(audio []byte) {
		received = append(received, audio...)
	})

	// Send data below threshold -- callback should not fire
	smallChunk := make([]byte, inputBufferThreshold-1)
	for i := range smallChunk {
		smallChunk[i] = 0x42
	}
	if err := p.ProcessInputAudio(smallChunk); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received != nil {
		t.Fatal("callback should not fire below threshold")
	}

	// Send one more byte to cross threshold
	if err := p.ProcessInputAudio([]byte{0x43}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(received) != inputBufferThreshold {
		t.Errorf("expected %d bytes, got %d", inputBufferThreshold, len(received))
	}
}

func TestClearInputBuffer(t *testing.T) {
	p := newTestProcessor(t, 0x00, 320)
	p.SetInputAudioCallback(func(_ []byte) {
		t.Error("callback should not fire after clear")
	})

	// Fill buffer near threshold
	chunk := make([]byte, inputBufferThreshold-1)
	_ = p.ProcessInputAudio(chunk)

	p.ClearInputBuffer()

	// One byte should not cross threshold now since buffer was cleared
	_ = p.ProcessInputAudio([]byte{0x01})
}

func TestProcessOutputAudio_EmptyInput(t *testing.T) {
	p := newTestProcessor(t, 0xFF, 160)
	if err := p.ProcessOutputAudio(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if chunk := p.GetNextChunk(); chunk != nil {
		t.Error("expected nil chunk for empty output buffer")
	}
}

func TestProcessOutputAudio_GetNextChunk(t *testing.T) {
	p := newTestProcessor(t, 0xFF, 160)

	// Write exactly one frame worth of data
	data := make([]byte, 160)
	for i := range data {
		data[i] = 0xAA
	}
	if err := p.ProcessOutputAudio(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	chunk := p.GetNextChunk()
	if chunk == nil {
		t.Fatal("expected non-nil chunk")
	}
	if len(chunk.Data) != 160 {
		t.Errorf("expected 160 bytes, got %d", len(chunk.Data))
	}
	if chunk.Duration != chunkDuration {
		t.Errorf("expected %v duration, got %v", chunkDuration, chunk.Duration)
	}

	// Buffer should be empty now
	if next := p.GetNextChunk(); next != nil {
		t.Error("expected nil after consuming all data")
	}
}

func TestGetNextChunk_PaddingWithSilenceOnCompleted(t *testing.T) {
	p := newTestProcessor(t, 0xFF, 160)

	// Write less than one frame
	data := make([]byte, 100)
	for i := range data {
		data[i] = 0xBB
	}
	_ = p.ProcessOutputAudio(data)

	// Partial frame should be held until completion is signalled.
	if chunk := p.GetNextChunk(); chunk != nil {
		t.Fatal("expected nil chunk for partial frame before completion")
	}

	p.Complete()
	chunk := p.GetNextChunk()
	if chunk == nil {
		t.Fatal("expected padded chunk after completion flush")
	}
	if len(chunk.Data) != 160 {
		t.Errorf("expected padded to 160, got %d", len(chunk.Data))
	}
	// Verify first 100 bytes are data
	for i := 0; i < 100; i++ {
		if chunk.Data[i] != 0xBB {
			t.Errorf("expected data byte at %d, got %x", i, chunk.Data[i])
			break
		}
	}
	// Verify remaining bytes are silence
	for i := 100; i < 160; i++ {
		if chunk.Data[i] != 0xFF {
			t.Errorf("expected silence byte 0xFF at %d, got %x", i, chunk.Data[i])
			break
		}
	}
}

func TestGetNextChunk_SLINSilencePaddingOnCompleted(t *testing.T) {
	p := newTestProcessor(t, 0x00, 320)

	data := make([]byte, 100)
	for i := range data {
		data[i] = 0xCC
	}
	_ = p.ProcessOutputAudio(data)

	if chunk := p.GetNextChunk(); chunk != nil {
		t.Fatal("expected nil chunk for partial frame before completion")
	}

	p.Complete()
	chunk := p.GetNextChunk()
	if chunk == nil {
		t.Fatal("expected padded chunk after completion flush")
	}
	// Verify padding uses SLIN silence (0x00)
	for i := 100; i < 320; i++ {
		if chunk.Data[i] != 0x00 {
			t.Errorf("expected SLIN silence 0x00 at %d, got %x", i, chunk.Data[i])
			break
		}
	}
}

func TestClearOutputBuffer(t *testing.T) {
	p := newTestProcessor(t, 0xFF, 160)
	_ = p.ProcessOutputAudio(make([]byte, 160))
	p.ClearOutputBuffer()
	if chunk := p.GetNextChunk(); chunk != nil {
		t.Error("expected nil chunk after clearing output buffer")
	}
}

func TestCreateSilenceChunk(t *testing.T) {
	p := newTestProcessor(t, 0xFF, 160)
	silence := p.silenceChunk
	if silence == nil {
		t.Fatal("expected non-nil silence chunk")
	}
	if len(silence.Data) != 160 {
		t.Errorf("expected 160 bytes, got %d", len(silence.Data))
	}
	for i, b := range silence.Data {
		if b != 0xFF {
			t.Errorf("expected 0xFF at position %d, got %x", i, b)
			break
		}
	}
}

func TestXOFF_XON(t *testing.T) {
	p := newTestProcessor(t, 0x00, 160)

	if p.IsXOFF() {
		t.Error("should start with XOFF=false")
	}

	p.SetXOFF()
	if !p.IsXOFF() {
		t.Error("expected XOFF=true after SetXOFF")
	}

	p.SetXON()
	if p.IsXOFF() {
		t.Error("expected XOFF=false after SetXON")
	}
}

func TestXOFF_ConcurrentAccess(t *testing.T) {
	p := newTestProcessor(t, 0x00, 160)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			p.SetXOFF()
		}()
		go func() {
			defer wg.Done()
			p.SetXON()
		}()
		go func() {
			defer wg.Done()
			_ = p.IsXOFF()
		}()
	}
	wg.Wait()
}

func TestRunOutputSender_RespectsContextCancellation(t *testing.T) {
	p := newTestProcessor(t, 0xFF, 160)

	var count atomic.Int32
	p.SetOutputChunkCallback(func(_ *AudioChunk) error {
		count.Add(1)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		p.RunOutputSender(ctx)
		close(done)
	}()

	// Let it run for a couple of chunks
	time.Sleep(60 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("RunOutputSender did not exit after context cancellation")
	}

	if count.Load() == 0 {
		t.Error("expected at least one chunk to be sent")
	}
}

func TestRunOutputSender_SendsRealDataOverSilence(t *testing.T) {
	p := newTestProcessor(t, 0xFF, 160)

	var chunks []*AudioChunk
	var mu sync.Mutex
	p.SetOutputChunkCallback(func(chunk *AudioChunk) error {
		mu.Lock()
		chunks = append(chunks, chunk)
		mu.Unlock()
		return nil
	})

	// Buffer one frame of real data
	data := make([]byte, 160)
	for i := range data {
		data[i] = 0xAA
	}
	_ = p.ProcessOutputAudio(data)

	ctx, cancel := context.WithCancel(context.Background())
	go p.RunOutputSender(ctx)

	time.Sleep(50 * time.Millisecond)
	cancel()

	mu.Lock()
	defer mu.Unlock()

	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	// First chunk should contain our data
	if chunks[0].Data[0] != 0xAA {
		t.Errorf("expected real data 0xAA, got %x", chunks[0].Data[0])
	}
}

func TestRunOutputSender_XOFFSuppressesOutput(t *testing.T) {
	p := newTestProcessor(t, 0xFF, 160)

	var count atomic.Int32
	p.SetOutputChunkCallback(func(_ *AudioChunk) error {
		count.Add(1)
		return nil
	})

	p.SetXOFF()

	ctx, cancel := context.WithCancel(context.Background())
	go p.RunOutputSender(ctx)

	time.Sleep(80 * time.Millisecond)
	cancel()

	if count.Load() != 0 {
		t.Errorf("expected 0 chunks sent while XOFF, got %d", count.Load())
	}
}

func TestRunOutputSender_IsIdempotent(t *testing.T) {
	p := newTestProcessor(t, 0xFF, 160)

	var inFlight atomic.Int32
	var maxInFlight atomic.Int32
	p.SetOutputChunkCallback(func(_ *AudioChunk) error {
		cur := inFlight.Add(1)
		for {
			prev := maxInFlight.Load()
			if cur <= prev || maxInFlight.CompareAndSwap(prev, cur) {
				break
			}
		}
		time.Sleep(30 * time.Millisecond)
		inFlight.Add(-1)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	done1 := make(chan struct{})
	done2 := make(chan struct{})
	go func() {
		p.RunOutputSender(ctx)
		close(done1)
	}()
	go func() {
		p.RunOutputSender(ctx)
		close(done2)
	}()

	time.Sleep(110 * time.Millisecond)
	cancel()

	select {
	case <-done1:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("first sender did not stop")
	}
	select {
	case <-done2:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("second sender did not stop")
	}

	if got := maxInFlight.Load(); got > 1 {
		t.Fatalf("expected at most one sender loop; max concurrent callbacks=%d", got)
	}
	if stats := p.OutputHealthSnapshot(); stats.Ticks == 0 {
		t.Fatal("expected output health ticks to be recorded")
	}
}
