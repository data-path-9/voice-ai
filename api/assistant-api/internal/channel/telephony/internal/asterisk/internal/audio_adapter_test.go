package internal_asterisk

import (
	"errors"
	"testing"

	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	"github.com/rapidaai/protos"
)

type asteriskFakeResampler struct {
	out []byte
	err error
}

func (r *asteriskFakeResampler) Resample(_ []byte, _, _ *protos.AudioConfig) ([]byte, error) {
	if r.err != nil {
		return nil, r.err
	}
	return append([]byte(nil), r.out...), nil
}

type asteriskFakeMixer struct {
	err error
}

func (m *asteriskFakeMixer) Configure(cfg internal_ambient.Config) error { return nil }
func (m *asteriskFakeMixer) Mix(primary []byte) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return primary, nil
}
func (m *asteriskFakeMixer) Reset()                                 {}
func (m *asteriskFakeMixer) CurrentConfig() internal_ambient.Config { return internal_ambient.Config{} }

func TestAudioAdapter_ConvertOutput_UsesResampler(t *testing.T) {
	a := newAudioAdapter(
		&asteriskFakeResampler{out: []byte{9, 9}},
		&protos.AudioConfig{}, &protos.AudioConfig{AudioFormat: protos.AudioConfig_LINEAR16}, 160, 0x00,
	)
	got, err := a.ConvertOutput([]byte{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("unexpected output: %v", got)
	}
}

func TestAudioAdapter_MixAmbient_DefaultPassthrough(t *testing.T) {
	a := newAudioAdapter(
		&asteriskFakeResampler{},
		&protos.AudioConfig{}, &protos.AudioConfig{AudioFormat: protos.AudioConfig_AudioFormat(999)}, 160, 0x00,
	)
	in := []byte{1, 2, 3}
	out := a.MixAmbient(in, &asteriskFakeMixer{err: errors.New("mix")})
	if len(out) != len(in) {
		t.Fatalf("expected passthrough for unsupported format")
	}
}
