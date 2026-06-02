// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_registration

import (
	"context"
	"time"

	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
)

func (m *Manager) RegistrationRenewed(ctx context.Context, event sip_infra.RegistrationEvent) {
	retryCount := 0
	m.writeRegistrationStatus(ctx, event.DeploymentID, RegistrationStatusUpdate{
		Status:        StatusActive,
		Error:         "",
		RetryCount:    &retryCount,
		OwnerInstance: m.instanceID,
		LastSuccessAt: time.Now().UTC(),
	})
}

func (m *Manager) RegistrationRenewalFailed(ctx context.Context, event sip_infra.RegistrationEvent) {
	m.writeRegistrationStatus(ctx, event.DeploymentID, m.registrationStatusUpdateFromEvent(event, StatusActive))
}

func (m *Manager) RegistrationExpired(ctx context.Context, event sip_infra.RegistrationEvent) {
	m.writeRegistrationStatus(ctx, event.DeploymentID, m.registrationStatusUpdateFromEvent(event, StatusUnreachable))
	m.releaseOwner(ctx, event.DID)
}

func (m *Manager) RegistrationUnregisterFailed(ctx context.Context, event sip_infra.RegistrationEvent) {
	m.writeRegistrationStatus(ctx, event.DeploymentID, m.registrationStatusUpdateFromEvent(event, StatusActive))
}

func (m *Manager) registrationStatusUpdateFromEvent(event sip_infra.RegistrationEvent, status RegistrationStatus) RegistrationStatusUpdate {
	return RegistrationStatusUpdate{
		Status:        status,
		Error:         registrationEventError(event),
		FailureClass:  event.FailureClass,
		FailureReason: event.FailureReason,
		ResponseCode:  event.StatusCode,
		ResponseText:  event.StatusText,
		RetryCount:    &event.RetryCount,
		LastAttemptAt: time.Now().UTC(),
		NextRetryAt:   event.NextRetryAt,
		OwnerInstance: m.instanceID,
	}
}

func registrationEventError(event sip_infra.RegistrationEvent) string {
	if event.Error == nil {
		return ""
	}
	return event.Error.Error()
}
