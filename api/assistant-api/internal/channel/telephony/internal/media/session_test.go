package internal_telephony_media

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type fakeEngine struct {
	onInput          func([]byte)
	inputCount       atomic.Int32
	outputCount      atomic.Int32
	completeCount    atomic.Int32
	clearCount       atomic.Int32
	runCount         atomic.Int32
	configureCount   atomic.Int32
	processOutputErr error
}

func (f *fakeEngine) SetInputAudioCallback(callback func(audio []byte)) { f.onInput = callback }
func (f *fakeEngine) ProcessInputAudio(audio []byte) error {
	f.inputCount.Add(1)
	if f.onInput != nil {
		f.onInput(audio)
	}
	return nil
}
func (f *fakeEngine) ProcessOutputAudio(audio []byte) error {
	f.outputCount.Add(1)
	return f.processOutputErr
}
func (f *fakeEngine) Complete()          { f.completeCount.Add(1) }
func (f *fakeEngine) ClearOutputBuffer() { f.clearCount.Add(1) }
func (f *fakeEngine) ConfigureAmbient(cfg internal_ambient.Config) error {
	f.configureCount.Add(1)
	return nil
}
func (f *fakeEngine) RunOutputSender(ctx context.Context) {
	f.runCount.Add(1)
	<-ctx.Done()
}

func TestMediaSession_StartAndShutdown_Idempotent(t *testing.T) {
	engine := &fakeEngine{}
	s := NewMediaSession(context.Background(), nil, engine, nil)
	s.Start()
	s.Start()
	time.Sleep(20 * time.Millisecond)
	if engine.runCount.Load() != 1 {
		t.Fatalf("runCount=%d want=1", engine.runCount.Load())
	}
	s.Shutdown()
	s.Shutdown()
}

func TestMediaSession_HandleAssistantAudio(t *testing.T) {
	engine := &fakeEngine{}
	s := NewMediaSession(context.Background(), nil, engine, nil)
	if err := s.HandleAssistantAudio([]byte{1, 2}, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if engine.outputCount.Load() != 1 {
		t.Fatalf("outputCount=%d want=1", engine.outputCount.Load())
	}
	if engine.completeCount.Load() != 1 {
		t.Fatalf("completeCount=%d want=1", engine.completeCount.Load())
	}
}

func TestMediaSession_HandleAssistantAudio_PropagatesError(t *testing.T) {
	engine := &fakeEngine{processOutputErr: errors.New("boom")}
	s := NewMediaSession(context.Background(), nil, engine, nil)
	if err := s.HandleAssistantAudio([]byte{1}, false); err == nil {
		t.Fatal("expected error")
	}
}

func TestMediaSession_HandleInterrupt_ClearsAndSends(t *testing.T) {
	engine := &fakeEngine{}
	var clearCount atomic.Int32
	s := NewMediaSession(context.Background(), nil, engine, func() error {
		clearCount.Add(1)
		return nil
	})
	s.HandleInterrupt()
	if engine.clearCount.Load() != 1 {
		t.Fatalf("clearCount=%d want=1", engine.clearCount.Load())
	}
	if clearCount.Load() != 1 {
		t.Fatalf("sendClear=%d want=1", clearCount.Load())
	}
}

func TestMediaSession_HandleProviderAudio_UsesInputSink(t *testing.T) {
	engine := &fakeEngine{}
	s := NewMediaSession(context.Background(), nil, engine, nil)
	got := make(chan []byte, 1)
	s.SetInputSink(func(audio []byte) { got <- append([]byte(nil), audio...) })
	if err := s.HandleProviderAudio([]byte{7, 8}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	select {
	case audio := <-got:
		if len(audio) != 2 || audio[0] != 7 {
			t.Fatalf("unexpected audio: %v", audio)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting input sink")
	}
}

func TestMediaSession_HandleInitialization_ParsesAmbient(t *testing.T) {
	engine := &fakeEngine{}
	s := NewMediaSession(context.Background(), nil, engine, nil)
	meta, err := utils.InterfaceMapToAnyMap(map[string]interface{}{
		"speaker.ambient":        "cafe",
		"speaker.ambient_volume": 40,
	})
	if err != nil {
		t.Fatalf("metadata conversion failed: %v", err)
	}
	s.HandleInitialization(&protos.ConversationInitialization{
		Options: meta,
	})
	if engine.configureCount.Load() != 1 {
		t.Fatalf("configureCount=%d want=1", engine.configureCount.Load())
	}
}
