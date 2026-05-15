// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_openai_chat_complete

import (
	"context"
	"errors"
	"net/http"
	"testing"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func findMetric(t *testing.T, metrics []*protos.Metric, name string) *protos.Metric {
	t.Helper()
	for _, metric := range metrics {
		if metric.GetName() == name {
			return metric
		}
	}
	return nil
}

func newTestLogger() commons.Logger {
	lgr, _ := commons.NewApplicationLogger()
	return lgr
}

func credentialWithKey(t *testing.T) *protos.Credential {
	t.Helper()
	value, err := structpb.NewStruct(map[string]interface{}{"key": "sk-test"})
	require.NoError(t, err)
	return &protos.Credential{Value: value}
}

func TestNew_RejectsMissingCredential(t *testing.T) {
	caller, err := NewChat(newTestLogger(), nil)
	require.Error(t, err)
	assert.Nil(t, caller)
}

func TestNew_AcceptsValidCredential(t *testing.T) {
	caller, err := NewChat(newTestLogger(), credentialWithKey(t))
	require.NoError(t, err)
	assert.NotNil(t, caller)
}

func TestNewStream_RejectsMissingCredential(t *testing.T) {
	caller, err := NewStream(newTestLogger(), nil)
	require.Error(t, err)
	assert.Nil(t, caller)
}

func TestStream_ConnectAndCloseLifecycle(t *testing.T) {
	caller, err := NewStream(newTestLogger(), credentialWithKey(t))
	require.NoError(t, err)

	s, ok := caller.(*streamCaller)
	require.True(t, ok)

	err = s.Connect(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, s.client)
	firstClient := s.client
	require.NotNil(t, s.httpClient)

	err = s.Connect(context.Background(), nil)
	require.NoError(t, err)
	assert.Same(t, firstClient, s.client)

	err = s.Close(context.Background())
	require.NoError(t, err)
	assert.Nil(t, s.client)
	assert.Nil(t, s.httpClient)
}

func TestStream_ChatConnectFailureReportsFailureMetrics(t *testing.T) {
	caller, err := NewStream(newTestLogger(), credentialWithKey(t))
	require.NoError(t, err)

	s, ok := caller.(*streamCaller)
	require.True(t, ok)
	s.credential = nil

	var postHookPayload map[string]interface{}
	var postHookMetrics []*protos.Metric
	var onErrorRequestID string
	var onErrorErr error
	onMetricsCalled := false

	options := &internal_callers.ChatStreamCompletionOptions{
		AIOptions: internal_callers.AIOptions{
			RequestId: 77,
			PostHook: func(rst map[string]interface{}, metrics []*protos.Metric) {
				postHookPayload = rst
				postHookMetrics = metrics
			},
		},
		Request: &protos.StreamChatInput{RequestId: "req-connect-failure"},
	}

	err = s.Chat(
		context.Background(),
		nil,
		options,
		nil,
		func(string, *protos.Message, []*protos.Metric) error {
			onMetricsCalled = true
			return nil
		},
		func(requestID string, streamErr error) {
			onErrorRequestID = requestID
			onErrorErr = streamErr
		},
	)
	require.Error(t, err)
	require.Error(t, onErrorErr)
	assert.Equal(t, "req-connect-failure", onErrorRequestID)
	assert.False(t, onMetricsCalled)
	require.NotNil(t, postHookPayload)
	require.NotNil(t, postHookPayload["error"])
	status := findMetric(t, postHookMetrics, type_enums.STATUS.String())
	require.NotNil(t, status)
	assert.Equal(t, type_enums.RECORD_FAILED.String(), status.GetValue())
}

func TestStream_ChatInitFailureReportsFailureMetrics(t *testing.T) {
	caller, err := NewStream(newTestLogger(), credentialWithKey(t))
	require.NoError(t, err)

	s, ok := caller.(*streamCaller)
	require.True(t, ok)

	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("stream init failed")
		}),
	}
	client := openai.NewClient(
		option.WithAPIKey("sk-test"),
		option.WithHTTPClient(httpClient),
	)
	s.client = &client
	s.httpClient = httpClient

	var postHookPayload map[string]interface{}
	var postHookMetrics []*protos.Metric
	var onErrorRequestID string
	var onErrorErr error
	onMetricsCalled := false

	options := &internal_callers.ChatStreamCompletionOptions{
		AIOptions: internal_callers.AIOptions{
			RequestId: 88,
			PostHook: func(rst map[string]interface{}, metrics []*protos.Metric) {
				postHookPayload = rst
				postHookMetrics = metrics
			},
		},
		Request: &protos.StreamChatInput{RequestId: "req-stream-init"},
	}
	messages := []*protos.Message{
		{
			Role: "user",
			Message: &protos.Message_User{
				User: &protos.UserMessage{Content: "hello"},
			},
		},
	}

	err = s.Chat(
		context.Background(),
		messages,
		options,
		nil,
		func(string, *protos.Message, []*protos.Metric) error {
			onMetricsCalled = true
			return nil
		},
		func(requestID string, streamErr error) {
			onErrorRequestID = requestID
			onErrorErr = streamErr
		},
	)
	require.Error(t, err)
	require.Error(t, onErrorErr)
	assert.Equal(t, "req-stream-init", onErrorRequestID)
	assert.False(t, onMetricsCalled)
	require.NotNil(t, postHookPayload)
	require.NotNil(t, postHookPayload["error"])
	status := findMetric(t, postHookMetrics, type_enums.STATUS.String())
	require.NotNil(t, status)
	assert.Equal(t, type_enums.RECORD_FAILED.String(), status.GetValue())
}
