// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package types

/*
Service scope
*/
type ServiceScope struct {
	UserId         *uint64 `json:"userId"`
	ProjectId      *uint64 `json:"projectId"`
	OrganizationId *uint64 `json:"organizationId"`
	CurrentToken   string  `json:"currentToken"`
}

func (ss *ServiceScope) GetUserId() *uint64 {
	return ss.UserId
}
func (ss *ServiceScope) GetCurrentProjectId() *uint64 {
	return ss.ProjectId
}
func (ss *ServiceScope) GetCurrentOrganizationId() *uint64 {
	return ss.OrganizationId
}

func (ss *ServiceScope) HasOrganization() bool {
	return ss.GetCurrentOrganizationId() != nil
}

func (ss *ServiceScope) HasUser() bool {
	return ss.GetUserId() != nil
}

func (ss *ServiceScope) HasProject() bool {
	return ss.GetCurrentProjectId() != nil
}

func (ss *ServiceScope) IsAuthenticated() bool {
	// org scope is already to have only org
	return (ss.HasUser() || ss.HasProject()) && ss.HasOrganization()
}

func (ss *ServiceScope) GetCurrentToken() string {
	return ss.CurrentToken
}

func (ss *ServiceScope) Type() AuthType {
	return AuthTypeService
}
