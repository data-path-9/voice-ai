package internal_llm_model

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

type modelRecvResult struct {
	out *protos.StreamChatResponse
	err error
}

type mockModelStream struct {
	mu        sync.Mutex
	sendCalls []*protos.StreamChatRequest
	sendErr   error
	recvCh    chan modelRecvResult
	closeSent atomic.Bool
}

func newMockModelStream() *mockModelStream {
	return &mockModelStream{
		recvCh: make(chan modelRecvResult, 16),
	}
}

func (m *mockModelStream) Send(req *protos.StreamChatRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendCalls = append(m.sendCalls, req)
	return m.sendErr
}

func (m *mockModelStream) Recv() (*protos.StreamChatResponse, error) {
	r, ok := <-m.recvCh
	if !ok {
		return nil, io.EOF
	}
	return r.out, r.err
}

func (m *mockModelStream) CloseSend() error {
	m.closeSent.Store(true)
	close(m.recvCh)
	return nil
}

func (m *mockModelStream) Header() (metadata.MD, error) { return nil, nil }
func (m *mockModelStream) Trailer() metadata.MD         { return nil }
func (m *mockModelStream) Context() context.Context     { return context.Background() }
func (m *mockModelStream) SendMsg(any) error            { return nil }
func (m *mockModelStream) RecvMsg(any) error            { return nil }

func newTestModelConnection(t *testing.T, stream *mockModelStream) *ModelConnection {
	t.Helper()

	connection := NewModelConnection("openai")
	if stream != nil {
		require.NoError(t, connection.OpenStream(context.Background(), &testComm{
			integrationCaller: &testIntegrationClient{stream: stream},
		}))
	}
	return connection
}

func TestModelConnection_OpenStreamRejectsNilCommunication(t *testing.T) {
	connection := NewModelConnection("openai")

	err := connection.OpenStream(context.Background(), nil)

	require.ErrorIs(t, err, ErrModelConnectionOpenStream)
	require.ErrorIs(t, err, ErrModelConnectionNotConnected)
}

func TestModelConnection_OpenStreamRejectsNilStream(t *testing.T) {
	connection := NewModelConnection("openai")

	err := connection.OpenStream(context.Background(), &testComm{
		integrationCaller: &testIntegrationClient{},
	})

	require.ErrorIs(t, err, ErrModelConnectionOpenStream)
	require.ErrorIs(t, err, ErrModelConnectionNotConnected)
}

func TestModelConnection_OpenStreamRejectsDuplicateStream(t *testing.T) {
	stream := newMockModelStream()
	connection := newTestModelConnection(t, stream)
	duplicateStream := newMockModelStream()
	integration := &testIntegrationClient{stream: duplicateStream}

	err := connection.OpenStream(context.Background(), &testComm{integrationCaller: integration})

	require.ErrorIs(t, err, ErrModelConnectionOpenStream)
	require.ErrorIs(t, err, ErrModelConnectionStreamAlreadyOpen)
	assert.Equal(t, 0, integration.calls)
	assert.False(t, duplicateStream.closeSent.Load())
	assert.False(t, stream.closeSent.Load())
}

func TestModelConnection_Send(t *testing.T) {
	stream := newMockModelStream()
	connection := newTestModelConnection(t, stream)
	req := &protos.StreamChatRequest{
		Request: &protos.StreamChatRequest_Chat{
			Chat: &protos.StreamChatInput{RequestId: "ctx-1"},
		},
	}

	require.NoError(t, connection.Send(req))

	require.Len(t, stream.sendCalls, 1)
	assert.Equal(t, "ctx-1", stream.sendCalls[0].GetChat().GetRequestId())
}

func TestModelConnection_Recv(t *testing.T) {
	stream := newMockModelStream()
	connection := newTestModelConnection(t, stream)
	expected := &protos.StreamChatResponse{
		Response: &protos.StreamChatResponse_Close{
			Close: &protos.StreamChatClose{Reason: "done"},
		},
	}
	stream.recvCh <- modelRecvResult{out: expected}

	actual, err := connection.Recv()

	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestModelConnection_NotConnected(t *testing.T) {
	connection := NewModelConnection("openai")

	err := connection.Send(&protos.StreamChatRequest{})
	require.ErrorIs(t, err, ErrModelConnectionSend)
	require.ErrorIs(t, err, ErrModelConnectionNotConnected)

	_, err = connection.Recv()
	require.ErrorIs(t, err, ErrModelConnectionRecv)
	require.ErrorIs(t, err, ErrModelConnectionNotConnected)
}

func TestModelConnection_CloseIsIdempotent(t *testing.T) {
	stream := newMockModelStream()
	connection := newTestModelConnection(t, stream)

	require.NoError(t, connection.Close("session ended"))
	require.NoError(t, connection.Close("session ended"))

	assert.True(t, stream.closeSent.Load())
	require.Len(t, stream.sendCalls, 1)
	assert.Equal(t, "session ended", stream.sendCalls[0].GetClose().GetReason())
}

func TestModelConnection_ConcurrentSendAndClose(t *testing.T) {
	stream := newMockModelStream()
	connection := newTestModelConnection(t, stream)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			err := connection.Send(&protos.StreamChatRequest{
				Request: &protos.StreamChatRequest_Chat{
					Chat: &protos.StreamChatInput{RequestId: "ctx"},
				},
			})
			if err != nil && !errors.Is(err, ErrModelConnectionNotConnected) {
				t.Errorf("unexpected send error: %v", err)
			}
		}
	}()

	go func() {
		defer wg.Done()
		time.Sleep(time.Millisecond)
		assert.NoError(t, connection.Close("session ended"))
	}()

	wg.Wait()
}
