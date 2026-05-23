package internal_exotel

import (
	"errors"
	"testing"

	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	"github.com/rapidaai/protos"
)

type exotelFakeResampler struct {
	out []byte
	err error
}

func (r *exotelFakeResampler) Resample(_ []byte, _, _ *protos.AudioConfig) ([]byte, error) {
	if r.err != nil {
		return nil, r.err
	}
	return append([]byte(nil), r.out...), nil
}

type exotelFakeMixer struct {
	out []byte
	err error
}

func (m *exotelFakeMixer) Configure(cfg internal_ambient.Config) error { return nil }
func (m *exotelFakeMixer) Mix(primary []byte) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.out != nil {
		return append([]byte(nil), m.out...), nil
	}
	return primary, nil
}
func (m *exotelFakeMixer) Reset()                                 {}
func (m *exotelFakeMixer) CurrentConfig() internal_ambient.Config { return internal_ambient.Config{} }

func TestAudioAdapter_ConvertOutput_UsesResampler(t *testing.T) {
	a := newAudioAdapter(&exotelFakeResampler{out: []byte{7, 8}}, &protos.AudioConfig{}, &protos.AudioConfig{}, 320)
	got, err := a.ConvertOutput([]byte{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != 7 {
		t.Fatalf("unexpected output: %v", got)
	}
}

func TestAudioAdapter_MixAmbient_FallbackOnError(t *testing.T) {
	a := newAudioAdapter(&exotelFakeResampler{}, &protos.AudioConfig{}, &protos.AudioConfig{}, 320)
	in := []byte{1, 2, 3}
	out := a.MixAmbient(in, &exotelFakeMixer{err: errors.New("mix")})
	if len(out) != len(in) {
		t.Fatalf("expected passthrough on mixer error")
	}
}
