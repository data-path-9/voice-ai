// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_callcontext

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
)

// Store provides operations to save and retrieve call contexts from Postgres.
//
// Call contexts are session-scoped records that bridge the HTTP call-setup request
// (inbound webhook or outbound gRPC) and the media connection (AudioSocket/WebSocket).
//
// Status lifecycle: PENDING → CLAIMED
//   - Save creates with PENDING (call in progress)
//   - Claim transitions to CLAIMED (call ended, context consumed)
//   - Get reads regardless of status (async callbacks may arrive anytime)
type Store interface {
	// Save stores a call context with status PENDING and a generated contextId.
	Save(ctx context.Context, cc *CallContext) (string, error)

	// Get retrieves a call context by contextId regardless of status.
	// Used by outbound dispatcher (read call info) and async callbacks.
	Get(ctx context.Context, contextID string) (*CallContext, error)

	// Claim atomically transitions PENDING → CLAIMED. Only one caller wins.
	// Called when the call session ends to mark the context as consumed.
	Claim(ctx context.Context, contextID string) (*CallContext, error)

	// UpdateField sets a single column on an existing call context.
	UpdateField(ctx context.Context, contextID, field, value string) error
}

type postgresStore struct {
	postgres connectors.PostgresConnector
	logger   commons.Logger
}

func NewStore(postgres connectors.PostgresConnector, logger commons.Logger) Store {
	return &postgresStore{
		postgres: postgres,
		logger:   logger,
	}
}

func (s *postgresStore) Save(ctx context.Context, cc *CallContext) (string, error) {
	if cc.ContextID == "" {
		cc.ContextID = uuid.New().String()
	}
	cc.Status = StatusPending

	db := s.postgres.DB(ctx)
	if err := db.Create(cc).Error; err != nil {
		return "", fmt.Errorf("failed to save call context %s: %w", cc.ContextID, err)
	}

	s.logger.Infof("saved call context: contextId=%s, assistant=%d, conversation=%d, direction=%s",
		cc.ContextID, cc.AssistantID, cc.ConversationID, cc.Direction)

	return cc.ContextID, nil
}

func (s *postgresStore) Get(ctx context.Context, contextID string) (*CallContext, error) {
	db := s.postgres.DB(ctx)
	var cc CallContext
	if err := db.Where("context_id = ?", contextID).First(&cc).Error; err != nil {
		return nil, fmt.Errorf("call context not found: %s: %w", contextID, err)
	}
	return &cc, nil
}

// Claim atomically transitions PENDING → CLAIMED in a single query.
func (s *postgresStore) Claim(ctx context.Context, contextID string) (*CallContext, error) {
	db := s.postgres.DB(ctx)

	var cc CallContext
	result := db.Raw(`
		UPDATE call_contexts
		SET status = ?, updated_date = ?
		WHERE context_id = ? AND status = ?
		RETURNING *`,
		StatusClaimed, time.Now(), contextID, StatusPending,
	).Scan(&cc)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to claim call context %s: %w", contextID, result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("call context %s not found or already claimed", contextID)
	}

	s.logger.Debugf("claimed call context: contextId=%s, assistant=%d, conversation=%d",
		cc.ContextID, cc.AssistantID, cc.ConversationID)

	return &cc, nil
}

func (s *postgresStore) UpdateField(ctx context.Context, contextID, field, value string) error {
	db := s.postgres.DB(ctx)

	allowed := map[string]bool{
		"channel_uuid": true,
		"status":       true,
		"provider":     true,
	}
	if !allowed[field] {
		return fmt.Errorf("field %q is not updatable on call context", field)
	}

	result := db.Model(&CallContext{}).
		Where("context_id = ?", contextID).
		Update(field, value)

	if result.Error != nil {
		return fmt.Errorf("failed to update field %s on call context %s: %w", field, contextID, result.Error)
	}

	s.logger.Debugf("updated call context field: contextId=%s, %s=%s", contextID, field, value)
	return nil
}
