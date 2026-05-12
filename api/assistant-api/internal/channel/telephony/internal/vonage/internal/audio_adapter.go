// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_vonage

import internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"

type audioAdapter struct {
	frameSize int
}

func newAudioAdapter(frameSize int) *audioAdapter {
	return &audioAdapter{frameSize: frameSize}
}

func (a *audioAdapter) FrameSize() int    { return a.frameSize }
func (a *audioAdapter) SilenceByte() byte { return 0x00 }

func (a *audioAdapter) ConvertOutput(audio []byte) ([]byte, error) {
	return audio, nil
}

func (a *audioAdapter) MixAmbient(frame []byte, mixer internal_ambient.Mixer) []byte {
	if mixer == nil {
		return frame
	}
	mixed, err := mixer.Mix(frame)
	if err != nil {
		return frame
	}
	return mixed
}
