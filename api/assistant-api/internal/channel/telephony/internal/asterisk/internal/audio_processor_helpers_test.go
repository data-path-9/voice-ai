package internal_asterisk

import (
	"errors"
	"testing"

	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	"github.com/rapidaai/protos"
)

type asteriskFakeMixer struct {
	err error
}

func (mixer *asteriskFakeMixer) Configure(_ internal_ambient.Config) error { return nil }

func (mixer *asteriskFakeMixer) Mix(primary []byte) ([]byte, error) {
	if mixer.err != nil {
		return nil, mixer.err
	}
	return primary, nil
}

func (mixer *asteriskFakeMixer) Reset() {}

func (mixer *asteriskFakeMixer) CurrentConfig() internal_ambient.Config {
	return internal_ambient.Config{}
}

func TestAudioProcessor_ConvertOutputAudio_UsesResampler(t *testing.T) {
	audioProcessor := &AudioProcessor{
		resampler:        &mockResampler{},
		downstreamConfig: &protos.AudioConfig{},
		asteriskConfig:   &protos.AudioConfig{AudioFormat: protos.AudioConfig_LINEAR16},
	}

	got, err := audioProcessor.convertOutputAudio([]byte{1, 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("unexpected output: %v", got)
	}
}

func TestAudioProcessor_ApplyAmbient_DefaultPassthrough(t *testing.T) {
	audioProcessor := &AudioProcessor{
		asteriskConfig: &protos.AudioConfig{AudioFormat: protos.AudioConfig_AudioFormat(999)},
		ambientMixer:   &asteriskFakeMixer{err: errors.New("mix")},
	}
	input := []byte{1, 2, 3}

	output := audioProcessor.applyAmbient(input)
	if len(output) != len(input) {
		t.Fatalf("expected passthrough for unsupported format")
	}
}
