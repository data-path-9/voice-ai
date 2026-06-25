// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package errors

import "net/http"

const (
	DeleteUserFromOrganizationUnauthenticatedCode     ErrorCode = 1023001
	DeleteUserFromOrganizationMissingOrganizationCode ErrorCode = 1023002
	DeleteUserFromOrganizationUnauthorizedCode        ErrorCode = 1023003
	DeleteUserFromOrganizationInvalidUserCode         ErrorCode = 1023004
	DeleteUserFromOrganizationUserNotInOrgCode        ErrorCode = 1023005
	DeleteUserFromOrganizationArchiveUserCode         ErrorCode = 1023006
	DeleteUserFromOrganizationOwnerCode               ErrorCode = 1023007
)

var (
	DeleteUserFromOrganizationUnauthenticated = PlatformError{
		HTTPStatusCode: http.StatusUnauthorized,
		Code:           DeleteUserFromOrganizationUnauthenticatedCode,
		Error:          "unauthenticated request",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	DeleteUserFromOrganizationMissingOrganization = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           DeleteUserFromOrganizationMissingOrganizationCode,
		Error:          "missing active organization",
		ErrorMessage:   "Please create or switch to an active organization before deleting users.",
	}
	DeleteUserFromOrganizationUnauthorized = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           DeleteUserFromOrganizationUnauthorizedCode,
		Error:          "user is not authorized to delete users from organization",
		ErrorMessage:   "You do not have permission to delete users from this organization.",
	}
	DeleteUserFromOrganizationInvalidUser = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           DeleteUserFromOrganizationInvalidUserCode,
		Error:          "invalid user id",
		ErrorMessage:   "Please select a valid user.",
	}
	DeleteUserFromOrganizationUserNotInOrg = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           DeleteUserFromOrganizationUserNotInOrgCode,
		Error:          "user is not part of this organization",
		ErrorMessage:   "Please select a user from this organization.",
	}
	DeleteUserFromOrganizationArchiveUser = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           DeleteUserFromOrganizationArchiveUserCode,
		Error:          "unable to delete user from organization",
		ErrorMessage:   "Unable to delete user from organization, please try again later.",
	}
	DeleteUserFromOrganizationOwner = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           DeleteUserFromOrganizationOwnerCode,
		Error:          "cannot delete organization owner",
		ErrorMessage:   "Organization owners cannot be deleted.",
	}
)
