// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package core

import (
	internal_inbound "github.com/rapidaai/api/assistant-api/sip/internal/inbound"
)

type inboundSetupError struct {
	statusCode   int
	failureClass internal_inbound.FailureClass
	reason       LifecycleReason
	err          error
}

func newInboundSetupError(statusCode int, failureClass internal_inbound.FailureClass, reason LifecycleReason, err error) *inboundSetupError {
	return &inboundSetupError{
		statusCode:   statusCode,
		failureClass: failureClass,
		reason:       reason,
		err:          err,
	}
}

func (e *inboundSetupError) Error() string {
	if e == nil || e.err == nil {
		return "inbound setup failed"
	}
	return e.err.Error()
}

func (e *inboundSetupError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func inboundSetupFailureDetails(
	err error,
	statusCode int,
	failureClass internal_inbound.FailureClass,
	reason LifecycleReason,
) (int, internal_inbound.FailureClass, LifecycleReason, error) {
	if setupError, ok := err.(*inboundSetupError); ok {
		return setupError.statusCode, setupError.failureClass, setupError.reason, setupError.err
	}
	return statusCode, failureClass, reason, err
}
