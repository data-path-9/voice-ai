package internal_twilio

import (
	"errors"
	"testing"

	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	"github.com/rapidaai/protos"
)

type twilioFakeResampler struct {
	out []byte
	err error
}

func (r *twilioFakeResampler) Resample(_ []byte, _, _ *protos.AudioConfig) ([]byte, error) {
	if r.err != nil {
		return nil, r.err
	}
	return append([]byte(nil), r.out...), nil
}

func TestAudioAdapter_ConvertOutput_UsesResampler(t *testing.T) {
	a := newAudioAdapter(&twilioFakeResampler{out: []byte{1, 2, 3}}, &protos.AudioConfig{}, &protos.AudioConfig{}, 160, 0xFF)
	got, err := a.ConvertOutput([]byte{9})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Fatalf("unexpected output: %v", got)
	}
}

func TestAudioAdapter_ConvertOutput_PropagatesError(t *testing.T) {
	a := newAudioAdapter(&twilioFakeResampler{err: errors.New("boom")}, &protos.AudioConfig{}, &protos.AudioConfig{}, 160, 0xFF)
	if _, err := a.ConvertOutput([]byte{1}); err == nil {
		t.Fatal("expected error")
	}
}

func TestAudioAdapter_MixAmbient_NilMixerPassthrough(t *testing.T) {
	a := newAudioAdapter(&twilioFakeResampler{}, &protos.AudioConfig{}, &protos.AudioConfig{}, 160, 0xFF)
	in := []byte{1, 2, 3}
	out := a.MixAmbient(in, nil)
	if len(out) != len(in) {
		t.Fatalf("expected passthrough length, got=%d want=%d", len(out), len(in))
	}
}

func TestAudioAdapter_MixAmbient_EmptyOnMixerError(t *testing.T) {
	a := newAudioAdapter(&twilioFakeResampler{}, &protos.AudioConfig{}, &protos.AudioConfig{}, 160, 0xFF)
	mixer := &twilioFakeMixer{err: errors.New("mix")}
	if out := a.MixAmbient(nil, mixer); out != nil {
		t.Fatalf("expected nil output for idle mix error, got=%v", out)
	}
}

type twilioFakeMixer struct {
	err error
}

func (m *twilioFakeMixer) Configure(cfg internal_ambient.Config) error { return nil }
func (m *twilioFakeMixer) Mix(primary []byte) ([]byte, error)          { return primary, m.err }
func (m *twilioFakeMixer) Reset()                                      {}
func (m *twilioFakeMixer) CurrentConfig() internal_ambient.Config      { return internal_ambient.Config{} }
