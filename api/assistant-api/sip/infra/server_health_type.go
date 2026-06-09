// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import internal_core "github.com/rapidaai/api/assistant-api/sip/internal/core"

type ServerHealthSnapshot struct {
	Ready         bool
	Reason        string
	State         ServerState
	ActiveCalls   int
	RTPPortsInUse int
}

func serverHealthSnapshotFromCore(snapshot internal_core.ServerHealthSnapshot) ServerHealthSnapshot {
	return ServerHealthSnapshot{
		Ready:         snapshot.Ready,
		Reason:        snapshot.Reason,
		State:         ServerState(snapshot.State),
		ActiveCalls:   snapshot.ActiveCalls,
		RTPPortsInUse: snapshot.RTPPortsInUse,
	}
}
