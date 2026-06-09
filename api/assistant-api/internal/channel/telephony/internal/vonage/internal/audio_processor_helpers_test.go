package internal_vonage

import (
	"errors"
	"testing"

	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
)

type vonageFakeMixer struct {
	err error
}

func (mixer *vonageFakeMixer) Configure(_ internal_ambient.Config) error { return nil }

func (mixer *vonageFakeMixer) Mix(primary []byte) ([]byte, error) {
	if mixer.err != nil {
		return nil, mixer.err
	}
	return primary, nil
}

func (mixer *vonageFakeMixer) Reset() {}

func (mixer *vonageFakeMixer) CurrentConfig() internal_ambient.Config {
	return internal_ambient.Config{}
}

func TestAudioProcessor_ProcessAssistantAudio_Passthrough(t *testing.T) {
	audioProcessor := newTestAudioProcessor()
	input := []byte{1, 2, 3}

	if err := audioProcessor.ProcessAssistantAudio(input, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output, ok := audioProcessor.outputBuffer.Next(len(input))
	if !ok {
		t.Fatal("expected output audio")
	}
	if len(output) != len(input) {
		t.Fatalf("unexpected passthrough output")
	}
}

func TestAudioProcessor_ApplyAmbient_PassthroughOnError(t *testing.T) {
	input := []byte{1, 2}
	audioProcessor := &AudioProcessor{
		ambientMixer: &vonageFakeMixer{err: errors.New("mix")},
	}

	output := audioProcessor.applyAmbient(input)
	if len(output) != len(input) {
		t.Fatalf("expected passthrough on error")
	}
}
