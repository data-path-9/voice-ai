// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package adapter_internal

import (
	"context"
	"fmt"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
)

func (deb *genericRequestor) createMessage(_ context.Context, msg internal_type.MessagePacket) error {
	assistant, err := deb.Assistant()
	if err != nil {
		return err
	}
	conversation, err := deb.Conversation()
	if err != nil {
		return err
	}
	deb.histories = append(deb.histories, msg)
	utils.Go(context.Background(), func() {
		dbCtx, cancel := context.WithTimeout(context.Background(), dbWriteTimeout)
		defer cancel()
		_, err := deb.conversationService.CreateConversationMessage(dbCtx, deb.Auth(), deb.GetSource(), assistant.Id, assistant.AssistantProviderId, conversation.Id,
			fmt.Sprintf("%s-%s", msg.Role(), msg.ContextId()), msg.Role(), msg.Content())
		if err != nil {
			deb.logger.Debugf("error while persisting conversation recording %+v", err)
		}
	})
	return nil
}

func (gr *genericRequestor) createConversationRecording(_ context.Context, user, assistant, conversation []byte) error {
	currentAssistant, err := gr.Assistant()
	if err != nil {
		return err
	}
	currentConversation, err := gr.Conversation()
	if err != nil {
		return err
	}
	utils.Go(context.Background(), func() {
		dbCtx, cancel := context.WithTimeout(context.Background(), recordingTimeout)
		defer cancel()
		_, err := gr.conversationService.CreateConversationRecording(dbCtx, gr.auth, currentAssistant.Id, currentConversation.Id, user, assistant, conversation)
		if err != nil {
			gr.logger.Debugf("error while persisting conversation recording %+v", err)
		}
	})
	return nil
}

func (kr *genericRequestor) createKnowledgeLog(ctx context.Context, knowledgeId uint64, retrievalMethod string,
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
