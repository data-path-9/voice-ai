// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package errors

import "net/http"

const (
	CreatePhoneCallInvalidRequestCode        ErrorCode = 1002001
	CreatePhoneCallUnauthenticatedCode       ErrorCode = 1002002
	CreatePhoneCallMissingToNumberCode       ErrorCode = 1002003
	CreatePhoneCallInvalidAssistantCode      ErrorCode = 1002004
	CreatePhoneCallInvalidMetadataCode       ErrorCode = 1002005
	CreatePhoneCallInvalidArgumentsCode      ErrorCode = 1002006
	CreatePhoneCallInvalidOptionsCode        ErrorCode = 1002007
	CreatePhoneCallInitiateOutboundCode      ErrorCode = 1002008
	CreateBulkPhoneCallInvalidRequestCode    ErrorCode = 1003001
	CreateBulkPhoneCallUnauthenticatedCode   ErrorCode = 1003002
	CreateBulkPhoneCallMissingPhoneCallsCode ErrorCode = 1003003
	CreateBulkPhoneCallMissingToNumberCode   ErrorCode = 1003004
	CreateBulkPhoneCallInvalidAssistantCode  ErrorCode = 1003005
	CreateBulkPhoneCallInvalidMetadataCode   ErrorCode = 1003006
	CreateBulkPhoneCallInvalidArgumentsCode  ErrorCode = 1003007
	CreateBulkPhoneCallInvalidOptionsCode    ErrorCode = 1003008
	CreateBulkPhoneCallInitiateOutboundCode  ErrorCode = 1003009
)

var (
	CreatePhoneCallInvalidRequest = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreatePhoneCallInvalidRequestCode,
		Error:          "invalid request",
		ErrorMessage:   "Invalid request.",
	}
	CreatePhoneCallUnauthenticated = PlatformError{
		HTTPStatusCode: http.StatusUnauthorized,
		Code:           CreatePhoneCallUnauthenticatedCode,
		Error:          "unauthenticated request",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	CreatePhoneCallMissingToNumber = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreatePhoneCallMissingToNumberCode,
		Error:          "missing toNumber parameter",
		ErrorMessage:   "Please provide the required toNumber parameter.",
	}
	CreatePhoneCallInvalidAssistant = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreatePhoneCallInvalidAssistantCode,
		Error:          "invalid assistant parameter",
		ErrorMessage:   "Please provide a valid assistant.",
	}
	CreatePhoneCallInvalidMetadata = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreatePhoneCallInvalidMetadataCode,
		Error:          "invalid metadata parameter",
		ErrorMessage:   "Please provide valid metadata.",
	}
	CreatePhoneCallInvalidArguments = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreatePhoneCallInvalidArgumentsCode,
		Error:          "invalid arguments parameter",
		ErrorMessage:   "Please provide valid arguments.",
	}
	CreatePhoneCallInvalidOptions = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreatePhoneCallInvalidOptionsCode,
		Error:          "invalid options parameter",
		ErrorMessage:   "Please provide valid options.",
	}
	CreatePhoneCallInitiateOutbound = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           CreatePhoneCallInitiateOutboundCode,
		Error:          "unable to initiate outbound call",
		ErrorMessage:   "Unable to initiate outbound call, please try again later.",
	}
	CreateBulkPhoneCallInvalidRequest = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateBulkPhoneCallInvalidRequestCode,
		Error:          "invalid request",
		ErrorMessage:   "Invalid request.",
	}
	CreateBulkPhoneCallUnauthenticated = PlatformError{
		HTTPStatusCode: http.StatusUnauthorized,
		Code:           CreateBulkPhoneCallUnauthenticatedCode,
		Error:          "unauthenticated request",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	CreateBulkPhoneCallMissingPhoneCalls = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateBulkPhoneCallMissingPhoneCallsCode,
		Error:          "missing phone_calls parameter",
		ErrorMessage:   "Please provide at least one phone call.",
	}
	CreateBulkPhoneCallMissingToNumber = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateBulkPhoneCallMissingToNumberCode,
		Error:          "missing toNumber parameter",
		ErrorMessage:   "Please provide the required toNumber parameter.",
	}
	CreateBulkPhoneCallInvalidAssistant = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateBulkPhoneCallInvalidAssistantCode,
		Error:          "invalid assistant parameter",
		ErrorMessage:   "Please provide a valid assistant.",
	}
	CreateBulkPhoneCallInvalidMetadata = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateBulkPhoneCallInvalidMetadataCode,
		Error:          "invalid metadata parameter",
		ErrorMessage:   "Please provide valid metadata.",
	}
	CreateBulkPhoneCallInvalidArguments = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateBulkPhoneCallInvalidArgumentsCode,
		Error:          "invalid arguments parameter",
		ErrorMessage:   "Please provide valid arguments.",
	}
	CreateBulkPhoneCallInvalidOptions = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateBulkPhoneCallInvalidOptionsCode,
		Error:          "invalid options parameter",
		ErrorMessage:   "Please provide valid options.",
	}
	CreateBulkPhoneCallInitiateOutbound = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           CreateBulkPhoneCallInitiateOutboundCode,
		Error:          "unable to initiate outbound call",
		ErrorMessage:   "Unable to initiate outbound call, please try again later.",
	}
)
