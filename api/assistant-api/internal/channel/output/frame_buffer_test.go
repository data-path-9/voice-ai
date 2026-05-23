// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_channel_output

import "testing"

func TestBytesFrameBuffer_Next_HoldsPartialFrame(t *testing.T) {
	b := NewBytesFrameBuffer(0)
	b.Write([]byte{1, 2, 3})

	if frame, ok := b.Next(4); ok || frame != nil {
		t.Fatal("expected no frame when buffer has partial data")
	}
	if got := b.Len(); got != 3 {
		t.Fatalf("expected length 3, got %d", got)
	}
}

func TestBytesFrameBuffer_Complete_PadsToFrame(t *testing.T) {
	b := NewBytesFrameBuffer(0)
	b.Write([]byte{1, 2, 3})
	b.Complete(4, 0xFF)

	frame, ok := b.Next(4)
	if !ok {
		t.Fatal("expected frame after completion padding")
	}
	if len(frame) != 4 {
		t.Fatalf("expected frame size 4, got %d", len(frame))
	}
	if frame[3] != 0xFF {
		t.Fatalf("expected pad byte 0xFF, got 0x%X", frame[3])
	}
}

func TestBytesFrameBuffer_Clear(t *testing.T) {
	b := NewBytesFrameBuffer(0)
	b.Write([]byte{1, 2, 3, 4})
	b.Clear()
	if got := b.Len(); got != 0 {
		t.Fatalf("expected empty buffer, got %d", got)
	}
}
