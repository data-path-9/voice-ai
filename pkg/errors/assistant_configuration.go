// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package errors

import "net/http"

const (
	AssistantConfigurationInvalidRequestCode     ErrorCode = 1019001
	AssistantConfigurationUnauthenticatedCode    ErrorCode = 1019002
	AssistantConfigurationMissingAuthScopeCode   ErrorCode = 1019003
	AssistantConfigurationInvalidAssistantIDCode ErrorCode = 1019004
	AssistantConfigurationInvalidIDCode          ErrorCode = 1019005
	AssistantConfigurationMissingTypeCode        ErrorCode = 1019006
	AssistantConfigurationInvalidTypeCode        ErrorCode = 1019007
	AssistantConfigurationMissingProviderCode    ErrorCode = 1019008
	AssistantConfigurationInvalidOptionCode      ErrorCode = 1019009
	AssistantConfigurationCreateCode             ErrorCode = 1019010
	AssistantConfigurationUpdateCode             ErrorCode = 1019011
	AssistantConfigurationGetCode                ErrorCode = 1019012
	AssistantConfigurationGetAllCode             ErrorCode = 1019013
	AssistantConfigurationDeleteCode             ErrorCode = 1019014
)

var (
	AssistantConfigurationInvalidRequest = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           AssistantConfigurationInvalidRequestCode,
		Error:          "invalid assistant configuration request",
		ErrorMessage:   "Invalid assistant configuration request.",
	}
	AssistantConfigurationUnauthenticated = PlatformError{
		HTTPStatusCode: http.StatusUnauthorized,
		Code:           AssistantConfigurationUnauthenticatedCode,
		Error:          "unauthenticated request",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	AssistantConfigurationMissingAuthScope = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           AssistantConfigurationMissingAuthScopeCode,
		Error:          "missing authentication scope",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	AssistantConfigurationInvalidAssistantID = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           AssistantConfigurationInvalidAssistantIDCode,
		Error:          "invalid assistant_id parameter",
		ErrorMessage:   "Please provide a valid assistantId parameter.",
	}
	AssistantConfigurationInvalidID = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           AssistantConfigurationInvalidIDCode,
		Error:          "invalid assistant configuration id parameter",
		ErrorMessage:   "Please provide a valid assistant configuration id parameter.",
	}
	AssistantConfigurationMissingType = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           AssistantConfigurationMissingTypeCode,
		Error:          "missing configuration_type parameter",
		ErrorMessage:   "Please provide the required configurationType parameter.",
	}
	AssistantConfigurationInvalidType = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           AssistantConfigurationInvalidTypeCode,
		Error:          "invalid configuration_type parameter",
		ErrorMessage:   "Please provide a valid configurationType parameter.",
	}
	AssistantConfigurationMissingProvider = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           AssistantConfigurationMissingProviderCode,
		Error:          "missing provider parameter",
		ErrorMessage:   "Please provide the required provider parameter.",
	}
	AssistantConfigurationInvalidOption = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           AssistantConfigurationInvalidOptionCode,
		Error:          "invalid option parameter",
		ErrorMessage:   "Please provide valid option key/value pairs.",
	}
	AssistantConfigurationCreate = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           AssistantConfigurationCreateCode,
		Error:          "unable to create assistant configuration",
		ErrorMessage:   "Unable to create assistant configuration, please try again later.",
	}
	AssistantConfigurationUpdate = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           AssistantConfigurationUpdateCode,
		Error:          "unable to update assistant configuration",
		ErrorMessage:   "Unable to update assistant configuration, please try again later.",
	}
	AssistantConfigurationGet = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           AssistantConfigurationGetCode,
		Error:          "unable to get assistant configuration",
		ErrorMessage:   "Unable to get assistant configuration, please try again later.",
	}
	AssistantConfigurationGetAll = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           AssistantConfigurationGetAllCode,
		Error:          "unable to get assistant configurations",
		ErrorMessage:   "Unable to get assistant configurations, please try again later.",
	}
	AssistantConfigurationDelete = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           AssistantConfigurationDeleteCode,
		Error:          "unable to delete assistant configuration",
		ErrorMessage:   "Unable to delete assistant configuration, please try again later.",
	}
)
