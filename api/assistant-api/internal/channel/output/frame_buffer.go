// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_channel_output

import (
	"bytes"
	"sync"
)

// FrameBuffer defines a thread-safe output buffer contract used by paced
// channel writers. It holds encoded provider bytes and yields fixed-size frames.
type FrameBuffer interface {
	Write(data []byte)
	Next(frameSize int) ([]byte, bool)
	Complete(frameSize int, padByte byte)
	Clear()
	Len() int
}

type BytesFrameBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func NewBytesFrameBuffer(capacity int) *BytesFrameBuffer {
	b := &BytesFrameBuffer{}
	if capacity > 0 {
		b.buf.Grow(capacity)
	}
	return b
}

func (b *BytesFrameBuffer) Write(data []byte) {
	if len(data) == 0 {
		return
	}
	b.mu.Lock()
	b.buf.Write(data)
	b.mu.Unlock()
}

func (b *BytesFrameBuffer) Next(frameSize int) ([]byte, bool) {
	if frameSize <= 0 {
		return nil, false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.buf.Len() < frameSize {
		return nil, false
	}
	frame := make([]byte, frameSize)
	_, _ = b.buf.Read(frame)
	return frame, true
}

func (b *BytesFrameBuffer) Complete(frameSize int, padByte byte) {
	if frameSize <= 0 {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.buf.Len() == 0 {
		return
	}
	rem := b.buf.Len() % frameSize
	if rem == 0 {
		return
	}
	pad := frameSize - rem
	padding := make([]byte, pad)
	if padByte != 0 {
		for i := range padding {
			padding[i] = padByte
		}
	}
	b.buf.Write(padding)
}

func (b *BytesFrameBuffer) Clear() {
	b.mu.Lock()
	b.buf.Reset()
	b.mu.Unlock()
}

func (b *BytesFrameBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Len()
}
