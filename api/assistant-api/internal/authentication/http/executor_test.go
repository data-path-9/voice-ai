package internal_authentication_http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	"github.com/stretchr/testify/require"
)

type testCallback struct{}

func (testCallback) OnPacket(context.Context, ...internal_type.Packet) error {
	return nil
}

func testConfiguration(serverURL string, failBehavior string) *internal_assistant_entity.AssistantConfiguration {
	options := []*internal_assistant_entity.AssistantConfigurationOption{
		{Metadata: gorm_model.Metadata{Key: OptionHTTPURLKey, Value: serverURL}},
		{Metadata: gorm_model.Metadata{Key: OptionHTTPBodyKey, Value: `{"token":"token"}`}},
	}
	if failBehavior != "" {
		options = append(options, &internal_assistant_entity.AssistantConfigurationOption{
			Metadata: gorm_model.Metadata{Key: "fail_behavior", Value: failBehavior},
		})
	}
	return &internal_assistant_entity.AssistantConfiguration{
		Provider: "http",
		Options:  options,
	}
}

func TestExecute_UnauthenticatedBlocksByDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthenticated"}`))
	}))
	defer server.Close()

	executor, err := New(
		WithConfiguration(testConfiguration(server.URL, "")),
		WithCallback(testCallback{}),
	)
	require.NoError(t, err)

	output, err := executor.Execute(context.Background(), internal_type.AuthenticationInput{
		ContextID: "ctx-auth-block",
	})

	require.Nil(t, output)
	require.EqualError(t, err, "authentication: unauthenticated")
}

func TestExecute_UnauthenticatedReturnsOutputWhenDoNothing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthenticated"}`))
	}))
	defer server.Close()

	executor, err := New(
		WithConfiguration(testConfiguration(server.URL, "DO_NOTHING")),
		WithCallback(testCallback{}),
	)
	require.NoError(t, err)

	output, err := executor.Execute(context.Background(), internal_type.AuthenticationInput{
		ContextID: "ctx-auth-do-nothing",
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	require.False(t, output.Authenticated)
}

func TestExecute_TransportErrorReturnsNilOutput(t *testing.T) {
	executor, err := New(
		WithConfiguration(testConfiguration("http://127.0.0.1:1", "DO_NOTHING")),
		WithCallback(testCallback{}),
	)
	require.NoError(t, err)

	output, err := executor.Execute(context.Background(), internal_type.AuthenticationInput{
		ContextID: "ctx-auth-error",
	})

	require.Nil(t, output)
	require.Error(t, err)
}
