// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package ambient

import (
	"encoding/binary"
	"fmt"
	"math"
	"sync"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type Mixer interface {
	Configure(cfg Config) error
	Mix(primary []byte) ([]byte, error)
	Reset()
	CurrentConfig() Config
}

type MixerSpec struct {
	Logger            commons.Logger
	Resampler         internal_type.AudioResampler
	TargetAudioConfig *protos.AudioConfig
	FrameBytes        int
}

type LoopMixer struct {
	mu   sync.Mutex
	spec MixerSpec
	cfg  Config

	ambientPCM    []byte
	ambientOffset int
}

func NewLoopMixer(spec MixerSpec) (*LoopMixer, error) {
	if spec.TargetAudioConfig == nil {
		return nil, fmt.Errorf("target audio config is required")
	}
	if spec.TargetAudioConfig.GetAudioFormat() != protos.AudioConfig_LINEAR16 {
		return nil, fmt.Errorf("ambient loop mixer currently supports LINEAR16 target only")
	}
	if spec.FrameBytes <= 0 {
		spec.FrameBytes = internal_audio.BytesPerMs(spec.TargetAudioConfig) * 20
	}
	if spec.FrameBytes <= 0 {
		return nil, fmt.Errorf("invalid frame bytes")
	}
	return &LoopMixer{
		spec: spec,
		cfg:  NewConfig(ProfileNone, 18),
	}, nil
}

func (m *LoopMixer) Configure(cfg Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cfg = NewConfig(cfg.Profile, cfg.Volume)
	m.ambientPCM = nil
	m.ambientOffset = 0
	if !m.cfg.Enabled {
		return nil
	}

	pcm, srcCfg, err := internal_audio.LoadAmbientPCM16LE(m.cfg.Profile)
	if err != nil {
		return err
	}
	if srcCfg == nil {
		srcCfg = internal_audio.WEBRTC_AUDIO_CONFIG
	}

	if !sameAudioConfig(srcCfg, m.spec.TargetAudioConfig) {
		if m.spec.Resampler == nil {
			return fmt.Errorf("resampler is required to convert ambient asset for target format")
		}
		resampled, rerr := m.spec.Resampler.Resample(pcm, srcCfg, m.spec.TargetAudioConfig)
		if rerr != nil {
			return fmt.Errorf("resample ambient: %w", rerr)
		}
		pcm = resampled
	}

	if len(pcm) < m.spec.FrameBytes {
		return fmt.Errorf("ambient asset too short for frame size")
	}
	m.ambientPCM = pcm
	return nil
}

func (m *LoopMixer) Mix(primary []byte) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.cfg.Enabled || len(m.ambientPCM) == 0 {
		return primary, nil
	}

	targetLen := len(primary)
	if targetLen == 0 {
		targetLen = m.spec.FrameBytes
	}
	ambientFrame := m.nextFrame(targetLen)
	if len(ambientFrame) == 0 {
		return primary, nil
	}

	if len(primary) == 0 {
		out := make([]byte, len(ambientFrame))
		scaleAmbientLinear16(out, ambientFrame, m.cfg.Volume)
		return out, nil
	}

	frameLen := len(primary)
	if len(ambientFrame) < frameLen {
		frameLen = len(ambientFrame)
	}
	out := make([]byte, frameLen)
	for i := 0; i+1 < frameLen; i += 2 {
		p := int32(int16(binary.LittleEndian.Uint16(primary[i : i+2])))
		a := int32(int16(binary.LittleEndian.Uint16(ambientFrame[i : i+2])))
		a = (a * int32(m.cfg.Volume)) / 100
		mixed := p + a
		if mixed > math.MaxInt16 {
			mixed = math.MaxInt16
		} else if mixed < math.MinInt16 {
			mixed = math.MinInt16
		}
		binary.LittleEndian.PutUint16(out[i:i+2], uint16(int16(mixed)))
	}
	return out, nil
}

func (m *LoopMixer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ambientOffset = 0
}

func (m *LoopMixer) CurrentConfig() Config {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cfg
}

func (m *LoopMixer) nextFrame(n int) []byte {
	if n <= 0 || len(m.ambientPCM) == 0 {
		return nil
	}
	frame := make([]byte, n)
	remain := len(m.ambientPCM) - m.ambientOffset
	if remain >= n {
		copy(frame, m.ambientPCM[m.ambientOffset:m.ambientOffset+n])
		m.ambientOffset += n
		if m.ambientOffset >= len(m.ambientPCM) {
			m.ambientOffset = 0
		}
		return frame
	}
	copy(frame, m.ambientPCM[m.ambientOffset:])
	copy(frame[remain:], m.ambientPCM[:n-remain])
	m.ambientOffset = n - remain
	if m.ambientOffset >= len(m.ambientPCM) {
		m.ambientOffset = 0
	}
	return frame
}

func scaleAmbientLinear16(dst, src []byte, volume int) {
	frameLen := len(dst)
	if len(src) < frameLen {
		frameLen = len(src)
	}
	for i := 0; i+1 < frameLen; i += 2 {
		v := int32(int16(binary.LittleEndian.Uint16(src[i : i+2])))
		v = (v * int32(volume)) / 100
		if v > math.MaxInt16 {
			v = math.MaxInt16
		} else if v < math.MinInt16 {
			v = math.MinInt16
		}
		binary.LittleEndian.PutUint16(dst[i:i+2], uint16(int16(v)))
	}
}

func sameAudioConfig(a, b *protos.AudioConfig) bool {
	if a == nil || b == nil {
		return false
	}
	return a.GetSampleRate() == b.GetSampleRate() &&
		a.GetChannels() == b.GetChannels() &&
		a.GetAudioFormat() == b.GetAudioFormat()
}
