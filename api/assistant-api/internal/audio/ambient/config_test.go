package ambient

import (
	"testing"

	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig_NormalizesAndClamps(t *testing.T) {
	cfg := NewConfig("  CAFE  ", 120)
	assert.Equal(t, ProfileCafe, cfg.Profile)
	assert.Equal(t, 100, cfg.Volume)
	assert.True(t, cfg.Enabled)

	disabled := NewConfig("unknown", 50)
	assert.Equal(t, ProfileNone, disabled.Profile)
	assert.False(t, disabled.Enabled)

	muted := NewConfig(ProfileOffice, -10)
	assert.Equal(t, 0, muted.Volume)
	assert.False(t, muted.Enabled)
}

func TestParseFromOptions(t *testing.T) {
	opts := utils.Option{
		OptionAmbient:       "office",
		OptionAmbientVolume: 33,
	}
	cfg, ok := ParseFromOptions(opts)
	require.True(t, ok)
	assert.Equal(t, ProfileOffice, cfg.Profile)
	assert.Equal(t, 33, cfg.Volume)
	assert.True(t, cfg.Enabled)
}

func TestParseFromOptions_Absent(t *testing.T) {
	_, ok := ParseFromOptions(utils.Option{"foo": "bar"})
	assert.False(t, ok)
}
