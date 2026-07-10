// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_talk_api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	channel_telephony "github.com/rapidaai/api/assistant-api/internal/channel/telephony"
)

// answerProvider is implemented by telephony providers that must return XML
// from an answer_url before the media WebSocket connects (e.g. vobiz_websocket
// returns a <Stream> verb pointing at the contextId WebSocket route).
type answerProvider interface {
	AnswerXML(provider, contextID string) (string, error)
}

// CallAnswerByContext serves the answer_url XML for providers that fetch it
// before connecting media. The XML is derived from the path params + config;
// no DB lookup is required.
func (cApi *ConversationApi) CallAnswerByContext(c *gin.Context) {
	provider := c.Param("telephony")
	contextID := c.Param("contextId")
	if contextID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing contextId"})
		return
	}

	tp, err := channel_telephony.GetTelephony(channel_telephony.Telephony(provider), cApi.cfg, cApi.logger)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown telephony provider"})
		return
	}
	ap, ok := tp.(answerProvider)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider does not support answer XML"})
		return
	}
	xml, err := ap.AnswerXML(provider, contextID)
	if err != nil {
		cApi.logger.Errorf("answer XML build failed for provider %s: %v", provider, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build answer"})
		return
	}
	c.Data(http.StatusOK, "text/xml", []byte(xml))
}
