package adapter_internal

import (
	"testing"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTTSResampler struct {
	calls int
	out   []byte
	err   error
	src   *protos.AudioConfig
	dst   *protos.AudioConfig
}

func (m *mockTTSResampler) Resample(data []byte, source, target *protos.AudioConfig) ([]byte, error) {
	m.calls++
	m.src = source
	m.dst = target
	if m.err != nil {
		return nil, m.err
	}
	if m.out != nil {
		return m.out, nil
	}
	return data, nil
}

func TestResolveTTSAudioSourceConfig_DefaultsToRapidaInternal(t *testing.T) {
	cfg := resolveTTSAudioSourceConfig(utils.Option{})
	require.NotNil(t, cfg)
	assert.Equal(t, uint32(16000), cfg.GetSampleRate())
	assert.Equal(t, protos.AudioConfig_LINEAR16, cfg.GetAudioFormat())
	assert.Equal(t, uint32(1), cfg.GetChannels())
}

func TestResolveTTSAudioSourceConfig_UsesSpeakAudioOptions(t *testing.T) {
	cfg := resolveTTSAudioSourceConfig(utils.Option{
		"speak.audio.sample_rate": float64(8000),
		"speak.audio.encoding":    "mulaw",
	})
	require.NotNil(t, cfg)
	assert.Equal(t, uint32(8000), cfg.GetSampleRate())
	assert.Equal(t, protos.AudioConfig_MuLaw8, cfg.GetAudioFormat())
	assert.Equal(t, uint32(1), cfg.GetChannels())
}

func TestResampleTTSAudioPackets_ResamplesOnlyTTSAudioPackets(t *testing.T) {
	source := &protos.AudioConfig{SampleRate: 8000, AudioFormat: protos.AudioConfig_MuLaw8, Channels: 1}
	target := &protos.AudioConfig{SampleRate: 16000, AudioFormat: protos.AudioConfig_LINEAR16, Channels: 1}
	resampler := &mockTTSResampler{out: []byte{0x01, 0x02, 0x03, 0x04}}

	out, err := resampleTTSAudioPackets(
		[]internal_type.Packet{
			internal_type.TextToSpeechAudioPacket{ContextID: "ctx-1", AudioChunk: []byte{0xFF, 0xF0}},
			internal_type.TextToSpeechEndPacket{ContextID: "ctx-1"},
		},
		resampler,
		source,
		target,
	)
	require.NoError(t, err)
	require.Len(t, out, 2)
	require.Equal(t, 1, resampler.calls)
	assert.Equal(t, source, resampler.src)
	assert.Equal(t, target, resampler.dst)

	audioPkt, ok := out[0].(internal_type.TextToSpeechAudioPacket)
	require.True(t, ok)
	assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, audioPkt.AudioChunk)
	_, isEnd := out[1].(internal_type.TextToSpeechEndPacket)
	assert.True(t, isEnd)
}

func TestResampleTTSAudioPackets_NoOpWhenConfigMatches(t *testing.T) {
	same := &protos.AudioConfig{SampleRate: 16000, AudioFormat: protos.AudioConfig_LINEAR16, Channels: 1}
	resampler := &mockTTSResampler{out: []byte{0xAA, 0xBB}}

	out, err := resampleTTSAudioPackets(
		[]internal_type.Packet{
			internal_type.TextToSpeechAudioPacket{ContextID: "ctx-1", AudioChunk: []byte{0x10, 0x11}},
		},
		resampler,
		same,
		same,
	)
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, 0, resampler.calls)
	audioPkt, ok := out[0].(internal_type.TextToSpeechAudioPacket)
	require.True(t, ok)
	assert.Equal(t, []byte{0x10, 0x11}, audioPkt.AudioChunk)
}
