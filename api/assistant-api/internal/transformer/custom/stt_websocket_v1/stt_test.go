// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_stt_websocket_v1

import (
	"context"
	"encoding/base64"
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sttPacketCollector struct {
	mu      sync.Mutex
	packets []internal_type.Packet
}

func (collector *sttPacketCollector) onPacket(pkt ...internal_type.Packet) error {
	collector.mu.Lock()
	defer collector.mu.Unlock()
	collector.packets = append(collector.packets, pkt...)
	return nil
}

func (collector *sttPacketCollector) all() []internal_type.Packet {
	collector.mu.Lock()
	defer collector.mu.Unlock()
	out := make([]internal_type.Packet, len(collector.packets))
	copy(out, collector.packets)
	return out
}

func waitForCondition(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for condition")
}

func TestSpeechToText_WebsocketFlow_JSONAudioRequest(t *testing.T) {
	var (
		gotAuthHeader string
		gotModel      string
		gotAudioReq   map[string]any
	)

	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotAuthHeader = request.Header.Get("Authorization")
		gotModel = request.URL.Query().Get("model")

		conn, err := upgrader.Upgrade(writer, request, nil)
		require.NoError(t, err)
		defer conn.Close()

		messageType, payload, err := conn.ReadMessage()
		require.NoError(t, err)
		require.Equal(t, websocket.TextMessage, messageType)
		require.NoError(t, json.Unmarshal(payload, &gotAudioReq))

		require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"partial","text":"hello","confidence":0.4,"language":"en-US"}`)))
		require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"final","text":"hello world","confidence":0.9,"language":"en-US"}`)))
		require.NoError(t, conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second),
		))
	}))
	defer server.Close()

	baseURL := strings.Replace(server.URL, "http://", "ws://", 1)
	collector := &sttPacketCollector{}

	transformer, err := NewSpeechToText(
		context.Background(),
		transformer_testutil.NewTestLogger(),
		testCredential(t, map[string]any{
			credentialKeyBaseURLCamel: baseURL,
			credentialKeyHeaders:      `{"Authorization":"Bearer abc"}`,
		}),
		collector.onPacket,
		utils.Option{
			optionKeyModel:       "model-a",
			optionKeyQueryParams: `{"model":{"$var":"model"}}`,
			optionKeyAudioRequest: `{
				"audio":{"$var":"audio"},
				"encoding":{"$var":"encoding"},
				"sample_rate":{"$cast":"number","value":{"$var":"sample_rate"}}
			}`,
			optionKeyResponseParser: `[
				{"when":{"frame":"json","path":"type","equals":"partial"},"emit":{"script":{"$path":"text"},"confidence":{"$cast":"number","value":{"$path":"confidence"}},"language":{"$path":"language"},"interim":true}},
				{"when":{"frame":"json","path":"type","equals":"final"},"emit":{"script":{"$path":"text"},"confidence":{"$cast":"number","value":{"$path":"confidence"}},"language":{"$path":"language"},"interim":false}}
			]`,
		},
	)
	require.NoError(t, err)
	require.NoError(t, transformer.Initialize())
	require.NoError(t, transformer.Transform(context.Background(), internal_type.TurnChangePacket{ContextID: "ctx-1"}))
	require.NoError(t, transformer.Transform(context.Background(), internal_type.SpeechToTextInterruptPacket{ContextID: "ctx-1"}))

	audio := []byte{0x01, 0x02, 0x03, 0x04}
	require.NoError(t, transformer.Transform(context.Background(), internal_type.SpeechToTextAudioPacket{
		ContextID: "ctx-1",
		Audio:     audio,
	}))

	waitForCondition(t, 3*time.Second, func() bool {
		for _, packet := range collector.all() {
			transcript, ok := packet.(internal_type.SpeechToTextPacket)
			if ok && !transcript.Interim && transcript.Script == "hello world" {
				return true
			}
		}
		return false
	})

	require.NoError(t, transformer.Close(context.Background()))

	assert.Equal(t, "Bearer abc", gotAuthHeader)
	assert.Equal(t, "model-a", gotModel)
	require.NotEmpty(t, gotAudioReq)
	assert.Equal(t, "LINEAR16", gotAudioReq["encoding"])
	assert.Equal(t, float64(16000), gotAudioReq["sample_rate"])

	encoded, ok := gotAudioReq["audio"].(string)
	require.True(t, ok)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, err)
	assert.Equal(t, audio, decoded)

	var (
		hasInterimTranscript       bool
		hasFinalTranscript         bool
		hasLatencyMetric           bool
		hasSpeechToTextError       bool
		interruptionPacketCount    int
		latencyMetricPacketCount   int
		firstLatencyMetricIndex    = -1
		finalTranscriptPacketIndex = -1
	)

	for packetIndex, packet := range collector.all() {
		switch typed := packet.(type) {
		case internal_type.SpeechToTextPacket:
			if typed.Interim && typed.Script == "hello" {
				hasInterimTranscript = true
			}
			if !typed.Interim && typed.Script == "hello world" {
				hasFinalTranscript = true
				finalTranscriptPacketIndex = packetIndex
			}
		case internal_type.InterruptionDetectedPacket:
			interruptionPacketCount++
		case internal_type.UserMessageMetricPacket:
			for _, metric := range typed.Metrics {
				if metric.GetName() == "stt_latency_ms" {
					hasLatencyMetric = true
					latencyMetricPacketCount++
					if firstLatencyMetricIndex == -1 {
						firstLatencyMetricIndex = packetIndex
					}
				}
			}
		case internal_type.SpeechToTextErrorPacket:
			hasSpeechToTextError = true
		}
	}

	assert.True(t, hasInterimTranscript, "expected interim transcript")
	assert.True(t, hasFinalTranscript, "expected final transcript")
	assert.GreaterOrEqual(t, interruptionPacketCount, 2, "expected interruption packet for interim and final transcripts")
	assert.True(t, hasLatencyMetric, "expected stt_latency_ms metric")
	assert.Equal(t, 1, latencyMetricPacketCount, "expected one latency metric per interruption window")
	assert.NotEqual(t, -1, firstLatencyMetricIndex, "expected latency metric packet index")
	assert.NotEqual(t, -1, finalTranscriptPacketIndex, "expected final transcript packet index")
	assert.Less(t, firstLatencyMetricIndex, finalTranscriptPacketIndex, "expected latency metric before final transcript packet")
	assert.False(t, hasSpeechToTextError, "did not expect stt error packet")
}

func TestSpeechToText_BinaryAudioResampledWithoutAudioRequest(t *testing.T) {
	var (
		gotMessageType int
		gotAudioChunk  []byte
	)

	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		conn, err := upgrader.Upgrade(writer, request, nil)
		require.NoError(t, err)
		defer conn.Close()

		messageType, payload, err := conn.ReadMessage()
		require.NoError(t, err)
		gotMessageType = messageType
		gotAudioChunk = append([]byte(nil), payload...)

		require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"final","text":"ok","language":"en-US","confidence":0.8}`)))
		require.NoError(t, conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second),
		))
	}))
	defer server.Close()

	baseURL := strings.Replace(server.URL, "http://", "ws://", 1)
	collector := &sttPacketCollector{}

	transformer, err := NewSpeechToText(
		context.Background(),
		transformer_testutil.NewTestLogger(),
		testCredential(t, map[string]any{
			credentialKeyBaseURLCamel: baseURL,
		}),
		collector.onPacket,
		utils.Option{
			optionKeyEncoding:   "MuLaw8",
			optionKeySampleRate: "8000",
			optionKeyResponseParser: `[
				{"when":{"frame":"json","path":"type","equals":"final"},"emit":{"script":{"$path":"text"},"confidence":{"$cast":"number","value":{"$path":"confidence"}},"language":{"$path":"language"},"interim":false}}
			]`,
		},
	)
	require.NoError(t, err)
	require.NoError(t, transformer.Initialize())
	require.NoError(t, transformer.Transform(context.Background(), internal_type.TurnChangePacket{ContextID: "ctx-2"}))
	require.NoError(t, transformer.Transform(context.Background(), internal_type.SpeechToTextInterruptPacket{ContextID: "ctx-2"}))

	audio := transformer_testutil.SineTonePCM(440, 1.0)
	require.NoError(t, transformer.Transform(context.Background(), internal_type.SpeechToTextAudioPacket{
		ContextID: "ctx-2",
		Audio:     audio,
	}))

	waitForCondition(t, 3*time.Second, func() bool {
		for _, packet := range collector.all() {
			transcript, ok := packet.(internal_type.SpeechToTextPacket)
			if ok && !transcript.Interim && transcript.Script == "ok" {
				return true
			}
		}
		return false
	})

	require.NoError(t, transformer.Close(context.Background()))

	assert.Equal(t, websocket.BinaryMessage, gotMessageType)
	assert.Greater(t, len(gotAudioChunk), 0, "expected non-empty resampled chunk")
	assert.NotEqual(t, audio, gotAudioChunk)
}

func TestSpeechToText_WebsocketFlow_TextTranscriptFrames(t *testing.T) {
	var (
		gotMessageType int
		gotAudioChunk  []byte
	)

	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		conn, err := upgrader.Upgrade(writer, request, nil)
		require.NoError(t, err)
		defer conn.Close()

		messageType, payload, err := conn.ReadMessage()
		require.NoError(t, err)
		gotMessageType = messageType
		gotAudioChunk = append([]byte(nil), payload...)

		require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte("namaste duniya")))
		require.NoError(t, conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second),
		))
	}))
	defer server.Close()

	baseURL := strings.Replace(server.URL, "http://", "ws://", 1)
	collector := &sttPacketCollector{}

	transformer, err := NewSpeechToText(
		context.Background(),
		transformer_testutil.NewTestLogger(),
		testCredential(t, map[string]any{
			credentialKeyBaseURLCamel: baseURL,
		}),
		collector.onPacket,
		utils.Option{
			optionKeyResponseParser: `[
				{"when":{"frame":"text"},"emit":{"script":{"$frame":"text"},"language":"hi","interim":false}}
			]`,
		},
	)
	require.NoError(t, err)
	require.NoError(t, transformer.Initialize())
	require.NoError(t, transformer.Transform(context.Background(), internal_type.TurnChangePacket{ContextID: "ctx-text"}))
	require.NoError(t, transformer.Transform(context.Background(), internal_type.SpeechToTextInterruptPacket{ContextID: "ctx-text"}))
	require.NoError(t, transformer.Transform(context.Background(), internal_type.SpeechToTextAudioPacket{
		ContextID: "ctx-text",
		Audio:     []byte{0x01, 0x02, 0x03, 0x04},
	}))

	waitForCondition(t, 3*time.Second, func() bool {
		for _, packet := range collector.all() {
			transcript, ok := packet.(internal_type.SpeechToTextPacket)
			if ok && !transcript.Interim && transcript.Script == "namaste duniya" {
				return true
			}
		}
		return false
	})

	require.NoError(t, transformer.Close(context.Background()))

	assert.Equal(t, websocket.BinaryMessage, gotMessageType)
	assert.NotEmpty(t, gotAudioChunk)

	var (
		hasTranscript bool
		hasLatency    bool
	)

	for _, packet := range collector.all() {
		switch typed := packet.(type) {
		case internal_type.SpeechToTextPacket:
			if !typed.Interim && typed.Script == "namaste duniya" && typed.Language == "hi" {
				hasTranscript = true
			}
		case internal_type.UserMessageMetricPacket:
			for _, metric := range typed.Metrics {
				if metric.GetName() == "stt_latency_ms" {
					hasLatency = true
				}
			}
		}
	}

	assert.True(t, hasTranscript, "expected transcript from text response frame")
	assert.True(t, hasLatency, "expected final transcript latency metric")
}

func TestSpeechToText_DoesNotEmitLatencyMetricWithoutInterruption(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		connection, err := upgrader.Upgrade(writer, request, nil)
		require.NoError(t, err)
		defer connection.Close()

		_, _, err = connection.ReadMessage()
		require.NoError(t, err)

		require.NoError(t, connection.WriteMessage(websocket.TextMessage, []byte(`{"type":"final","text":"hello","confidence":0.8,"language":"en-US"}`)))
		require.NoError(t, connection.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second),
		))
	}))
	defer server.Close()

	baseURL := strings.Replace(server.URL, "http://", "ws://", 1)
	collector := &sttPacketCollector{}

	transformer, err := NewSpeechToText(
		context.Background(),
		transformer_testutil.NewTestLogger(),
		testCredential(t, map[string]any{
			credentialKeyBaseURLCamel: baseURL,
		}),
		collector.onPacket,
		utils.Option{
			optionKeyResponseParser: `[
				{"when":{"frame":"json","path":"type","equals":"final"},"emit":{"script":{"$path":"text"},"confidence":{"$cast":"number","value":{"$path":"confidence"}},"language":{"$path":"language"},"interim":false}}
			]`,
		},
	)
	require.NoError(t, err)
	require.NoError(t, transformer.Initialize())
	require.NoError(t, transformer.Transform(context.Background(), internal_type.TurnChangePacket{ContextID: "ctx-no-interruption"}))
	require.NoError(t, transformer.Transform(context.Background(), internal_type.SpeechToTextAudioPacket{
		ContextID: "ctx-no-interruption",
		Audio:     []byte{0x01, 0x02, 0x03, 0x04},
	}))

	waitForCondition(t, 3*time.Second, func() bool {
		for _, packet := range collector.all() {
			transcript, ok := packet.(internal_type.SpeechToTextPacket)
			if ok && !transcript.Interim && transcript.Script == "hello" {
				return true
			}
		}
		return false
	})

	require.NoError(t, transformer.Close(context.Background()))

	latencyMetricPacketCount := 0
	for _, packet := range collector.all() {
		metricPacket, ok := packet.(internal_type.UserMessageMetricPacket)
		if !ok {
			continue
		}
		for _, metric := range metricPacket.Metrics {
			if metric.GetName() == "stt_latency_ms" {
				latencyMetricPacketCount++
			}
		}
	}

	assert.Equal(t, 0, latencyMetricPacketCount, "did not expect latency metric without interruption start")
}
