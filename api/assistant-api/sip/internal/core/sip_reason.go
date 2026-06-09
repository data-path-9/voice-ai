// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package core

import (
	"strconv"
	"strings"

	"github.com/emiago/sipgo/sip"
)

func parseSIPDisconnectMetadata(request *sip.Request) DisconnectMetadata {
	metadata := DisconnectMetadata{Reason: DisconnectReasonRemoteHangup}
	if request == nil {
		return metadata
	}

	for _, header := range request.GetHeaders("Reason") {
		candidate := parseSIPReasonHeader(header.Value())
		if candidate.Reason == "" {
			continue
		}
		return candidate
	}
	return metadata
}

func parseSIPReasonHeader(headerValue string) DisconnectMetadata {
	metadata := DisconnectMetadata{
		Reason: DisconnectReasonRemoteHangup,
		Raw:    strings.TrimSpace(headerValue),
	}
	parts := splitReasonHeaderParts(headerValue)
	if len(parts) == 0 {
		return metadata
	}

	protocol := strings.ToUpper(strings.TrimSpace(parts[0]))
	for _, part := range parts[1:] {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "cause":
			cause, err := strconv.Atoi(strings.TrimSpace(value))
			if err == nil {
				metadata.ProviderStatusCode = cause
			}
		case "text":
			metadata.Text = strings.Trim(unquoteReasonValue(value), " ")
		}
	}
	metadata.Reason = classifyDisconnectReason(protocol, metadata.ProviderStatusCode)
	return metadata
}

func splitReasonHeaderParts(headerValue string) []string {
	parts := make([]string, 0, 3)
	var builder strings.Builder
	inQuotes := false
	for _, char := range headerValue {
		switch char {
		case '"':
			inQuotes = !inQuotes
			builder.WriteRune(char)
		case ';':
			if inQuotes {
				builder.WriteRune(char)
				continue
			}
			parts = appendReasonPart(parts, builder.String())
			builder.Reset()
		default:
			builder.WriteRune(char)
		}
	}
	parts = appendReasonPart(parts, builder.String())
	return parts
}

func appendReasonPart(parts []string, part string) []string {
	trimmedPart := strings.TrimSpace(part)
	if trimmedPart == "" {
		return parts
	}
	return append(parts, trimmedPart)
}

func unquoteReasonValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
		return value
	}
	return strings.ReplaceAll(value[1:len(value)-1], `\"`, `"`)
}

func classifyDisconnectReason(protocol string, cause int) string {
	switch protocol {
	case "Q.850":
		return classifyQ850DisconnectReason(cause)
	case "SIP":
		return classifySIPDisconnectReason(cause)
	default:
		if cause > 0 {
			return DisconnectReasonRemoteError
		}
		return DisconnectReasonRemoteHangup
	}
}

func classifyQ850DisconnectReason(cause int) string {
	switch cause {
	case 16, 31:
		return DisconnectReasonNormalClearing
	case 17:
		return DisconnectReasonBusy
	case 18, 19:
		return DisconnectReasonNoAnswer
	case 21:
		return DisconnectReasonRejected
	case 34, 38, 41, 42, 47:
		return DisconnectReasonNetworkFailure
	default:
		if cause > 0 {
			return DisconnectReasonRemoteError
		}
		return DisconnectReasonRemoteHangup
	}
}

func classifySIPDisconnectReason(cause int) string {
	switch cause {
	case 200:
		return DisconnectReasonNormalClearing
	case 408, 480:
		return DisconnectReasonNoAnswer
	case 486, 600:
		return DisconnectReasonBusy
	case 487:
		return DisconnectReasonCancelled
	case 403, 603:
		return DisconnectReasonRejected
	case 500, 502, 503, 504:
		return DisconnectReasonNetworkFailure
	default:
		if cause >= 400 {
			return DisconnectReasonRemoteError
		}
		return DisconnectReasonRemoteHangup
	}
}
