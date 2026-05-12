// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_channel_input

import (
	"bytes"
	"sync"
)

// InputBuffer defines a thread-safe contract for accumulating and draining
// input audio at a configured threshold.
type InputBuffer interface {
	Write(data []byte)
	DrainIfReady(threshold int) ([]byte, bool)
	Clear()
	Len() int
}

type BytesInputBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func NewBytesInputBuffer(capacity int) *BytesInputBuffer {
	b := &BytesInputBuffer{}
	if capacity > 0 {
		b.buf.Grow(capacity)
	}
	return b
}

func (b *BytesInputBuffer) Write(data []byte) {
	if len(data) == 0 {
		return
	}
	b.mu.Lock()
	b.buf.Write(data)
	b.mu.Unlock()
}

func (b *BytesInputBuffer) DrainIfReady(threshold int) ([]byte, bool) {
	if threshold <= 0 {
		return nil, false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.buf.Len() < threshold {
		return nil, false
	}
	out := make([]byte, b.buf.Len())
	_, _ = b.buf.Read(out)
	return out, true
}

func (b *BytesInputBuffer) Clear() {
	b.mu.Lock()
	b.buf.Reset()
	b.mu.Unlock()
}

func (b *BytesInputBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Len()
}
