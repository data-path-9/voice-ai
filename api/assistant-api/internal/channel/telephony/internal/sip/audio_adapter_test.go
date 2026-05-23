package internal_sip_telephony

import (
	"errors"
	"testing"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/rapidaai/protos"
)

type sipFakeResampler struct {
	out []byte
	err error
}

func (r *sipFakeResampler) Resample(_ []byte, _, _ *protos.AudioConfig) ([]byte, error) {
	if r.err != nil {
		return nil, r.err
	}
	return append([]byte(nil), r.out...), nil
}

type sipFakeMixer struct {
	err error
}

func (m *sipFakeMixer) Configure(cfg internal_ambient.Config) error { return nil }
func (m *sipFakeMixer) Mix(primary []byte) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return primary, nil
}
func (m *sipFakeMixer) Reset()                                 {}
func (m *sipFakeMixer) CurrentConfig() internal_ambient.Config { return internal_ambient.Config{} }

func TestAudioAdapter_ConvertOutput_PCMAConverts(t *testing.T) {
	codec := sip_infra.CodecPCMA
	a := newAudioAdapter(&sipFakeResampler{out: []byte{0xFF, 0x7F}}, func() *sip_infra.Codec { return &codec })
	got, err := a.ConvertOutput([]byte{1, 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := internal_audio.UlawToAlaw([]byte{0xFF, 0x7F})
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("unexpected PCMA conversion: got=%v want=%v", got, want)
	}
}

func TestAudioAdapter_MixAmbient_NilMixerPassthrough(t *testing.T) {
	a := newAudioAdapter(&sipFakeResampler{}, func() *sip_infra.Codec { return &sip_infra.CodecPCMU })
	in := []byte{1, 2, 3}
	out := a.MixAmbient(in, nil)
	if len(out) != len(in) {
		t.Fatalf("expected passthrough")
	}
}

func TestAudioAdapter_MixAmbient_ErrorFallback(t *testing.T) {
	a := newAudioAdapter(&sipFakeResampler{}, func() *sip_infra.Codec { return &sip_infra.CodecPCMU })
	in := []byte{1, 2, 3}
	out := a.MixAmbient(in, &sipFakeMixer{err: errors.New("mix")})
	if len(out) != len(in) {
		t.Fatalf("expected fallback on error")
	}
}
