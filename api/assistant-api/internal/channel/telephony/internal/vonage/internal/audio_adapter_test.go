package internal_vonage

import (
	"errors"
	"testing"

	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
)

type vonageFakeMixer struct {
	err error
}

func (m *vonageFakeMixer) Configure(cfg internal_ambient.Config) error { return nil }
func (m *vonageFakeMixer) Mix(primary []byte) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return primary, nil
}
func (m *vonageFakeMixer) Reset()                                 {}
func (m *vonageFakeMixer) CurrentConfig() internal_ambient.Config { return internal_ambient.Config{} }

func TestAudioAdapter_ConvertOutput_Passthrough(t *testing.T) {
	a := newAudioAdapter(640)
	in := []byte{1, 2, 3}
	out, err := a.ConvertOutput(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != len(in) {
		t.Fatalf("unexpected passthrough output")
	}
}

func TestAudioAdapter_MixAmbient_PassthroughOnError(t *testing.T) {
	a := newAudioAdapter(640)
	in := []byte{1, 2}
	out := a.MixAmbient(in, &vonageFakeMixer{err: errors.New("mix")})
	if len(out) != len(in) {
		t.Fatalf("expected passthrough on error")
	}
}
