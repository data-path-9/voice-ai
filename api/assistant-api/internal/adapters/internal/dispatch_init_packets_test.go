package adapter_internal

import (
	"context"
	"errors"
	"testing"

	adapter_channel "github.com/rapidaai/api/assistant-api/internal/adapters/channel"
	adapter_lifecycle "github.com/rapidaai/api/assistant-api/internal/adapters/lifecycle"
	adapter_router "github.com/rapidaai/api/assistant-api/internal/adapters/router"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_conversation_entity "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializationPackets_RouteToBootstrapChannel(t *testing.T) {
	conversationInitialization := &protos.ConversationInitialization{}
	initializationError := errors.New("initialization failed")

	initializationPackets := []internal_type.Packet{
		internal_type.InitializeAssistantPacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.InitializeConversationPacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.InitializeSessionRuntimePacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.InitializeAuthenticationPacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.ExecuteSessionAuthenticationPacket{ContextID: "ctx", Initialization: conversationInitialization},
		internal_type.SessionAuthenticationSucceededPacket{ContextID: "ctx", Initialization: conversationInitialization},
		internal_type.SessionAuthenticationFailedPacket{ContextID: "ctx", Initialization: conversationInitialization, Error: initializationError},
		internal_type.InitializeSpeechToTextPacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.InitializeTextToSpeechPacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.InitializeAssistantExecutorPacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.InitializeVoiceActivityDetectionPacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.InitializeEndOfSpeechPacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.InitializeDenoisePacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.InitializeBehaviorPacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.InitializationCompletedPacket{ContextID: "ctx", Config: conversationInitialization},
		internal_type.InitializationFailedPacket{ContextID: "ctx", Stage: internal_type.InitializationStageService, Error: initializationError},
		internal_type.InitializeTelemetryPacket{ContextID: "ctx"},
		internal_type.InitializeInboundDispatcherPacket{ContextID: "ctx"},
	}

	for _, initializationPacket := range initializationPackets {
		t.Run(string(initializationPacket.PacketName()), func(t *testing.T) {
			requestorChannels := adapter_channel.NewRequestorChannels()
			requestor := &genericRequestor{channels: requestorChannels}

			err := requestor.OnPacket(context.Background(), initializationPacket)
			require.NoError(t, err)

			select {
			case envelope := <-requestor.channels.BootstrapChannel():
				assert.Equal(t, initializationPacket.PacketName(), envelope.Pkt.PacketName())
			default:
				t.Fatalf("expected %s in bootstrap channel", initializationPacket.PacketName())
			}

			assert.Empty(t, requestor.channels.ControlChannel())
			assert.Empty(t, requestor.channels.IngressChannel())
			assert.Empty(t, requestor.channels.EgressChannel())
			assert.Empty(t, requestor.channels.DataChannel())
			assert.Empty(t, requestor.channels.BackgroundChannel())
		})
	}
}

func TestHandleSessionAuthenticationSucceeded_TextMode_EnqueuesTextInitializationPackets(t *testing.T) {
	requestorChannels := adapter_channel.NewRequestorChannels()
	requestor := &genericRequestor{
		source:           utils.Debugger,
		assistant:        &internal_assistant_entity.Assistant{},
		args:             map[string]interface{}{},
		metadata:         map[string]interface{}{},
		options:          map[string]interface{}{},
		messageLifecycle: adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle: adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateInitializing),
		dispatchRoute:    adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		channels:         requestorChannels,
	}
	requestor.assistant.Id = 101
	requestor.assistant.AssistantProviderId = 202
	requestor.assistantConversation = &internal_conversation_entity.AssistantConversation{}
	requestor.assistantConversation.Id = 303

	requestorDispatchHandler{r: requestor}.HandleSessionAuthenticationSucceeded(context.Background(), internal_type.SessionAuthenticationSucceededPacket{
		ContextID: "ctx-text-init",
		Initialization: &protos.ConversationInitialization{
			StreamMode: protos.StreamMode_STREAM_MODE_TEXT,
		},
	})

	var packetNames []internal_type.PacketName
	for len(requestor.channels.BootstrapChannel()) > 0 {
		packetNames = append(packetNames, (<-requestor.channels.BootstrapChannel()).Pkt.PacketName())
	}

	assert.Equal(t, []internal_type.PacketName{
		internal_type.PacketNameInitializeAssistantExecutor,
		internal_type.PacketNameInitializeBehavior,
		internal_type.PacketNameInitializationCompleted,
	}, packetNames)
	assert.Equal(t, type_enums.TextMode, requestor.GetMode())
}

func TestHandleSessionAuthenticationSucceeded_AudioMode_EnqueuesAudioInitializationPackets(t *testing.T) {
	requestorChannels := adapter_channel.NewRequestorChannels()
	requestor := &genericRequestor{
		source:           utils.Debugger,
		assistant:        &internal_assistant_entity.Assistant{},
		args:             map[string]interface{}{},
		metadata:         map[string]interface{}{},
		options:          map[string]interface{}{},
		messageLifecycle: adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle: adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateInitializing),
		dispatchRoute:    adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		channels:         requestorChannels,
	}
	requestor.assistant.Id = 101
	requestor.assistant.AssistantProviderId = 202
	requestor.assistantConversation = &internal_conversation_entity.AssistantConversation{}
	requestor.assistantConversation.Id = 303

	requestorDispatchHandler{r: requestor}.HandleSessionAuthenticationSucceeded(context.Background(), internal_type.SessionAuthenticationSucceededPacket{
		ContextID: "ctx-audio-init",
		Initialization: &protos.ConversationInitialization{
			StreamMode: protos.StreamMode_STREAM_MODE_AUDIO,
		},
	})

	var packetNames []internal_type.PacketName
	for len(requestor.channels.BootstrapChannel()) > 0 {
		packetNames = append(packetNames, (<-requestor.channels.BootstrapChannel()).Pkt.PacketName())
	}

	assert.Equal(t, []internal_type.PacketName{
		internal_type.PacketNameInitializeSpeechToText,
		internal_type.PacketNameInitializeTextToSpeech,
		internal_type.PacketNameInitializeAssistantExecutor,
		internal_type.PacketNameInitializeVoiceActivityDetection,
		internal_type.PacketNameInitializeEndOfSpeech,
		internal_type.PacketNameInitializeDenoise,
		internal_type.PacketNameInitializeBehavior,
		internal_type.PacketNameInitializationCompleted,
	}, packetNames)
	assert.Equal(t, type_enums.AudioMode, requestor.GetMode())
}

func TestHandleInitializationCompleted_EmitsConversationWebhookRecord(t *testing.T) {
	testCases := []struct {
		name                    string
		assistantConversationID uint64
		expectedEvent           observability.EventName
		expectedDataKey         string
		expectedDataValue       interface{}
	}{
		{
			name:              "begin",
			expectedEvent:     observability.ConversationBegin,
			expectedDataKey:   "is_new",
			expectedDataValue: "true",
		},
		{
			name:                    "resume",
			assistantConversationID: 303,
			expectedEvent:           observability.ConversationResume,
			expectedDataKey:         "message_count",
			expectedDataValue:       "0",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			requestorChannels := adapter_channel.NewRequestorChannels()
			messageLifecycle := adapter_lifecycle.NewMessageLifecycle()
			messageLifecycle.SetContextID("ctx-init-webhook")
			messageLifecycle.SetMode(type_enums.TextMode)
			requestor := &genericRequestor{
				source:           utils.Debugger,
				streamer:         &streamTestStreamer{},
				assistant:        &internal_assistant_entity.Assistant{},
				args:             map[string]interface{}{},
				metadata:         map[string]interface{}{},
				options:          map[string]interface{}{},
				messageLifecycle: messageLifecycle,
				sessionLifecycle: adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateInitializing),
				dispatchRoute:    adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
				channels:         requestorChannels,
			}
			requestor.assistant.Id = 101
			requestor.assistant.AssistantProviderId = 202
			requestor.assistantConversation = &internal_conversation_entity.AssistantConversation{}
			requestor.assistantConversation.Id = 303

			requestorDispatchHandler{r: requestor}.HandleInitializationCompleted(context.Background(), internal_type.InitializationCompletedPacket{
				ContextID: "ctx-init-webhook",
				Config: &protos.ConversationInitialization{
					AssistantConversationId: testCase.assistantConversationID,
				},
			})

			var webhookPacket internal_type.ObservabilityWebhookRecordPacket
			found := false
			for len(requestor.channels.BackgroundChannel()) > 0 {
				envelope := <-requestor.channels.BackgroundChannel()
				packet, ok := envelope.Pkt.(internal_type.ObservabilityWebhookRecordPacket)
				if !ok {
					continue
				}
				webhookPacket = packet
				found = true
			}

			require.True(t, found, "expected ObservabilityWebhookRecordPacket")
			assert.Equal(t, "ctx-init-webhook", webhookPacket.ContextID)
			assert.Equal(t, internal_type.ObservabilityRecordScopeConversation, webhookPacket.Scope)
			assert.Equal(t, testCase.expectedEvent, webhookPacket.Record.Event)
			assert.Equal(t, testCase.expectedEvent.String(), webhookPacket.Record.Payload["event"])
			assistantPayload, ok := webhookPacket.Record.Payload["assistant"].(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, uint64(101), assistantPayload["id"])
			conversationPayload, ok := webhookPacket.Record.Payload["conversation"].(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, uint64(303), conversationPayload["id"])
			dataPayload, ok := webhookPacket.Record.Payload["data"].(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, testCase.expectedDataValue, dataPayload[testCase.expectedDataKey])
		})
	}
}

func TestHandleFinalizeSessionRuntime_EmitsConversationCompletedWebhookRecord(t *testing.T) {
	requestorChannels := adapter_channel.NewRequestorChannels()
	requestor := &genericRequestor{
		assistant:                 &internal_assistant_entity.Assistant{},
		assistantConversation:     &internal_conversation_entity.AssistantConversation{},
		messageLifecycle:          adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle:          adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateDisconnecting),
		dispatchRoute:             adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		channels:                  requestorChannels,
		assistantAnalyseExecutors: []internal_type.AnalysisExecutor{},
		histories: []internal_type.MessagePacket{
			internal_type.MessageCreatePacket{
				ContextID:   "msg-user-1",
				MessageRole: "user",
				Text:        "hello",
			},
			internal_type.MessageCreatePacket{
				ContextID:   "msg-assistant-1",
				MessageRole: "assistant",
				Text:        "hi",
			},
		},
		metadata: map[string]interface{}{
			"customer_id": "customer-1",
		},
		metrics: map[string]*protos.Metric{
			"turn_count": {
				Name:        "turn_count",
				Value:       "2",
				Description: "Number of conversation turns",
			},
		},
	}
	requestor.assistant.Id = 101
	requestor.assistantConversation.Id = 303

	requestorDispatchHandler{r: requestor}.HandleFinalizeSessionRuntime(context.Background(), internal_type.FinalizeSessionRuntimePacket{
		ContextID: "ctx-final-webhook",
	})

	var webhookPacket internal_type.ObservabilityWebhookRecordPacket
	found := false
	for len(requestor.channels.BackgroundChannel()) > 0 {
		envelope := <-requestor.channels.BackgroundChannel()
		packet, ok := envelope.Pkt.(internal_type.ObservabilityWebhookRecordPacket)
		if !ok {
			continue
		}
		webhookPacket = packet
		found = true
	}

	require.True(t, found, "expected ObservabilityWebhookRecordPacket")
	assert.Equal(t, "ctx-final-webhook", webhookPacket.ContextID)
	assert.Equal(t, internal_type.ObservabilityRecordScopeConversation, webhookPacket.Scope)
	assert.Equal(t, observability.ConversationCompleted, webhookPacket.Record.Event)
	assert.Equal(t, observability.ConversationCompleted.String(), webhookPacket.Record.Payload["event"])
	assistantPayload, ok := webhookPacket.Record.Payload["assistant"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, uint64(101), assistantPayload["id"])
	conversationPayload, ok := webhookPacket.Record.Payload["conversation"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, uint64(303), conversationPayload["id"])
	dataPayload, ok := webhookPacket.Record.Payload["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "conversation_completed", dataPayload["reason"])
	assert.Equal(t, "completed", dataPayload["status"])
	messagesPayload, ok := dataPayload["messages"].([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, messagesPayload, 2)
	assert.Equal(t, "msg-user-1", messagesPayload[0]["id"])
	assert.Equal(t, "user", messagesPayload[0]["role"])
	assert.Equal(t, "hello", messagesPayload[0]["content"])
	metadataPayload, ok := dataPayload["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "customer-1", metadataPayload["customer_id"])
	metricsPayload, ok := dataPayload["metrics"].([]map[string]interface{})
	require.True(t, ok)
	metricValues := map[string]string{}
	for _, metric := range metricsPayload {
		name, _ := metric["name"].(string)
		value, _ := metric["value"].(string)
		metricValues[name] = value
	}
	assert.Equal(t, "2", metricValues["turn_count"])
	assert.Equal(t, type_enums.CONVERSATION_COMPLETE.String(), metricValues[type_enums.CONVERSATION_STATUS.String()])
}

func TestHandleError_NonRecoverable_EmitsConversationErrorWebhookRecord(t *testing.T) {
	requestorChannels := adapter_channel.NewRequestorChannels()
	requestor := &genericRequestor{
		streamer:              &streamTestStreamer{},
		assistant:             &internal_assistant_entity.Assistant{},
		assistantConversation: &internal_conversation_entity.AssistantConversation{},
		messageLifecycle:      adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle:      adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateInitializing),
		dispatchRoute:         adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		channels:              requestorChannels,
	}
	requestor.assistant.Id = 101
	requestor.assistantConversation.Id = 303

	requestorDispatchHandler{r: requestor}.HandleError(context.Background(), internal_type.InitializationFailedPacket{
		ContextID: "ctx-error-webhook",
		Stage:     internal_type.InitializationStageTextToSpeech,
		Error:     errors.New("tts provider rejected credentials"),
	})

	var webhookPacket internal_type.ObservabilityWebhookRecordPacket
	found := false
	for len(requestor.channels.BackgroundChannel()) > 0 {
		envelope := <-requestor.channels.BackgroundChannel()
		packet, ok := envelope.Pkt.(internal_type.ObservabilityWebhookRecordPacket)
		if !ok {
			continue
		}
		webhookPacket = packet
		found = true
	}

	require.True(t, found, "expected ObservabilityWebhookRecordPacket")
	assert.Equal(t, "ctx-error-webhook", webhookPacket.ContextID)
	assert.Equal(t, internal_type.ObservabilityRecordScopeConversation, webhookPacket.Scope)
	assert.Equal(t, observability.ConversationError, webhookPacket.Record.Event)
	assert.Equal(t, observability.ConversationError.String(), webhookPacket.Record.Payload["event"])
	assistantPayload, ok := webhookPacket.Record.Payload["assistant"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, uint64(101), assistantPayload["id"])
	conversationPayload, ok := webhookPacket.Record.Payload["conversation"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, uint64(303), conversationPayload["id"])
	dataPayload, ok := webhookPacket.Record.Payload["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, protos.ConversationDisconnection_DISCONNECTION_TYPE_ERROR.String(), dataPayload["reason"])
	assert.Contains(t, dataPayload["message"], "tts provider rejected credentials")
}

func TestHandleInitializeBehavior_BehaviorUnavailable_LogsAndReturns(t *testing.T) {
	requestorChannels := adapter_channel.NewRequestorChannels()
	requestor := &genericRequestor{
		source:           utils.Debugger,
		messageLifecycle: adapter_lifecycle.NewMessageLifecycle(),
		sessionLifecycle: adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateInitializing),
		dispatchRoute:    adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), requestorChannels),
		channels:         requestorChannels,
	}

	requestorDispatchHandler{r: requestor}.HandleInitializeBehavior(context.Background(), internal_type.InitializeBehaviorPacket{
		ContextID: "ctx-behavior-missing",
		Config:    &protos.ConversationInitialization{},
	})

	select {
	case envelope := <-requestor.channels.BackgroundChannel():
		_, ok := envelope.Pkt.(internal_type.ObservabilityLogRecordPacket)
		require.True(t, ok, "expected ObservabilityLogRecordPacket, got %T", envelope.Pkt)
	default:
		t.Fatal("expected behavior initialization failure log")
	}
	assert.Empty(t, requestor.channels.BootstrapChannel())
}
