// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_vault_service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	"github.com/rapidaai/pkg/clients/vobiz"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

// VobizProvider is the integration code for the Vobiz auto-provisioning flow.
// The provisioned trunk's SIP connection fields (sip_uri/sip_username/
// sip_password) are stored in this credential, and assistant-api routes
// telephony_provider "vobiz_sip" through the SIP engine.
const VobizProvider = "vobiz_sip"

// newVobizClient is overridable in tests.
var newVobizClient = vobiz.NewClient

// provisionVobiz uses the Vobiz trunk-management API to create an outbound SIP
// trunk + username/password credential, then stores the result as a "sip"
// vault credential (so the existing SIP provider can place calls) plus a
// "vobiz" management row holding the account creds + trunk id for idempotency.
//
// It returns the "sip" vault row (carrying sip_uri) so the UI can surface the
// generated trunk domain.
//
// NOTE: provisioning works from anywhere (outbound HTTPS to api.vobiz.ai), but
// completing an actual call still requires assistant-api to run on a public-IP
// host — the SIP provider receives the trunk's responses/RTP directly.
func (vs *vaultService) provisionVobiz(ctx context.Context, auth types.SimplePrinciple, name string, credential map[string]interface{}) (*internal_entity.Vault, error) {
	authID := strings.TrimSpace(asString(credential["auth_id"]))
	authToken := strings.TrimSpace(asString(credential["auth_token"]))
	sipUsername := strings.TrimSpace(asString(credential["sip_username"]))
	sipPassword := strings.TrimSpace(asString(credential["sip_password"]))
	trunkName := strings.TrimSpace(asString(credential["trunk_name"]))
	if trunkName == "" {
		trunkName = name
	}

	if authID == "" || authToken == "" {
		return nil, fmt.Errorf("vobiz auth_id and auth_token are required")
	}
	if sipUsername == "" {
		return nil, fmt.Errorf("vobiz sip_username is required")
	}
	if sipPassword == "" {
		generated, err := generatePassword()
		if err != nil {
			return nil, fmt.Errorf("failed to generate sip password: %w", err)
		}
		sipPassword = generated
	} else if len(sipPassword) < 8 {
		return nil, fmt.Errorf("vobiz sip_password must be at least 8 characters")
	}

	// Idempotency: reuse an existing trunk provisioned for the same name.
	if existing, err := vs.findExistingVobizSIP(ctx, auth, trunkName); err == nil && existing != nil {
		vs.logger.Debugf("vobiz trunk %q already provisioned, reusing existing sip credential", trunkName)
		return existing, nil
	}

	client := newVobizClient()

	// 1. Create a standalone account-level SIP credential (returns credential_uuid).
	cred, err := client.CreateCredential(ctx, authID, authToken, vobiz.CreateCredentialRequest{
		Username: sipUsername,
		Password: sipPassword,
		Enabled:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("vobiz: failed to create credential: %w", err)
	}

	// 2. Create the trunk and ATTACH the credential via credential_uuid, so the
	//    SIP server can authenticate the trunk's INVITEs.
	trunk, err := client.CreateTrunk(ctx, authID, authToken, vobiz.CreateTrunkRequest{
		Name:           trunkName,
		TrunkStatus:    "enabled",
		TrunkDirection: "outbound",
		Transport:      "udp",
		CredentialUUID: cred.ID,
	})
	if err != nil {
		// Roll back the orphaned credential; do not mask the original error.
		if delErr := client.DeleteCredential(ctx, authID, authToken, cred.ID); delErr != nil {
			vs.logger.Errorf("vobiz: orphaned credential could not be rolled back: %v", delErr)
		}
		return nil, fmt.Errorf("vobiz: failed to create trunk after credential creation: %w", err)
	}

	// Store a single "vobiz_sip" credential carrying BOTH the SIP connection
	// fields the SIP engine reads (sip_uri/sip_username/sip_password) and the
	// vobiz management fields. The assistant-api routes telephony_provider
	// "vobiz_sip" through the SIP engine, which reads these keys directly.
	vobizSIPValue := map[string]interface{}{
		"sip_uri":       trunk.TrunkDomain,
		"sip_username":  sipUsername,
		"sip_password":  sipPassword,
		"auth_id":       authID,
		"auth_token":    authToken,
		"trunk_name":    trunkName,
		"trunk_id":      trunk.TrunkID,
		"trunk_domain":  trunk.TrunkDomain,
		"credential_id": cred.ID,
	}

	db := vs.postgres.DB(ctx)
	row := vs.newVault(auth, VobizProvider, trunkName, vobizSIPValue)
	if err := db.Save(row).Error; err != nil {
		vs.logger.Errorf("vobiz: trunk provisioned (trunk_id=%s) but persisting vault row failed: %v", trunk.TrunkID, err)
		return nil, fmt.Errorf("vobiz: trunk provisioned but failed to store credential: %w", err)
	}

	return row, nil
}

// findExistingVobizSIP returns the existing "vobiz_sip" vault row for a
// previously provisioned trunk with the given name, or nil if none exists.
func (vs *vaultService) findExistingVobizSIP(ctx context.Context, auth types.SimplePrinciple, trunkName string) (*internal_entity.Vault, error) {
	db := vs.postgres.DB(ctx)
	var rows []*internal_entity.Vault
	if err := db.Where("provider = ? AND organization_id = ? AND project_id = ? AND status = ?",
		VobizProvider,
		*auth.GetCurrentOrganizationId(),
		*auth.GetCurrentProjectId(),
		type_enums.RECORD_ACTIVE,
	).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		if asString(row.Value["trunk_name"]) == trunkName {
			return row, nil
		}
	}
	return nil, nil
}

// newVault builds a Vault entity mirroring Create()'s construction.
func (vs *vaultService) newVault(auth types.SimplePrinciple, provider, name string, value map[string]interface{}) *internal_entity.Vault {
	return &internal_entity.Vault{
		Mutable: gorm_models.Mutable{
			CreatedBy: *auth.GetUserId(),
		},
		Organizational: gorm_models.Organizational{
			OrganizationId: *auth.GetCurrentOrganizationId(),
			ProjectId:      *auth.GetCurrentProjectId(),
		},
		Name:     name,
		Provider: provider,
		Value:    value,
	}
}

func asString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// generatePassword returns a 32-char crypto-random hex string (>= vobiz's 8 min).
func generatePassword() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
