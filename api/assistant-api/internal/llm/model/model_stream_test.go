package internal_llm_model

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestModel_Listen_RecvError_EmitsSystemPanic(t *testing.T) {
	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)

	errStream := &listenErrorStream{recvErr: errors.New("stream broke")}
	connection := NewModelConnection("openai")
	require.NoError(t, connection.OpenStream(context.Background(), &testComm{
		integrationCaller: &testIntegrationClient{stream: errStream},
	}))
	e := &modelAssistantExecutor{
		logger:        logger,
		history:       NewConversationHistory(),
		connection:    connection,
		currentPacket: &internal_type.UserInputPacket{ContextID: "ctx-1"},
	}
	comm := &testComm{
		assistant: &internal_assistant_entity.Assistant{
			AssistantProviderModel: &internal_assistant_entity.AssistantProviderModel{ModelProviderName: "openai"},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e.listen(ctx, comm)

	errPkt, ok := findPacket[internal_type.LLMErrorPacket](comm.pkts)
	require.True(t, ok)
	require.Equal(t, "ctx-1", errPkt.ContextID)
	require.Equal(t, internal_type.LLMSystemPanic, errPkt.Type)
}

func TestModel_SendStreamConfiguration_IncludesConnectionOptions(t *testing.T) {
	e, comm, stream, _ := newModelTestEnv(t)
	comm.options = utils.Option{
		"connection.transport": "websocket",
		"model.temperature":    "0.4",
	}

	require.NoError(t, e.sendStreamConfiguration(context.Background(), e.connection, comm))
	require.Len(t, stream.sendCalls, 1)

	cfg := stream.sendCalls[0].GetConfiguration()
	require.NotNil(t, cfg)
	require.Equal(t, "websocket", cfg.GetConnectionOptions()["connection.transport"])
	_, hasModelKey := cfg.GetConnectionOptions()["model.temperature"]
	require.False(t, hasModelKey)
}

type listenErrorStream struct {
	recvErr error
}

func (m *listenErrorStream) Send(*protos.StreamChatRequest) error { return nil }
func (m *listenErrorStream) Recv() (*protos.StreamChatResponse, error) {
	if m.recvErr != nil {
		err := m.recvErr
		m.recvErr = nil
		return nil, err
	}
	time.Sleep(5 * time.Millisecond)
	return nil, io.EOF
}
func (m *listenErrorStream) CloseSend() error             { return nil }
func (m *listenErrorStream) Header() (metadata.MD, error) { return nil, nil }
func (m *listenErrorStream) Trailer() metadata.MD         { return nil }
func (m *listenErrorStream) Context() context.Context     { return context.Background() }
func (m *listenErrorStream) SendMsg(any) error            { return nil }
func (m *listenErrorStream) RecvMsg(any) error            { return nil }
