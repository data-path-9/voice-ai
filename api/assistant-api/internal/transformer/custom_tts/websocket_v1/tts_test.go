// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_tts_websocket_v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	transformer_testutil "github.com/rapidaai/api/assistant-api/internal/transformer/internal/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

type packetCollector struct {
	mu      sync.Mutex
	packets []internal_type.Packet
	endCh   chan struct{}
}

func newPacketCollector() *packetCollector {
	return &packetCollector{endCh: make(chan struct{}, 1)}
}

func (collector *packetCollector) onPacket(pkt ...internal_type.Packet) error {
	collector.mu.Lock()
	defer collector.mu.Unlock()
	collector.packets = append(collector.packets, pkt...)
	for _, packet := range pkt {
		if _, ok := packet.(internal_type.TextToSpeechEndPacket); ok {
			select {
			case collector.endCh <- struct{}{}:
			default:
			}
		}
	}
	return nil
}

func (collector *packetCollector) all() []internal_type.Packet {
	collector.mu.Lock()
	defer collector.mu.Unlock()
	out := make([]internal_type.Packet, len(collector.packets))
	copy(out, collector.packets)
	return out
}

func (collector *packetCollector) hasTTSError() bool {
	for _, packet := range collector.all() {
		if _, ok := packet.(internal_type.TextToSpeechErrorPacket); ok {
			return true
		}
	}
	return false
}

func testWSCredential(t *testing.T, values map[string]any) *protos.VaultCredential {
	t.Helper()
	pb, err := structpb.NewStruct(values)
	require.NoError(t, err)
	return &protos.VaultCredential{Value: pb}
}

func TestTextToSpeech_WebsocketFlow(t *testing.T) {
	var (
		gotAuthHeader  string
		gotMessageID   string
		gotTextRequest map[string]any
		gotDoneRequest map[string]any
	)

	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotAuthHeader = request.Header.Get("Authorization")
		gotMessageID = request.URL.Query().Get("message_id")

		conn, err := upgrader.Upgrade(writer, request, nil)
		require.NoError(t, err)
		defer conn.Close()

		_, firstMessage, err := conn.ReadMessage()
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(firstMessage, &gotTextRequest))

		require.NoError(t, conn.WriteMessage(websocket.BinaryMessage, []byte("pcm-audio")))

		_, doneMessage, err := conn.ReadMessage()
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(doneMessage, &gotDoneRequest))

		require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(`{"event":"done","request_id":"ctx-1","is_final":true}`)))
	}))
	defer server.Close()

	baseURL := strings.Replace(server.URL, "http://", "ws://", 1)
	collector := newPacketCollector()

	opts := utils.Option{
		optionKeyVoiceID:     "virat",
		optionKeyQueryParams: `{"message_id":{"$var":"message_id"}}`,
		optionKeyTextRequest: `{"text":{"$var":"text"},"voice_id":{"$var":"voice_id"},"request_id":{"$var":"message_id"}}`,
		optionKeyDoneRequest: `{"request_id":{"$var":"message_id"},"continue":false}`,
		optionKeyResponseParser: `[
			{"when":{"frame":"binary"},"emit":{"audio":{"$frame":"binary"}}},
			{"when":{"frame":"json","path":"is_final","equals":true},"emit":{"message_id":{"$path":"request_id"},"done":true}}
		]`,
	}

	transformer, err := NewTextToSpeech(
		context.Background(),
		transformer_testutil.NewTestLogger(),
		testWSCredential(t, map[string]any{
			credentialKeyBaseURLCamel: baseURL,
			credentialKeyHeaders:      `{"Authorization":"Bearer abc"}`,
		}),
		collector.onPacket,
		opts,
	)
	require.NoError(t, err)
	require.NoError(t, transformer.Initialize())

	require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-1",
		Text:      "hello world",
	}))
	require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechDonePacket{
		ContextID: "ctx-1",
	}))

	select {
	case <-collector.endCh:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for TextToSpeechEndPacket")
	}

	require.NoError(t, transformer.Close(context.Background()))

	assert.Equal(t, "Bearer abc", gotAuthHeader)
	assert.Equal(t, "ctx-1", gotMessageID)
	assert.Equal(t, "hello world", gotTextRequest["text"])
	assert.Equal(t, "virat", gotTextRequest["voice_id"])
	assert.Equal(t, "ctx-1", gotTextRequest["request_id"])
	assert.Equal(t, false, gotDoneRequest["continue"])
	assert.Equal(t, "ctx-1", gotDoneRequest["request_id"])

	packets := collector.all()
	var (
		hasAudio bool
		hasEnd   bool
	)
	for _, packet := range packets {
		switch typed := packet.(type) {
		case internal_type.TextToSpeechAudioPacket:
			if typed.ContextID == "ctx-1" && string(typed.AudioChunk) == "pcm-audio" {
				hasAudio = true
			}
		case internal_type.TextToSpeechEndPacket:
			if typed.ContextID == "ctx-1" {
				hasEnd = true
			}
		}
	}

	assert.True(t, hasAudio, "expected audio packet")
	assert.True(t, hasEnd, "expected end packet")
}

func TestTextToSpeech_EndsOnCleanServerClose(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		conn, err := upgrader.Upgrade(writer, request, nil)
		require.NoError(t, err)
		defer conn.Close()

		_, _, err = conn.ReadMessage()
		require.NoError(t, err)

		require.NoError(t, conn.WriteMessage(websocket.BinaryMessage, []byte("pcm-audio")))
		require.NoError(t, conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second),
		))
	}))
	defer server.Close()

	baseURL := strings.Replace(server.URL, "http://", "ws://", 1)
	collector := newPacketCollector()
	transformer, err := NewTextToSpeech(
		context.Background(),
		transformer_testutil.NewTestLogger(),
		testWSCredential(t, map[string]any{
			credentialKeyBaseURLCamel: baseURL,
		}),
		collector.onPacket,
		utils.Option{
			optionKeyVoiceID:        "virat",
			optionKeyTextRequest:    `{"text":{"$var":"text"}}`,
			optionKeyResponseParser: `[{"when":{"frame":"binary"},"emit":{"audio":{"$frame":"binary"}}}]`,
		},
	)
	require.NoError(t, err)
	require.NoError(t, transformer.Initialize())
	require.NoError(t, transformer.Transform(context.Background(), internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-close",
		Text:      "hello",
	}))

	select {
	case <-collector.endCh:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for TextToSpeechEndPacket")
	}

	require.NoError(t, transformer.Close(context.Background()))
	assert.False(t, collector.hasTTSError(), "did not expect TextToSpeechErrorPacket on clean close")
}
