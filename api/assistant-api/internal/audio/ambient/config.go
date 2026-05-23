// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package ambient

import (
	"strings"

	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

const (
	ProfileNone       = "none"
	ProfileOffice     = "office"
	ProfileCafe       = "cafe"
	ProfileCalmStudio = "calm_studio"

	OptionAmbient       = "speaker.ambient"
	OptionAmbientVolume = "speaker.ambient_volume"
)

type Config struct {
	Profile string
	Volume  int
	Enabled bool
}

func NewConfig(profile string, volume int) Config {
	p := normalizeProfile(profile)
	v := clampVolume(volume)
	return Config{
		Profile: p,
		Volume:  v,
		Enabled: p != ProfileNone && v > 0,
	}
}

func ParseFromInitialization(init *protos.ConversationInitialization) (Config, bool) {
	if init == nil || len(init.GetOptions()) == 0 {
		return Config{}, false
	}
	options, err := utils.AnyMapToInterfaceMap(init.GetOptions())
	if err != nil {
		return Config{}, false
	}
	return ParseFromOptions(utils.Option(options))
}

func ParseFromOptions(opts utils.Option) (Config, bool) {
	if opts == nil {
		return Config{}, false
	}
	_, hasAmbient := opts[OptionAmbient]
	_, hasVolume := opts[OptionAmbientVolume]
	if !hasAmbient && !hasVolume {
		return Config{}, false
	}

	ambient, _ := opts.GetString(OptionAmbient)
	volume := 18
	if n, err := opts.GetUint64(OptionAmbientVolume); err == nil {
		volume = int(n)
	}
	return NewConfig(ambient, volume), true
}

func normalizeProfile(v string) string {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case "", "off", "disabled", ProfileNone:
		return ProfileNone
	case ProfileOffice:
		return ProfileOffice
	case ProfileCafe:
		return ProfileCafe
	case ProfileCalmStudio:
		return ProfileCalmStudio
	default:
		return ProfileNone
	}
}

func clampVolume(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}
