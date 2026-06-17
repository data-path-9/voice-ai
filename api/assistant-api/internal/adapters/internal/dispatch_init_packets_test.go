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
