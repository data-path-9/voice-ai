// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package adapter_internal

import (
	"context"

	type_enums "github.com/rapidaai/pkg/types/enums"
)

func (kr *genericRequestor) CreateKnowledgeLog(ctx context.Context, knowledgeId uint64, retrievalMethod string,
	topK uint32,
	scoreThreshold float32,
	documentCount int,
	timeTaken int64,
	additionalData map[string]string,
	status type_enums.RecordState,
	request, response []byte) error {
	dbCtx, cancel := context.WithTimeout(context.Background(), dbWriteTimeout)
	defer cancel()
	_, err := kr.knowledgeService.CreateLog(dbCtx, kr.Auth(), knowledgeId, retrievalMethod, topK, scoreThreshold, documentCount, timeTaken, additionalData, status, request, response)
	return err
}

func (cr *genericRequestor) CreateHTTPLog(
	ctx context.Context,
	source string,
	sourceRefID uint64,
	sourceEvent string,
	contextID string,
	httpURL, httpMethod string,
	responseStatus int64,
	timeTaken int64,
	retryCount uint32,
	status type_enums.RecordState,
	errorMessage *string,
	request, response []byte) error {
	dbCtx, cancel := context.WithTimeout(context.Background(), dbWriteTimeout)
	defer cancel()

	var conversationID *uint64
	if cr.assistantConversation != nil {
		conversationID = &cr.assistantConversation.Id
	}
	var assistantID uint64
	if cr.assistant != nil {
		assistantID = cr.assistant.Id
	}

	_, err := cr.httpLogService.CreateLog(
		dbCtx,
		cr.auth,
		source,
		sourceRefID,
		sourceEvent,
		contextID,
		assistantID,
		conversationID,
		httpURL,
		httpMethod,
		responseStatus,
		timeTaken,
		retryCount,
		status,
		errorMessage,
		request,
		response,
	)
	return err
}

func (cr *genericRequestor) CreateToolLog(
	ctx context.Context,
	messageId string,
	toolCallId string,
	toolName string,
	status type_enums.RecordState,
	request []byte) error {
	dbCtx, cancel := context.WithTimeout(context.Background(), dbWriteTimeout)
	defer cancel()
	_, err := cr.assistantToolService.CreateLog(
		dbCtx, cr.Auth(), cr.assistant.Id,
		cr.assistantConversation.Id, messageId, toolCallId, toolName,
		status, request,
	)
	return err
}

func (cr *genericRequestor) UpdateToolLog(
	ctx context.Context,
	toolCallId string,
	status type_enums.RecordState,
	response []byte) error {
	dbCtx, cancel := context.WithTimeout(context.Background(), dbWriteTimeout)
	defer cancel()
	_, err := cr.assistantToolService.UpdateLog(
		dbCtx, cr.Auth(), toolCallId, cr.assistantConversation.Id,
		status, response,
	)
	return err
}
