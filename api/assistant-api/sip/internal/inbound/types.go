// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package inbound

const SDPContentType = "application/sdp"

type FailureClass string

const (
	FailureConfig FailureClass = "config"
	FailureAuth   FailureClass = "auth"
	FailureMedia  FailureClass = "media"
	FailureRTP    FailureClass = "rtp"
	FailureDialog FailureClass = "dialog"
	FailureSetup  FailureClass = "setup"
)
