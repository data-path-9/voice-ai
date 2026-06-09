// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_telephony

import (
	"bufio"
	"context"
	"net"
	"testing"
)

func TestStreamerOption_AppliesSIPStreamerOptions(t *testing.T) {
	ctx := context.Background()
	var resolvedOptions StreamerOptions

	WithSIPStreamer(ctx, nil, nil)(&resolvedOptions)

	if resolvedOptions.Context != ctx {
		t.Fatal("expected streamer options to preserve context")
	}
}

func TestStreamerOption_AppliesAudioSocketStreamerOptions(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	reader := bufio.NewReader(clientConn)
	writer := bufio.NewWriter(clientConn)
	var resolvedOptions StreamerOptions

	WithAudioSocketStreamer(serverConn, reader, writer)(&resolvedOptions)

	if resolvedOptions.AudioSocketConn != serverConn {
		t.Fatal("expected streamer options to preserve AudioSocket connection")
	}
	if resolvedOptions.AudioSocketReader != reader {
		t.Fatal("expected streamer options to preserve AudioSocket reader")
	}
	if resolvedOptions.AudioSocketWriter != writer {
		t.Fatal("expected streamer options to preserve AudioSocket writer")
	}
}
