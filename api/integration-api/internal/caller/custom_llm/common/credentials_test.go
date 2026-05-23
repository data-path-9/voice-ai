package internal_custom_llm_common

import (
	"testing"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func testLogger() commons.Logger {
	logger, _ := commons.NewApplicationLogger()
	return logger
}

func testCredential(
	t *testing.T,
	values map[string]interface{},
) *protos.Credential {
	t.Helper()
	pb, err := structpb.NewStruct(values)
	require.NoError(t, err)
	return &protos.Credential{Value: pb}
}

func TestParseClientConfig_DefaultCompatibility(t *testing.T) {
	cfg, err := ParseClientConfig(testLogger(), testCredential(t, map[string]interface{}{
		CredentialKeyBaseURLSnake: "http://localhost:8000/v1",
	}))
	require.NoError(t, err)
	assert.Equal(t, CompatibilityOpenAIChatCompletions, cfg.Compatibility)
}

func TestParseClientConfig_SupportsCamelCaseKeys(t *testing.T) {
	cfg, err := ParseClientConfig(testLogger(), testCredential(t, map[string]interface{}{
		CredentialKeyAPICompatibilityCamel: "openai_responses",
		CredentialKeyBaseURLCamel:          "http://localhost:8000/v1",
	}))
	require.NoError(t, err)
	assert.Equal(t, CompatibilityOpenAIResponses, cfg.Compatibility)
	assert.Equal(t, "http://localhost:8000/v1", cfg.BaseURL)
}

func TestParseClientConfig_ParsesHeaders(t *testing.T) {
	cfg, err := ParseClientConfig(testLogger(), testCredential(t, map[string]interface{}{
		CredentialKeyAPICompatibilitySnake: "openai_chat_completions",
		CredentialKeyBaseURLSnake:          "http://localhost:8000/v1",
		CredentialKeyHeaders:               `{"Authorization":"Bearer token","X-Test":"ok"}`,
	}))
	require.NoError(t, err)
	assert.Equal(t, "Bearer token", cfg.Headers["Authorization"])
	assert.Equal(t, "ok", cfg.Headers["X-Test"])
}

func TestParseClientConfig_ValidatesBaseURLAndCompatibilityType(t *testing.T) {
	_, err := ParseClientConfig(testLogger(), testCredential(t, map[string]interface{}{
		CredentialKeyAPICompatibilitySnake: 123,
		CredentialKeyBaseURLSnake:          "http://localhost:8000/v1",
	}))
	require.Error(t, err)

	_, err = ParseClientConfig(testLogger(), testCredential(t, map[string]interface{}{
		CredentialKeyAPICompatibilitySnake: "openai_chat_completions",
	}))
	require.Error(t, err)
}

func TestResolveCompatibility_DefaultAndCredentialValue(t *testing.T) {
	compatibility, err := ResolveCompatibility(nil)
	require.NoError(t, err)
	assert.Equal(t, CompatibilityOpenAIChatCompletions, compatibility)

	compatibility, err = ResolveCompatibility(testCredential(t, map[string]interface{}{
		CredentialKeyAPICompatibilitySnake: string(CompatibilityOpenAIResponses),
	}))
	require.NoError(t, err)
	assert.Equal(t, CompatibilityOpenAIResponses, compatibility)
}
