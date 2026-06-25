// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package errors

import "net/http"

const (
	DeleteUserFromProjectUnauthenticatedCode     ErrorCode = 1024001
	DeleteUserFromProjectMissingOrganizationCode ErrorCode = 1024002
	DeleteUserFromProjectUnauthorizedCode        ErrorCode = 1024003
	DeleteUserFromProjectInvalidUserCode         ErrorCode = 1024004
	DeleteUserFromProjectUserNotInOrgCode        ErrorCode = 1024005
	DeleteUserFromProjectInvalidProjectCode      ErrorCode = 1024006
	DeleteUserFromProjectUserNotInProjectCode    ErrorCode = 1024007
	DeleteUserFromProjectArchiveRoleCode         ErrorCode = 1024008
)

var (
	DeleteUserFromProjectUnauthenticated = PlatformError{
		HTTPStatusCode: http.StatusUnauthorized,
		Code:           DeleteUserFromProjectUnauthenticatedCode,
		Error:          "unauthenticated request",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	DeleteUserFromProjectMissingOrganization = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           DeleteUserFromProjectMissingOrganizationCode,
		Error:          "missing active organization",
		ErrorMessage:   "Please create or switch to an active organization before deleting users from projects.",
	}
	DeleteUserFromProjectUnauthorized = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           DeleteUserFromProjectUnauthorizedCode,
		Error:          "user is not authorized to delete users from projects",
		ErrorMessage:   "You do not have permission to delete users from projects.",
	}
	DeleteUserFromProjectInvalidUser = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           DeleteUserFromProjectInvalidUserCode,
		Error:          "invalid user id",
		ErrorMessage:   "Please select a valid user.",
	}
	DeleteUserFromProjectUserNotInOrg = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           DeleteUserFromProjectUserNotInOrgCode,
		Error:          "user is not part of this organization",
		ErrorMessage:   "Please select a user from this organization.",
	}
	DeleteUserFromProjectInvalidProject = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           DeleteUserFromProjectInvalidProjectCode,
		Error:          "invalid project id",
		ErrorMessage:   "Please select a valid active project in this organization.",
	}
	DeleteUserFromProjectUserNotInProject = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           DeleteUserFromProjectUserNotInProjectCode,
		Error:          "user is not part of this project",
		ErrorMessage:   "Please select a user assigned to this project.",
	}
	DeleteUserFromProjectArchiveRole = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           DeleteUserFromProjectArchiveRoleCode,
		Error:          "unable to delete user from project",
		ErrorMessage:   "Unable to delete user from project, please try again later.",
	}
)
