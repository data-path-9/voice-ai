// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import (
	"context"

	internal_core "github.com/rapidaai/api/assistant-api/sip/internal/core"
)

type BridgeEndReason int

const (
	BridgeEndInboundBye BridgeEndReason = iota
	BridgeEndOutboundBye
	BridgeEndContext
	BridgeEndTimeout
)

func (s *Server) BridgeTransfer(ctx context.Context, inbound, outbound *Session, onOperatorAudio func([]byte)) (BridgeEndReason, error) {
	reason, err := s.inner.BridgeTransfer(ctx, inbound.unwrap(), outbound.unwrap(), onOperatorAudio)
	return BridgeEndReason(reason), err
}

func bridgeEndReasonToCore(reason BridgeEndReason) internal_core.BridgeEndReason {
	return internal_core.BridgeEndReason(reason)
}
