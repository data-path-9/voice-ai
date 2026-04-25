package internal_transformer_elevenlabs

import (
	"testing"

	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/internal/testutil"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

func newVaultCredential(m map[string]interface{}) *protos.VaultCredential {
	val, _ := structpb.NewStruct(m)
	return &protos.VaultCredential{Value: val}
}

// --- Constructor Tests ---

func TestNewElevenLabsOption_ValidCredentials(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "test-api-key"})
	opt, err := NewElevenLabsOption(testutil.NewTestLogger(), cred, utils.Option{})
	assert.NoError(t, err)
	assert.NotNil(t, opt)
	assert.Equal(t, "test-api-key", opt.GetKey())
}

func TestNewElevenLabsOption_MissingKey(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"other": "value"})
	opt, err := NewElevenLabsOption(testutil.NewTestLogger(), cred, utils.Option{})
	assert.Error(t, err)
	assert.Nil(t, opt)
	assert.Contains(t, err.Error(), "illegal vault config")
}

func TestNewElevenLabsOption_EmptyVault(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{})
	opt, err := NewElevenLabsOption(testutil.NewTestLogger(), cred, utils.Option{})
	assert.Error(t, err)
	assert.Nil(t, opt)
}

// --- Encoding Tests ---

func TestElevenLabsGetEncoding(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opt, _ := NewElevenLabsOption(testutil.NewTestLogger(), cred, utils.Option{})
	assert.Equal(t, "pcm_16000", opt.GetEncoding())
}

// --- GetTextToSpeechConnectionString Tests ---

func TestGetTextToSpeechConnectionString_Default(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opt, _ := NewElevenLabsOption(testutil.NewTestLogger(), cred, utils.Option{})
	connStr := opt.GetTextToSpeechConnectionString()

	assert.Contains(t, connStr, "wss://api.elevenlabs.io/v1/text-to-speech/")
	assert.Contains(t, connStr, ELEVENLABS_VOICE_ID)
	assert.Contains(t, connStr, "output_format=pcm_16000")
	assert.Contains(t, connStr, "enable_ssml_parsing=true")
}

func TestGetTextToSpeechConnectionString_WithVoiceOverride(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opts := utils.Option{
		"speak.voice.id": "custom-voice-id",
	}
	opt, _ := NewElevenLabsOption(testutil.NewTestLogger(), cred, opts)
	connStr := opt.GetTextToSpeechConnectionString()

	assert.Contains(t, connStr, "/custom-voice-id/multi-stream-input?")
	assert.NotContains(t, connStr, ELEVENLABS_VOICE_ID)
	assert.Contains(t, connStr, "output_format=pcm_16000")
}

func TestGetTextToSpeechConnectionString_WithLanguageAndModel(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opts := utils.Option{
		"speak.language": "fr",
		"speak.model":    "eleven_turbo_v2",
	}
	opt, _ := NewElevenLabsOption(testutil.NewTestLogger(), cred, opts)
	connStr := opt.GetTextToSpeechConnectionString()

	assert.Contains(t, connStr, "language=fr")
	assert.Contains(t, connStr, "model_id=eleven_turbo_v2")
	assert.Contains(t, connStr, "output_format=pcm_16000")
}

func TestGetTextToSpeechConnectionString_AllOptions(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opts := utils.Option{
		"speak.voice.id": "my-voice",
		"speak.language": "es",
		"speak.model":    "eleven_multilingual_v2",
	}
	opt, _ := NewElevenLabsOption(testutil.NewTestLogger(), cred, opts)
	connStr := opt.GetTextToSpeechConnectionString()

	assert.Contains(t, connStr, "/my-voice/multi-stream-input?")
	assert.Contains(t, connStr, "language=es")
	assert.Contains(t, connStr, "model_id=eleven_multilingual_v2")
	assert.Contains(t, connStr, "output_format=pcm_16000")
	assert.Contains(t, connStr, "enable_ssml_parsing=true")
}
