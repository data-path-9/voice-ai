// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_vonage

import "time"

type VonageWebSocketEvent struct {
	Event string `json:"event"`
}

type VonageClearMessage struct {
	Action string `json:"action"`
}

const (
	ChunkDuration        = 20 * time.Millisecond
	Linear16BytesPerMs   = 32
	OutputChunkSize      = Linear16BytesPerMs * 20
	InputBufferThreshold = Linear16BytesPerMs * 60
)
