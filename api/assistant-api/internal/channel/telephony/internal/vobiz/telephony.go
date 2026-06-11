// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_vobiz_telephony

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/rapidaai/api/assistant-api/config"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_vobiz "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/vobiz/internal"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/clients/vobiz"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

// newVobizClient is overridable in tests.
var newVobizClient = vobiz.NewClient

type vobizTelephony struct {
	appCfg *config.AssistantConfig
	logger commons.Logger
}

func NewVobizTelephony(cfg *config.AssistantConfig, logger commons.Logger) (internal_type.Telephony, error) {
	return &vobizTelephony{appCfg: cfg, logger: logger}, nil
}

// vobizCreds extracts the account API credentials from the vault credential.
func vobizCreds(vaultCredential *protos.VaultCredential) (authID, authToken, fromNumber string, err error) {
	if vaultCredential.GetValue() == nil {
		return "", "", "", internal_vobiz.ErrVaultCredentialValueMissing
	}
	m := vaultCredential.GetValue().AsMap()
	authID, _ = m["auth_id"].(string)
	authToken, _ = m["auth_token"].(string)
	fromNumber, _ = m["from_number"].(string)
	if authID == "" {
		return "", "", "", internal_vobiz.ErrVaultAuthIDMissing
	}
	if authToken == "" {
		return "", "", "", internal_vobiz.ErrVaultAuthTokenMissing
	}
	return authID, authToken, fromNumber, nil
}

// answerPath is the outbound answer_url path vobiz fetches to get the <Stream> XML.
func answerPath(contextID string) string {
	return fmt.Sprintf("v1/talk/%s/ctx/%s/answer", internal_vobiz.VobizProvider, contextID)
}

// isSafeContextID guards against XML injection: context IDs are system-generated
// UUIDs, so only alphanumerics and hyphens are allowed in the answer-XML URLs.
func isSafeContextID(id string) bool {
	if id == "" {
		return false
	}
	for _, r := range id {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-') {
			return false
		}
	}
	return true
}

// AnswerXML returns the <Stream> XML that points vobiz at our WebSocket. It is
// built from path params only (no DB lookup), invoked by the answer route.
func (v *vobizTelephony) AnswerXML(provider, contextID string) (string, error) {
	if !isSafeContextID(contextID) {
		return "", fmt.Errorf("invalid context id %q", contextID)
	}
	public := v.appCfg.Assistant.Public
	wsURL := fmt.Sprintf("wss://%s/%s", public, internal_type.GetContextAnswerPath(provider, contextID))
	statusURL := fmt.Sprintf("https://%s/%s", public, internal_type.GetContextEventPath(provider, contextID))
	return fmt.Sprintf(`<Response><Stream bidirectional="true" audioTrack="inbound" contentType="audio/x-mulaw;rate=8000" keepCallAlive="true" statusCallbackUrl="%s">%s</Stream></Response>`, statusURL, wsURL), nil
}

func (v *vobizTelephony) OutboundCall(ctx context.Context, auth types.SimplePrinciple, toPhone string, fromPhone string, assistant *internal_assistant_entity.Assistant, assistantConversationId uint64, vaultCredential *protos.VaultCredential, statusReporter internal_type.ProviderCallStatusReporter, opts utils.Option) (*internal_type.CallInfo, error) {
	info := &internal_type.CallInfo{Provider: internal_vobiz.VobizProvider}

	if err := ctx.Err(); err != nil {
		info.Status = "FAILED"
		info.ErrorMessage = fmt.Sprintf("request cancelled: %s", err.Error())
		internal_telephony_base.ReportOutboundFailure(statusReporter,
			internal_telephony_base.OutboundFailureClassRequestCancelled, "request cancelled",
			internal_telephony_base.OutboundDisconnectReasonRequestCancelled, err, 0)
		return info, err
	}

	authID, authToken, fromNumber, err := vobizCreds(vaultCredential)
	if err != nil {
		info.Status = "FAILED"
		info.ErrorMessage = fmt.Sprintf("authentication error: %s", err.Error())
		internal_telephony_base.ReportOutboundFailure(statusReporter,
			internal_telephony_base.OutboundFailureClassAuthentication, "authentication error",
			internal_telephony_base.OutboundDisconnectReasonSetupFailed, err, 0)
		return info, err
	}

	from := fromPhone
	if from == "" {
		from = fromNumber
	}
	contextID, _ := opts.GetString("rapida.context_id")
	if contextID == "" {
		err := fmt.Errorf("missing rapida.context_id; cannot build answer/event callback URLs")
		info.Status = "FAILED"
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(statusReporter,
			internal_telephony_base.OutboundFailureClassProviderResponse, "missing context id",
			internal_telephony_base.OutboundDisconnectReasonSetupFailed, err, 0)
		return info, err
	}
	public := v.appCfg.Assistant.Public
	answerURL := fmt.Sprintf("https://%s/%s", public, answerPath(contextID))
	eventURL := fmt.Sprintf("https://%s/%s", public, internal_type.GetContextEventPath(internal_vobiz.VobizProvider, contextID))

	resp, err := newVobizClient().MakeCall(ctx, authID, authToken, vobiz.MakeCallRequest{
		From:         from,
		To:           toPhone,
		AnswerURL:    answerURL,
		AnswerMethod: "POST",
		RingURL:      eventURL,
		RingMethod:   "POST",
		HangupURL:    eventURL,
		HangupMethod: "POST",
	})
	if err != nil {
		info.Status = "FAILED"
		info.ErrorMessage = fmt.Sprintf("API error: %s", err.Error())
		internal_telephony_base.ReportOutboundFailure(statusReporter,
			internal_telephony_base.OutboundFailureClassProviderAPI, "provider API error",
			internal_telephony_base.OutboundDisconnectReasonSetupFailed, err, 0)
		return info, err
	}
	if resp.RequestUUID == "" {
		err := internal_vobiz.ErrOutboundResponseMissingUUID
		info.Status = "FAILED"
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(statusReporter,
			internal_telephony_base.OutboundFailureClassProviderResponse, "provider response missing request_uuid",
			internal_telephony_base.OutboundDisconnectReasonSetupFailed, err, 0)
		return info, err
	}

	info.ChannelUUID = resp.RequestUUID
	info.Status = "SUCCESS"
	info.StatusInfo = internal_type.StatusInfo{Event: resp.Message, Payload: resp}
	internal_telephony_base.ReportOutboundInitiated(statusReporter, info.ChannelUUID)
	return info, nil
}

// InboundCall answers an incoming call by returning the <Stream> XML.
func (v *vobizTelephony) InboundCall(c *gin.Context, auth types.SimplePrinciple, assistantId uint64, clientNumber string, assistantConversationId uint64) error {
	contextID, _ := c.Get("contextId")
	xml, err := v.AnswerXML(internal_vobiz.VobizProvider, fmt.Sprintf("%v", contextID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build answer"})
		return err
	}
	c.Data(http.StatusOK, "text/xml", []byte(xml))
	return nil
}

func (v *vobizTelephony) ReceiveCall(c *gin.Context) (*internal_type.CallInfo, error) {
	queryParams := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			queryParams[key] = values[0]
		}
	}
	info := &internal_type.CallInfo{
		Provider:   internal_vobiz.VobizProvider,
		Status:     "SUCCESS",
		StatusInfo: internal_type.StatusInfo{Event: "webhook", Payload: queryParams},
	}
	if v := queryParams["From"]; v != "" {
		info.CallerNumber = v
	}
	if v := queryParams["To"]; v != "" {
		info.FromNumber = v
	}
	if v := queryParams["CallUUID"]; v != "" {
		info.ChannelUUID = v
	}
	return info, nil
}

func (v *vobizTelephony) StatusCallback(c *gin.Context, auth types.SimplePrinciple, assistantId uint64, assistantConversationId uint64) (*internal_type.StatusInfo, error) {
	eventDetails, raw, err := parseCallbackPayload(c)
	if err != nil {
		return nil, err
	}
	callback, err := internal_vobiz.NewStatusCallback(eventDetails, raw)
	if err != nil {
		return nil, err
	}
	return callback.StatusInfo(), nil
}

func (v *vobizTelephony) CatchAllStatusCallback(c *gin.Context) (*internal_type.StatusInfo, error) {
	eventDetails, raw, err := parseCallbackPayload(c)
	if err != nil {
		return nil, err
	}
	callback, err := internal_vobiz.NewStatusCallback(eventDetails, raw)
	if err != nil {
		return nil, err
	}
	return callback.StatusInfo(), nil
}

// parseCallbackPayload reads query params (preferred) or url-encoded body into
// a utils.Option map, returning the raw payload too.
func parseCallbackPayload(c *gin.Context) (utils.Option, string, error) {
	eventDetails := utils.Option{}
	raw := c.Request.URL.RawQuery
	if len(c.Request.URL.Query()) > 0 {
		for key, values := range c.Request.URL.Query() {
			if len(values) > 0 {
				eventDetails[key] = values[0]
			} else {
				eventDetails[key] = nil
			}
		}
		return eventDetails, raw, nil
	}
	body, err := c.GetRawData()
	if err != nil {
		return nil, "", err
	}
	raw = string(body)
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, "", err
	}
	for key, value := range values {
		if len(value) > 0 {
			eventDetails[key] = value[0]
		} else {
			eventDetails[key] = nil
		}
	}
	return eventDetails, raw, nil
}
