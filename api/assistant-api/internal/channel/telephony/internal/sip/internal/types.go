// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_sip

import (
	"time"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
)

const (
	Provider               = "sip"
	DefaultOutboundSIPPort = 5060
	DefaultRingtone        = "ringtone_us"

	AudioChannelSize = 100
	ChunkDuration    = 20 * time.Millisecond
	MulawFrameSize   = 160
	MulawSilenceByte = 0xFF
)

var (
	Rapida16kConfig = internal_audio.NewLinear16khzMonoAudioConfig()
	Mulaw8kConfig   = internal_audio.NewMulaw8khzMonoAudioConfig()
	Linear8kConfig  = internal_audio.NewLinear8khzMonoAudioConfig()
)

type AudioProcessorConfig struct {
	RTPHandler *sip_infra.RTPHandler
	Resampler  internal_type.AudioResampler
	PushInput  func(internal_type.Stream)
	Ringtone   string
	Ambient    *internal_ambient.Config
}
