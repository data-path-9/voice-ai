// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_sip_telephony

import (
	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/zaf/g711"
)

type audioAdapter struct {
	resampler internal_type.AudioResampler
	codecFn   func() *sip_infra.Codec
}

func newAudioAdapter(resampler internal_type.AudioResampler, codecFn func() *sip_infra.Codec) *audioAdapter {
	return &audioAdapter{resampler: resampler, codecFn: codecFn}
}

func (a *audioAdapter) FrameSize() int    { return mulawFrameSize }
func (a *audioAdapter) SilenceByte() byte { return 0xFF }

func (a *audioAdapter) ConvertOutput(audio []byte) ([]byte, error) {
	outData, err := a.resampler.Resample(audio, rapida16kConfig, mulaw8kConfig)
	if err != nil {
		return nil, err
	}
	codec := a.codecFn()
	if codec != nil && codec.Name == "PCMA" {
		outData = internal_audio.UlawToAlaw(outData)
	}
	return outData, nil
}

func (a *audioAdapter) MixAmbient(frame []byte, mixer internal_ambient.Mixer) []byte {
	if mixer == nil {
		return frame
	}
	codec := a.codecFn()
	if codec == nil {
		codec = &sip_infra.CodecPCMU
	}
	var primaryPCM []byte
	if len(frame) > 0 {
		switch codec.Name {
		case "PCMA":
			primaryPCM = g711.DecodeAlaw(frame)
		default:
			primaryPCM = g711.DecodeUlaw(frame)
		}
	}
	mixedPCM, err := mixer.Mix(primaryPCM)
	if err != nil || len(mixedPCM) == 0 {
		return frame
	}
	switch codec.Name {
	case "PCMA":
		return g711.EncodeAlaw(mixedPCM)
	default:
		return g711.EncodeUlaw(mixedPCM)
	}
}
