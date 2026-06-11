// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_assistant_service

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/rapidaai/api/assistant-api/config"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/clients/vobiz"
	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// InboundProvisioningError marks a user-facing Vobiz inbound-provisioning
// failure whose message is safe to surface to the UI (e.g. "number already
// attached"). The deployment handler surfaces this type's message and keeps its
// generic message for every other error — so non-vobiz flows are unchanged.
type InboundProvisioningError struct {
	Message string
	Err     error // underlying cause — preserved for logs / errors.Unwrap (UI sees Message only)
}

func (e *InboundProvisioningError) Error() string { return e.Message }
func (e *InboundProvisioningError) Unwrap() error  { return e.Err }

type assistantDeploymentService struct {
	logger   commons.Logger
	postgres connectors.PostgresConnector
	cfg      *config.AssistantConfig
}

func NewAssistantDeploymentService(cfg *config.AssistantConfig,
	logger commons.Logger,
	postgres connectors.PostgresConnector) internal_services.AssistantDeploymentService {
	return &assistantDeploymentService{
		logger:   logger,
		postgres: postgres,
		cfg:      cfg,
	}
}

func (eService assistantDeploymentService) CreateWebPluginDeployment(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	greeting, mistake *string,
	IdleTimeout *uint64,
	IdleTimeoutBackoff *uint64,
	IdleTimeoutMessage *string, maxSessionDuration *uint64,
	suggestion []string,
	inputAudio, outputAudio *protos.DeploymentAudioProvider,
) (*internal_assistant_entity.AssistantWebPluginDeployment, error) {
	db := eService.postgres.DB(ctx)
	deployment := &internal_assistant_entity.AssistantWebPluginDeployment{
		AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
			AssistantDeployment: internal_assistant_entity.AssistantDeployment{
				Mutable: gorm_models.Mutable{
					CreatedBy: *auth.GetUserId(),
					Status:    type_enums.RECORD_ACTIVE,
				},
				AssistantId: assistantId,
			},
			Greeting:           greeting,
			Mistake:            mistake,
			IdleTimeout:        IdleTimeout,
			IdleTimeoutBackoff: IdleTimeoutBackoff,
			IdleTimeoutMessage: IdleTimeoutMessage,
			MaxSessionDuration: maxSessionDuration,
		},
		Suggestion: suggestion,
	}

	if err := eService.archiveDeploymentRecords(ctx, db, &internal_assistant_entity.AssistantWebPluginDeployment{}, assistantId, *auth.GetUserId()); err != nil {
		return nil, err
	}

	tx := db.Create(deployment)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create web plugin deployment for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}

	//
	if inputAudio != nil {
		eService.createAssistantDeploymentAudio(ctx, auth, deployment.Id, "input", inputAudio)
	}
	if outputAudio != nil {
		eService.createAssistantDeploymentAudio(ctx, auth, deployment.Id, "output", outputAudio)
	}

	return deployment, nil
}

func (eService assistantDeploymentService) createAssistantDeploymentAudio(
	ctx context.Context,
	auth types.SimplePrinciple, deploymentId uint64,
	audioType string,
	audioConfig *protos.DeploymentAudioProvider) (*internal_assistant_entity.AssistantDeploymentAudio, error) {
	db := eService.postgres.DB(ctx)
	deployment := &internal_assistant_entity.AssistantDeploymentAudio{
		Mutable: gorm_models.Mutable{
			CreatedBy: *auth.GetUserId(),
			Status:    type_enums.RecordState(audioConfig.GetStatus()),
		},
		AudioType:             audioType,
		AssistantDeploymentId: deploymentId,
		AudioProvider:         audioConfig.GetAudioProvider(),
	}

	tx := db.Create(deployment)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create deployment audio config for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}

	if len(audioConfig.GetAudioOptions()) == 0 {
		return deployment, nil
	}
	audioDeploymentOptions := make([]*internal_assistant_entity.AssistantDeploymentAudioOption, 0)
	for _, v := range audioConfig.GetAudioOptions() {
		audioDeploymentOptions = append(audioDeploymentOptions, &internal_assistant_entity.AssistantDeploymentAudioOption{
			AssistantDeploymentAudioId: deployment.Id,
			Mutable: gorm_models.Mutable{
				CreatedBy: *auth.GetUserId(),
				UpdatedBy: *auth.GetUserId(),
				Status:    type_enums.RecordState(audioConfig.GetStatus()),
			},
			Metadata: gorm_models.Metadata{
				Key:   v.GetKey(),
				Value: v.GetValue(),
			},
		})
	}
	tx = db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "assistant_deployment_audio_id"}, {Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value",
			"updated_by"}),
	}).Create(audioDeploymentOptions)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create deployment audio config metadata for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}
	return deployment, nil
}

func (eService assistantDeploymentService) CreateDebuggerDeployment(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	greeting, mistake *string,
	IdleTimeout *uint64,
	IdleTimeoutBackoff *uint64,
	IdleTimeoutMessage *string, maxSessionDuration *uint64,
	inputAudio, outputAudio *protos.DeploymentAudioProvider,
) (*internal_assistant_entity.AssistantDebuggerDeployment, error) {
	db := eService.postgres.DB(ctx)
	deployment := &internal_assistant_entity.AssistantDebuggerDeployment{
		AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
			AssistantDeployment: internal_assistant_entity.AssistantDeployment{
				Mutable: gorm_models.Mutable{
					CreatedBy: *auth.GetUserId(),
					Status:    type_enums.RECORD_ACTIVE,
				},
				AssistantId: assistantId,
			},
			Greeting:           greeting,
			Mistake:            mistake,
			IdleTimeout:        IdleTimeout,
			IdleTimeoutBackoff: IdleTimeoutBackoff,
			IdleTimeoutMessage: IdleTimeoutMessage,
			MaxSessionDuration: maxSessionDuration,
		},
	}

	if err := eService.archiveDeploymentRecords(ctx, db, &internal_assistant_entity.AssistantDebuggerDeployment{}, assistantId, *auth.GetUserId()); err != nil {
		return nil, err
	}

	tx := db.Create(deployment)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create web plugin deployment for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}
	if inputAudio != nil {
		eService.createAssistantDeploymentAudio(ctx, auth, deployment.Id, "input", inputAudio)
	}
	if outputAudio != nil {
		eService.createAssistantDeploymentAudio(ctx, auth, deployment.Id, "output", outputAudio)
	}

	return deployment, nil
}

func (eService assistantDeploymentService) CreateApiDeployment(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	greeting, mistake *string,
	IdleTimeout *uint64,
	IdleTimeoutBackoff *uint64,
	IdleTimeoutMessage *string, maxSessionDuration *uint64,
	inputAudio, outputAudio *protos.DeploymentAudioProvider,
) (*internal_assistant_entity.AssistantApiDeployment, error) {
	db := eService.postgres.DB(ctx)
	deployment := &internal_assistant_entity.AssistantApiDeployment{
		AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
			AssistantDeployment: internal_assistant_entity.AssistantDeployment{
				Mutable: gorm_models.Mutable{
					CreatedBy: *auth.GetUserId(),
					Status:    type_enums.RECORD_ACTIVE,
				},
				AssistantId: assistantId,
			},
			Greeting:           greeting,
			Mistake:            mistake,
			IdleTimeout:        IdleTimeout,
			IdleTimeoutBackoff: IdleTimeoutBackoff,
			IdleTimeoutMessage: IdleTimeoutMessage,
			MaxSessionDuration: maxSessionDuration,
		},
	}

	if err := eService.archiveDeploymentRecords(ctx, db, &internal_assistant_entity.AssistantApiDeployment{}, assistantId, *auth.GetUserId()); err != nil {
		return nil, err
	}

	tx := db.Create(deployment)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create web plugin deployment for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}
	if inputAudio != nil {
		eService.createAssistantDeploymentAudio(ctx, auth, deployment.Id, "input", inputAudio)
	}
	if outputAudio != nil {
		eService.createAssistantDeploymentAudio(ctx, auth, deployment.Id, "output", outputAudio)
	}

	return deployment, nil
}

func (eService assistantDeploymentService) CreateWhatsappDeployment(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	greeting, mistake *string,
	idleTimeout *uint64,
	idleTimeoutBackoff *uint64,
	idleTimeoutMessage *string, maxSessionDuration *uint64,
	whatsappProvider string,
	whatsappOptions []*protos.Metadata,
) (*internal_assistant_entity.AssistantWhatsappDeployment, error) {
	db := eService.postgres.DB(ctx)
	deployment := &internal_assistant_entity.AssistantWhatsappDeployment{
		AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
			AssistantDeployment: internal_assistant_entity.AssistantDeployment{
				Mutable: gorm_models.Mutable{
					CreatedBy: *auth.GetUserId(),
					Status:    type_enums.RECORD_ACTIVE,
				},
				AssistantId: assistantId,
			},
			Greeting:           greeting,
			Mistake:            mistake,
			IdleTimeout:        idleTimeout,
			IdleTimeoutBackoff: idleTimeoutBackoff,
			IdleTimeoutMessage: idleTimeoutMessage,
			MaxSessionDuration: maxSessionDuration,
		},
		AssistantDeploymentWhatsapp: internal_assistant_entity.AssistantDeploymentWhatsapp{
			WhatsappProvider: whatsappProvider,
		},
	}

	if err := eService.archiveDeploymentRecords(ctx, db, &internal_assistant_entity.AssistantWhatsappDeployment{}, assistantId, *auth.GetUserId()); err != nil {
		return nil, err
	}

	// TODO: Persist the deployment to the database
	tx := db.Create(deployment)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create web plugin deployment for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}

	if len(whatsappOptions) == 0 {
		return deployment, nil
	}

	whatsappOpts := make([]*internal_assistant_entity.AssistantDeploymentWhatsappOption, 0)
	for _, v := range whatsappOptions {
		whatsappOpts = append(whatsappOpts, &internal_assistant_entity.AssistantDeploymentWhatsappOption{
			AssistantDeploymentWhatsappId: deployment.Id,
			Mutable: gorm_models.Mutable{
				CreatedBy: *auth.GetUserId(),
				UpdatedBy: *auth.GetUserId(),
				Status:    type_enums.RECORD_ACTIVE,
			},
			Metadata: gorm_models.Metadata{
				Key:   v.GetKey(),
				Value: v.GetValue(),
			},
		})
	}
	tx = db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "assistant_deployment_whatsapp_id"}, {Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value",
			"updated_by"}),
	}).Create(whatsappOpts)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create whatsapp options for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}
	return deployment, nil
}

func (eService assistantDeploymentService) CreatePhoneDeployment(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	greeting, mistake *string,
	IdleTimeout *uint64,
	IdleTimeoutBackoff *uint64,
	IdleTimeoutMessage *string, maxSessionDuration *uint64,
	phoneProvider string,
	inputAudio, outputAudio *protos.DeploymentAudioProvider,
	opts []*protos.Metadata,
) (*internal_assistant_entity.AssistantPhoneDeployment, error) {
	db := eService.postgres.DB(ctx)

	// Auto-provision Vobiz inbound (origination URI -> inbound trunk -> assign
	// DID) when a vobiz_sip deployment enables inbound calls. Done before any DB
	// write so a provisioning failure surfaces to the UI without leaving a
	// partial deployment; returns extra options (inbound_trunk_id/uri_id).
	inboundOpts, perr := eService.maybeProvisionVobizInbound(ctx, auth, assistantId, phoneProvider, opts)
	if perr != nil {
		return nil, perr
	}
	opts = append(opts, inboundOpts...)

	deployment := &internal_assistant_entity.AssistantPhoneDeployment{
		AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
			AssistantDeployment: internal_assistant_entity.AssistantDeployment{
				Mutable: gorm_models.Mutable{
					CreatedBy: *auth.GetUserId(),
					Status:    type_enums.RECORD_ACTIVE,
				},
				AssistantId: assistantId,
			},
			Greeting:           greeting,
			Mistake:            mistake,
			IdleTimeout:        IdleTimeout,
			IdleTimeoutBackoff: IdleTimeoutBackoff,
			IdleTimeoutMessage: IdleTimeoutMessage,
			MaxSessionDuration: maxSessionDuration,
		},
		AssistantDeploymentTelephony: internal_assistant_entity.AssistantDeploymentTelephony{
			TelephonyProvider: phoneProvider,
		},
	}

	if err := eService.archiveDeploymentRecords(ctx, db, &internal_assistant_entity.AssistantPhoneDeployment{}, assistantId, *auth.GetUserId()); err != nil {
		return nil, err
	}

	tx := db.Create(deployment)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create web plugin deployment for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}

	if inputAudio != nil {
		eService.createAssistantDeploymentAudio(ctx, auth, deployment.Id, "input", inputAudio)
	}
	if outputAudio != nil {
		eService.createAssistantDeploymentAudio(ctx, auth, deployment.Id, "output", outputAudio)
	}

	if len(opts) == 0 {
		eService.logger.Warnf("no options for the telephony provider.")
		return deployment, nil
	}

	phoneOpts := make([]*internal_assistant_entity.AssistantDeploymentTelephonyOption, 0)
	for _, v := range opts {
		phoneOpts = append(phoneOpts, &internal_assistant_entity.AssistantDeploymentTelephonyOption{
			AssistantDeploymentTelephonyId: deployment.Id,
			Mutable: gorm_models.Mutable{
				CreatedBy: *auth.GetUserId(),
				UpdatedBy: *auth.GetUserId(),
				Status:    type_enums.RECORD_ACTIVE,
			},
			Metadata: gorm_models.Metadata{
				Key:   v.GetKey(),
				Value: v.GetValue(),
			},
		})
	}

	tx = db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "assistant_deployment_telephony_id"}, {Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value",
			"updated_by"}),
	}).Create(phoneOpts)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create telephony options for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}

	return deployment, nil
}

// maybeProvisionVobizInbound provisions Vobiz inbound routing when a vobiz_sip
// deployment enables inbound calls: it creates an origination URI pointing at
// this assistant-api's SIP server (cfg.SIPConfig.ExternalIP:Port), an inbound
// trunk referencing it, and assigns the DID. Returns extra telephony options
// (inbound_trunk_id / inbound_uri_id) to persist. Non-vobiz / non-inbound paths
// early-return (nil, nil), so they are byte-for-byte unchanged.
func (eService assistantDeploymentService) maybeProvisionVobizInbound(
	ctx context.Context, auth types.SimplePrinciple, assistantId uint64,
	phoneProvider string, opts []*protos.Metadata,
) ([]*protos.Metadata, error) {
	if phoneProvider != "vobiz_sip" {
		return nil, nil
	}
	optMap := make(map[string]string, len(opts))
	for _, o := range opts {
		optMap[o.GetKey()] = o.GetValue()
	}
	if optMap["rapida.sip_inbound"] != "true" {
		return nil, nil
	}

	did := optMap["phone"]
	if did == "" {
		return nil, &InboundProvisioningError{Message: "a caller-ID phone number is required to enable inbound calls"}
	}
	credIDStr := optMap["rapida.credential_id"]
	if credIDStr == "" {
		return nil, &InboundProvisioningError{Message: "a Vobiz SIP credential is required to enable inbound calls"}
	}
	credentialID, err := strconv.ParseUint(credIDStr, 10, 64)
	if err != nil {
		return nil, &InboundProvisioningError{Message: "invalid Vobiz SIP credential", Err: err}
	}

	// Idempotency: reuse an existing inbound trunk for the same DID if a prior
	// deployment already provisioned one (avoids duplicate Vobiz trunks on re-save).
	if trunkID, uriID := eService.existingInboundTrunk(ctx, assistantId, did); trunkID != "" {
		eService.logger.Infof("vobiz inbound: reusing existing trunk %s for DID %s", trunkID, did)
		return inboundMetadata(trunkID, uriID), nil
	}

	if eService.cfg.SIPConfig == nil || eService.cfg.SIPConfig.ExternalIP == "" {
		return nil, &InboundProvisioningError{Message: "the assistant SIP server has no external IP configured; cannot provision inbound"}
	}

	// Read the Vobiz account credentials via a redis-free, one-shot gRPC fetch.
	cred, err := web_client.GetCredentialDirect(&eService.cfg.AppConfig, eService.logger, auth, credentialID)
	if err != nil {
		eService.logger.Errorf("vobiz inbound: failed to read credential %d: %v", credentialID, err)
		return nil, &InboundProvisioningError{Message: "could not read the Vobiz SIP credential", Err: err}
	}
	value := cred.GetValue().AsMap()
	authID, idOK := value["auth_id"].(string)
	authToken, tokenOK := value["auth_token"].(string)
	if !idOK || !tokenOK || authID == "" || authToken == "" {
		return nil, &InboundProvisioningError{Message: "the Vobiz SIP credential is missing auth_id/auth_token"}
	}

	sipAddr := fmt.Sprintf("%s:%d", eService.cfg.SIPConfig.ExternalIP, eService.cfg.SIPConfig.Port)
	trunkID, uriID, err := eService.provisionVobizInbound(ctx, authID, authToken, sipAddr, did)
	if err != nil {
		return nil, err // already an *InboundProvisioningError
	}
	return inboundMetadata(trunkID, uriID), nil
}

// provisionVobizInbound performs the three Vobiz API calls and rolls back the
// trunk if number assignment fails.
func (eService assistantDeploymentService) provisionVobizInbound(
	ctx context.Context, authID, authToken, sipAddr, did string,
) (string, string, error) {
	client := vobiz.NewClient()

	uri, err := client.CreateOriginationURI(ctx, authID, authToken, vobiz.CreateOriginationURIRequest{
		URI: sipAddr, Transport: "udp", Priority: 1, Weight: 10, Enabled: true,
	})
	if err != nil {
		return "", "", &InboundProvisioningError{Message: "failed to create the Vobiz origination URI", Err: err}
	}

	trunk, err := client.CreateTrunk(ctx, authID, authToken, vobiz.CreateTrunkRequest{
		Name:                 fmt.Sprintf("rapida-inbound-%s", did),
		TrunkStatus:          "enabled",
		TrunkDirection:       "inbound",
		Transport:            "udp",
		ConcurrentCallsLimit: 10,
		CpsLimit:             2,
		PrimaryURIUUID:       uri.ID,
		InboundDestination:   uri.ID,
	})
	if err != nil {
		return "", "", &InboundProvisioningError{Message: "failed to create the Vobiz inbound trunk", Err: err}
	}

	if err := client.AssignNumber(ctx, authID, authToken, did, trunk.TrunkID); err != nil {
		// Roll back the just-created trunk; surface a user-friendly message.
		if delErr := client.DeleteTrunk(ctx, authID, authToken, trunk.TrunkID); delErr != nil {
			eService.logger.Errorf("vobiz inbound: failed to roll back trunk %s: %v", trunk.TrunkID, delErr)
		}
		return "", "", vobizAssignError(did, err)
	}
	return trunk.TrunkID, uri.ID, nil
}

// existingInboundTrunk returns the inbound trunk/URI ids already provisioned for
// the given DID on an active vobiz_sip deployment of this assistant, if any.
func (eService assistantDeploymentService) existingInboundTrunk(ctx context.Context, assistantId uint64, did string) (string, string) {
	var deployments []internal_assistant_entity.AssistantPhoneDeployment
	if err := eService.postgres.DB(ctx).Preload("TelephonyOption").
		Where("assistant_id = ? AND telephony_provider = ? AND status = ?", assistantId, "vobiz_sip", type_enums.RECORD_ACTIVE).
		Find(&deployments).Error; err != nil {
		// Log instead of silently swallowing: a query failure here could otherwise
		// look like "no existing trunk" and cause a duplicate provisioning attempt.
		eService.logger.Errorf("vobiz inbound: idempotency lookup failed for assistant %d, did %s: %v", assistantId, did, err)
		return "", ""
	}
	for _, dep := range deployments {
		o := dep.GetOptions()
		phone, _ := o.GetString("phone")
		trunkID, _ := o.GetString("rapida.inbound_trunk_id")
		uriID, _ := o.GetString("rapida.inbound_uri_id")
		if phone == did && trunkID != "" {
			return trunkID, uriID
		}
	}
	return "", ""
}

func inboundMetadata(trunkID, uriID string) []*protos.Metadata {
	return []*protos.Metadata{
		{Key: "rapida.inbound_trunk_id", Value: trunkID},
		{Key: "rapida.inbound_uri_id", Value: uriID},
	}
}

// vobizAssignError maps a Vobiz number-assignment failure to a clear UI message.
func vobizAssignError(did string, err error) error {
	var apiErr *vobiz.VobizAPIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 404:
			return &InboundProvisioningError{Message: fmt.Sprintf("phone number %s is not in your Vobiz account", did), Err: err}
		case 400, 409:
			return &InboundProvisioningError{Message: fmt.Sprintf("phone number %s is already attached to another Vobiz trunk — unlink it first", did), Err: err}
		}
	}
	return &InboundProvisioningError{Message: fmt.Sprintf("failed to attach %s to the Vobiz inbound trunk", did), Err: err}
}

func (eService assistantDeploymentService) GetAssistantApiDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantApiDeployment, error) {
	db := eService.postgres.DB(ctx)
	var apiDeployment *internal_assistant_entity.AssistantApiDeployment
	qry := db.
		Preload("InputAudio", "audio_type = ?", "input").
		Preload("InputAudio.AudioOptions").
		Preload("OutputAudio", "audio_type = ?", "output").
		Preload("OutputAudio.AudioOptions").
		Where("assistant_id = ? AND status IN ?", assistantId, []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_INACTIVE})
	tx := qry.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "created_date"},
		Desc:   true,
	}).First(&apiDeployment)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if tx.Error != nil {
		eService.logger.Errorf("not able to find api deployment for the assistant %d  with error %v", assistantId, tx.Error)
		return nil, tx.Error
	}
	return apiDeployment, nil
}
func (eService assistantDeploymentService) GetAssistantDebuggerDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantDebuggerDeployment, error) {
	db := eService.postgres.DB(ctx)
	var debuggerDeployment *internal_assistant_entity.AssistantDebuggerDeployment
	qry := db.
		Preload("InputAudio", "audio_type = ?", "input").
		Preload("InputAudio.AudioOptions").
		Preload("OutputAudio", "audio_type = ?", "output").
		Preload("OutputAudio.AudioOptions").
		Where("assistant_id = ? AND status IN ?", assistantId, []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_INACTIVE})
	tx := qry.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "created_date"},
		Desc:   true,
	}).First(&debuggerDeployment)

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if tx.Error != nil {
		eService.logger.Errorf("not able to find api deployment for the assistant %d  with error %v", assistantId, tx.Error)
		return nil, tx.Error
	}
	return debuggerDeployment, nil
}
func (eService assistantDeploymentService) GetAssistantPhoneDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantPhoneDeployment, error) {
	db := eService.postgres.DB(ctx)
	var phoneDeployment *internal_assistant_entity.AssistantPhoneDeployment
	qry := db.
		Preload("TelephonyOption").
		Preload("InputAudio", "audio_type = ?", "input").
		Preload("InputAudio.AudioOptions").
		Preload("OutputAudio", "audio_type = ?", "output").
		Preload("OutputAudio.AudioOptions").
		Where("assistant_id = ? AND status IN ?", assistantId, []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_INACTIVE})
	tx := qry.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "created_date"},
		Desc:   true,
	}).First(&phoneDeployment)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if tx.Error != nil {
		eService.logger.Errorf("not able to find api deployment for the assistant %d  with error %v", assistantId, tx.Error)
		return nil, tx.Error
	}
	return phoneDeployment, nil
}
func (eService assistantDeploymentService) GetAssistantWebpluginDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantWebPluginDeployment, error) {
	db := eService.postgres.DB(ctx)
	var webPluginDeployment *internal_assistant_entity.AssistantWebPluginDeployment
	qry := db.
		Preload("InputAudio", "audio_type = ?", "input").
		Preload("InputAudio.AudioOptions").
		Preload("OutputAudio", "audio_type = ?", "output").
		Preload("OutputAudio.AudioOptions").
		Where("assistant_id = ? AND status IN ?", assistantId, []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_INACTIVE})
	tx := qry.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "created_date"},
		Desc:   true,
	}).First(&webPluginDeployment)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if tx.Error != nil {
		eService.logger.Errorf("not able to find web plugin deployment for the assistant %d  with error %v", assistantId, tx.Error)
		return nil, tx.Error
	}
	return webPluginDeployment, nil
}
func (eService assistantDeploymentService) GetAssistantWhatsappDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantWhatsappDeployment, error) {
	db := eService.postgres.DB(ctx)
	var whatsappDeployment *internal_assistant_entity.AssistantWhatsappDeployment
	qry := db.
		Preload("WhatsappOptions").
		Where("assistant_id = ? AND status IN ?", assistantId, []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_INACTIVE})
	tx := qry.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "created_date"},
		Desc:   true,
	}).First(&whatsappDeployment)

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if tx.Error != nil {
		eService.logger.Errorf("not able to find whatsapp deployment for the assistant %d  with error %v", assistantId, tx.Error)
		return nil, tx.Error
	}
	return whatsappDeployment, nil
}

func (eService assistantDeploymentService) GetAllAssistantApiDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64, criterias []*protos.Criteria, paginate *protos.Paginate) (int64, []*internal_assistant_entity.AssistantApiDeployment, error) {
	db := eService.postgres.DB(ctx)
	var (
		deployments []*internal_assistant_entity.AssistantApiDeployment
		cnt         int64
	)
	qry := db.Model(&internal_assistant_entity.AssistantApiDeployment{}).
		Where("assistant_id = ?", assistantId)
	for _, ct := range criterias {
		qry = qry.Where(fmt.Sprintf("%s %s ?", ct.GetKey(), ct.GetLogic()), ct.GetValue())
	}
	tx := qry.
		Preload("InputAudio", "audio_type = ?", "input").
		Preload("InputAudio.AudioOptions").
		Preload("OutputAudio", "audio_type = ?", "output").
		Preload("OutputAudio.AudioOptions").
		Scopes(gorm_models.Paginate(gorm_models.NewPaginated(
			int(paginate.GetPage()),
			int(paginate.GetPageSize()),
			&cnt,
			qry,
		))).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "created_date"},
			Desc:   true,
		}).
		Find(&deployments)
	if tx.Error != nil {
		eService.logger.Errorf("not able to list api deployments for assistant %d with error %v", assistantId, tx.Error)
		return cnt, nil, tx.Error
	}
	return cnt, deployments, nil
}

func (eService assistantDeploymentService) GetAllAssistantDebuggerDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64, criterias []*protos.Criteria, paginate *protos.Paginate) (int64, []*internal_assistant_entity.AssistantDebuggerDeployment, error) {
	db := eService.postgres.DB(ctx)
	var (
		deployments []*internal_assistant_entity.AssistantDebuggerDeployment
		cnt         int64
	)
	qry := db.Model(&internal_assistant_entity.AssistantDebuggerDeployment{}).
		Where("assistant_id = ?", assistantId)
	for _, ct := range criterias {
		qry = qry.Where(fmt.Sprintf("%s %s ?", ct.GetKey(), ct.GetLogic()), ct.GetValue())
	}
	tx := qry.
		Preload("InputAudio", "audio_type = ?", "input").
		Preload("InputAudio.AudioOptions").
		Preload("OutputAudio", "audio_type = ?", "output").
		Preload("OutputAudio.AudioOptions").
		Scopes(gorm_models.Paginate(gorm_models.NewPaginated(
			int(paginate.GetPage()),
			int(paginate.GetPageSize()),
			&cnt,
			qry,
		))).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "created_date"},
			Desc:   true,
		}).
		Find(&deployments)
	if tx.Error != nil {
		eService.logger.Errorf("not able to list debugger deployments for assistant %d with error %v", assistantId, tx.Error)
		return cnt, nil, tx.Error
	}
	return cnt, deployments, nil
}

func (eService assistantDeploymentService) GetAllAssistantPhoneDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64, criterias []*protos.Criteria, paginate *protos.Paginate) (int64, []*internal_assistant_entity.AssistantPhoneDeployment, error) {
	db := eService.postgres.DB(ctx)
	var (
		deployments []*internal_assistant_entity.AssistantPhoneDeployment
		cnt         int64
	)
	qry := db.Model(&internal_assistant_entity.AssistantPhoneDeployment{}).
		Where("assistant_id = ?", assistantId)
	for _, ct := range criterias {
		qry = qry.Where(fmt.Sprintf("%s %s ?", ct.GetKey(), ct.GetLogic()), ct.GetValue())
	}
	tx := qry.
		Preload("TelephonyOption").
		Preload("InputAudio", "audio_type = ?", "input").
		Preload("InputAudio.AudioOptions").
		Preload("OutputAudio", "audio_type = ?", "output").
		Preload("OutputAudio.AudioOptions").
		Scopes(gorm_models.Paginate(gorm_models.NewPaginated(
			int(paginate.GetPage()),
			int(paginate.GetPageSize()),
			&cnt,
			qry,
		))).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "created_date"},
			Desc:   true,
		}).
		Find(&deployments)
	if tx.Error != nil {
		eService.logger.Errorf("not able to list phone deployments for assistant %d with error %v", assistantId, tx.Error)
		return cnt, nil, tx.Error
	}
	return cnt, deployments, nil
}

func (eService assistantDeploymentService) GetAllAssistantWebpluginDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64, criterias []*protos.Criteria, paginate *protos.Paginate) (int64, []*internal_assistant_entity.AssistantWebPluginDeployment, error) {
	db := eService.postgres.DB(ctx)
	var (
		deployments []*internal_assistant_entity.AssistantWebPluginDeployment
		cnt         int64
	)
	qry := db.Model(&internal_assistant_entity.AssistantWebPluginDeployment{}).
		Where("assistant_id = ?", assistantId)
	for _, ct := range criterias {
		qry = qry.Where(fmt.Sprintf("%s %s ?", ct.GetKey(), ct.GetLogic()), ct.GetValue())
	}
	tx := qry.
		Preload("InputAudio", "audio_type = ?", "input").
		Preload("InputAudio.AudioOptions").
		Preload("OutputAudio", "audio_type = ?", "output").
		Preload("OutputAudio.AudioOptions").
		Scopes(gorm_models.Paginate(gorm_models.NewPaginated(
			int(paginate.GetPage()),
			int(paginate.GetPageSize()),
			&cnt,
			qry,
		))).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "created_date"},
			Desc:   true,
		}).
		Find(&deployments)
	if tx.Error != nil {
		eService.logger.Errorf("not able to list webplugin deployments for assistant %d with error %v", assistantId, tx.Error)
		return cnt, nil, tx.Error
	}
	return cnt, deployments, nil
}

func (eService assistantDeploymentService) GetAllAssistantWhatsappDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64, criterias []*protos.Criteria, paginate *protos.Paginate) (int64, []*internal_assistant_entity.AssistantWhatsappDeployment, error) {
	db := eService.postgres.DB(ctx)
	var (
		deployments []*internal_assistant_entity.AssistantWhatsappDeployment
		cnt         int64
	)
	qry := db.Model(&internal_assistant_entity.AssistantWhatsappDeployment{}).
		Where("assistant_id = ?", assistantId)
	for _, ct := range criterias {
		qry = qry.Where(fmt.Sprintf("%s %s ?", ct.GetKey(), ct.GetLogic()), ct.GetValue())
	}
	tx := qry.
		Preload("WhatsappOptions").
		Scopes(gorm_models.Paginate(gorm_models.NewPaginated(
			int(paginate.GetPage()),
			int(paginate.GetPageSize()),
			&cnt,
			qry,
		))).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "created_date"},
			Desc:   true,
		}).
		Find(&deployments)
	if tx.Error != nil {
		eService.logger.Errorf("not able to list whatsapp deployments for assistant %d with error %v", assistantId, tx.Error)
		return cnt, nil, tx.Error
	}
	return cnt, deployments, nil
}

func (eService assistantDeploymentService) DisableAssistantApiDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantApiDeployment, error) {
	db := eService.postgres.DB(ctx)
	var out *internal_assistant_entity.AssistantApiDeployment
	err := db.Transaction(func(tx *gorm.DB) error {
		var current *internal_assistant_entity.AssistantApiDeployment
		getTx := tx.
			Preload("InputAudio", "audio_type = ?", "input").
			Preload("InputAudio.AudioOptions").
			Preload("OutputAudio", "audio_type = ?", "output").
			Preload("OutputAudio.AudioOptions").
			Where("assistant_id = ? AND status IN ?", assistantId, []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_INACTIVE}).
			Order(clause.OrderByColumn{Column: clause.Column{Name: "created_date"}, Desc: true}).
			First(&current)
		if errors.Is(getTx.Error, gorm.ErrRecordNotFound) {
			return nil
		}
		if getTx.Error != nil {
			return getTx.Error
		}

		if err := eService.archiveDeploymentRecords(ctx, tx, &internal_assistant_entity.AssistantApiDeployment{}, assistantId, *auth.GetUserId()); err != nil {
			return err
		}

		created := &internal_assistant_entity.AssistantApiDeployment{
			AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
				AssistantDeployment: internal_assistant_entity.AssistantDeployment{
					Mutable: gorm_models.Mutable{
						CreatedBy: *auth.GetUserId(),
						UpdatedBy: *auth.GetUserId(),
						Status:    type_enums.RECORD_INACTIVE,
					},
					AssistantId: assistantId,
				},
				Greeting:           current.Greeting,
				Mistake:            current.Mistake,
				IdleTimeout:        current.IdleTimeout,
				IdleTimeoutBackoff: current.IdleTimeoutBackoff,
				IdleTimeoutMessage: current.IdleTimeoutMessage,
				MaxSessionDuration: current.MaxSessionDuration,
			},
		}
		if err := tx.Create(created).Error; err != nil {
			return err
		}
		if current.InputAudio != nil {
			_, _ = eService.createAssistantDeploymentAudio(ctx, auth, created.Id, "input", toProtoAudioProvider(current.InputAudio))
		}
		if current.OutputAudio != nil {
			_, _ = eService.createAssistantDeploymentAudio(ctx, auth, created.Id, "output", toProtoAudioProvider(current.OutputAudio))
		}
		out = created
		return nil
	})
	return out, err
}

func (eService assistantDeploymentService) DisableAssistantDebuggerDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantDebuggerDeployment, error) {
	db := eService.postgres.DB(ctx)
	var out *internal_assistant_entity.AssistantDebuggerDeployment
	err := db.Transaction(func(tx *gorm.DB) error {
		var current *internal_assistant_entity.AssistantDebuggerDeployment
		getTx := tx.
			Preload("InputAudio", "audio_type = ?", "input").
			Preload("InputAudio.AudioOptions").
			Preload("OutputAudio", "audio_type = ?", "output").
			Preload("OutputAudio.AudioOptions").
			Where("assistant_id = ? AND status IN ?", assistantId, []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_INACTIVE}).
			Order(clause.OrderByColumn{Column: clause.Column{Name: "created_date"}, Desc: true}).
			First(&current)
		if errors.Is(getTx.Error, gorm.ErrRecordNotFound) {
			return nil
		}
		if getTx.Error != nil {
			return getTx.Error
		}

		if err := eService.archiveDeploymentRecords(ctx, tx, &internal_assistant_entity.AssistantDebuggerDeployment{}, assistantId, *auth.GetUserId()); err != nil {
			return err
		}

		created := &internal_assistant_entity.AssistantDebuggerDeployment{
			AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
				AssistantDeployment: internal_assistant_entity.AssistantDeployment{
					Mutable: gorm_models.Mutable{
						CreatedBy: *auth.GetUserId(),
						UpdatedBy: *auth.GetUserId(),
						Status:    type_enums.RECORD_INACTIVE,
					},
					AssistantId: assistantId,
				},
				Greeting:           current.Greeting,
				Mistake:            current.Mistake,
				IdleTimeout:        current.IdleTimeout,
				IdleTimeoutBackoff: current.IdleTimeoutBackoff,
				IdleTimeoutMessage: current.IdleTimeoutMessage,
				MaxSessionDuration: current.MaxSessionDuration,
			},
		}
		if err := tx.Create(created).Error; err != nil {
			return err
		}
		if current.InputAudio != nil {
			_, _ = eService.createAssistantDeploymentAudio(ctx, auth, created.Id, "input", toProtoAudioProvider(current.InputAudio))
		}
		if current.OutputAudio != nil {
			_, _ = eService.createAssistantDeploymentAudio(ctx, auth, created.Id, "output", toProtoAudioProvider(current.OutputAudio))
		}
		out = created
		return nil
	})
	return out, err
}

func (eService assistantDeploymentService) DisableAssistantPhoneDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantPhoneDeployment, error) {
	db := eService.postgres.DB(ctx)
	var out *internal_assistant_entity.AssistantPhoneDeployment
	err := db.Transaction(func(tx *gorm.DB) error {
		var current *internal_assistant_entity.AssistantPhoneDeployment
		getTx := tx.
			Preload("TelephonyOption").
			Preload("InputAudio", "audio_type = ?", "input").
			Preload("InputAudio.AudioOptions").
			Preload("OutputAudio", "audio_type = ?", "output").
			Preload("OutputAudio.AudioOptions").
			Where("assistant_id = ? AND status IN ?", assistantId, []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_INACTIVE}).
			Order(clause.OrderByColumn{Column: clause.Column{Name: "created_date"}, Desc: true}).
			First(&current)
		if errors.Is(getTx.Error, gorm.ErrRecordNotFound) {
			return nil
		}
		if getTx.Error != nil {
			return getTx.Error
		}

		if err := eService.archiveDeploymentRecords(ctx, tx, &internal_assistant_entity.AssistantPhoneDeployment{}, assistantId, *auth.GetUserId()); err != nil {
			return err
		}

		created := &internal_assistant_entity.AssistantPhoneDeployment{
			AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
				AssistantDeployment: internal_assistant_entity.AssistantDeployment{
					Mutable: gorm_models.Mutable{
						CreatedBy: *auth.GetUserId(),
						UpdatedBy: *auth.GetUserId(),
						Status:    type_enums.RECORD_INACTIVE,
					},
					AssistantId: assistantId,
				},
				Greeting:           current.Greeting,
				Mistake:            current.Mistake,
				IdleTimeout:        current.IdleTimeout,
				IdleTimeoutBackoff: current.IdleTimeoutBackoff,
				IdleTimeoutMessage: current.IdleTimeoutMessage,
				MaxSessionDuration: current.MaxSessionDuration,
			},
			AssistantDeploymentTelephony: internal_assistant_entity.AssistantDeploymentTelephony{
				TelephonyProvider: current.TelephonyProvider,
			},
		}
		if err := tx.Create(created).Error; err != nil {
			return err
		}

		if current.InputAudio != nil {
			_, _ = eService.createAssistantDeploymentAudio(ctx, auth, created.Id, "input", toProtoAudioProvider(current.InputAudio))
		}
		if current.OutputAudio != nil {
			_, _ = eService.createAssistantDeploymentAudio(ctx, auth, created.Id, "output", toProtoAudioProvider(current.OutputAudio))
		}

		if len(current.TelephonyOption) > 0 {
			phoneOpts := make([]*internal_assistant_entity.AssistantDeploymentTelephonyOption, 0, len(current.TelephonyOption))
			for _, v := range current.TelephonyOption {
				phoneOpts = append(phoneOpts, &internal_assistant_entity.AssistantDeploymentTelephonyOption{
					AssistantDeploymentTelephonyId: created.Id,
					Mutable: gorm_models.Mutable{
						CreatedBy: *auth.GetUserId(),
						UpdatedBy: *auth.GetUserId(),
						Status:    type_enums.RECORD_ACTIVE,
					},
					Metadata: gorm_models.Metadata{
						Key:   v.Key,
						Value: v.Value,
					},
				})
			}
			if err := tx.Create(phoneOpts).Error; err != nil {
				return err
			}
		}
		out = created
		return nil
	})
	return out, err
}

func (eService assistantDeploymentService) DisableAssistantWebpluginDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantWebPluginDeployment, error) {
	db := eService.postgres.DB(ctx)
	var out *internal_assistant_entity.AssistantWebPluginDeployment
	err := db.Transaction(func(tx *gorm.DB) error {
		var current *internal_assistant_entity.AssistantWebPluginDeployment
		getTx := tx.
			Preload("InputAudio", "audio_type = ?", "input").
			Preload("InputAudio.AudioOptions").
			Preload("OutputAudio", "audio_type = ?", "output").
			Preload("OutputAudio.AudioOptions").
			Where("assistant_id = ? AND status IN ?", assistantId, []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_INACTIVE}).
			Order(clause.OrderByColumn{Column: clause.Column{Name: "created_date"}, Desc: true}).
			First(&current)
		if errors.Is(getTx.Error, gorm.ErrRecordNotFound) {
			return nil
		}
		if getTx.Error != nil {
			return getTx.Error
		}

		if err := eService.archiveDeploymentRecords(ctx, tx, &internal_assistant_entity.AssistantWebPluginDeployment{}, assistantId, *auth.GetUserId()); err != nil {
			return err
		}

		created := &internal_assistant_entity.AssistantWebPluginDeployment{
			AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
				AssistantDeployment: internal_assistant_entity.AssistantDeployment{
					Mutable: gorm_models.Mutable{
						CreatedBy: *auth.GetUserId(),
						UpdatedBy: *auth.GetUserId(),
						Status:    type_enums.RECORD_INACTIVE,
					},
					AssistantId: assistantId,
				},
				Greeting:           current.Greeting,
				Mistake:            current.Mistake,
				IdleTimeout:        current.IdleTimeout,
				IdleTimeoutBackoff: current.IdleTimeoutBackoff,
				IdleTimeoutMessage: current.IdleTimeoutMessage,
				MaxSessionDuration: current.MaxSessionDuration,
			},
			Suggestion: current.Suggestion,
		}
		if err := tx.Create(created).Error; err != nil {
			return err
		}
		if current.InputAudio != nil {
			_, _ = eService.createAssistantDeploymentAudio(ctx, auth, created.Id, "input", toProtoAudioProvider(current.InputAudio))
		}
		if current.OutputAudio != nil {
			_, _ = eService.createAssistantDeploymentAudio(ctx, auth, created.Id, "output", toProtoAudioProvider(current.OutputAudio))
		}
		out = created
		return nil
	})
	return out, err
}

func (eService assistantDeploymentService) DisableAssistantWhatsappDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantWhatsappDeployment, error) {
	db := eService.postgres.DB(ctx)
	var out *internal_assistant_entity.AssistantWhatsappDeployment
	err := db.Transaction(func(tx *gorm.DB) error {
		var current *internal_assistant_entity.AssistantWhatsappDeployment
		getTx := tx.
			Preload("WhatsappOptions").
			Where("assistant_id = ? AND status IN ?", assistantId, []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_INACTIVE}).
			Order(clause.OrderByColumn{Column: clause.Column{Name: "created_date"}, Desc: true}).
			First(&current)
		if errors.Is(getTx.Error, gorm.ErrRecordNotFound) {
			return nil
		}
		if getTx.Error != nil {
			return getTx.Error
		}

		if err := eService.archiveDeploymentRecords(ctx, tx, &internal_assistant_entity.AssistantWhatsappDeployment{}, assistantId, *auth.GetUserId()); err != nil {
			return err
		}

		created := &internal_assistant_entity.AssistantWhatsappDeployment{
			AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
				AssistantDeployment: internal_assistant_entity.AssistantDeployment{
					Mutable: gorm_models.Mutable{
						CreatedBy: *auth.GetUserId(),
						UpdatedBy: *auth.GetUserId(),
						Status:    type_enums.RECORD_INACTIVE,
					},
					AssistantId: assistantId,
				},
				Greeting:           current.Greeting,
				Mistake:            current.Mistake,
				IdleTimeout:        current.IdleTimeout,
				IdleTimeoutBackoff: current.IdleTimeoutBackoff,
				IdleTimeoutMessage: current.IdleTimeoutMessage,
				MaxSessionDuration: current.MaxSessionDuration,
			},
			AssistantDeploymentWhatsapp: internal_assistant_entity.AssistantDeploymentWhatsapp{
				WhatsappProvider: current.WhatsappProvider,
			},
		}
		if err := tx.Create(created).Error; err != nil {
			return err
		}
		if len(current.WhatsappOptions) > 0 {
			whatsappOpts := make([]*internal_assistant_entity.AssistantDeploymentWhatsappOption, 0, len(current.WhatsappOptions))
			for _, v := range current.WhatsappOptions {
				whatsappOpts = append(whatsappOpts, &internal_assistant_entity.AssistantDeploymentWhatsappOption{
					AssistantDeploymentWhatsappId: created.Id,
					Mutable: gorm_models.Mutable{
						CreatedBy: *auth.GetUserId(),
						UpdatedBy: *auth.GetUserId(),
						Status:    type_enums.RECORD_ACTIVE,
					},
					Metadata: gorm_models.Metadata{
						Key:   v.Key,
						Value: v.Value,
					},
				})
			}
			if err := tx.Create(whatsappOpts).Error; err != nil {
				return err
			}
		}
		out = created
		return nil
	})
	return out, err
}

func (eService assistantDeploymentService) archiveDeploymentRecords(ctx context.Context, db *gorm.DB, model interface{}, assistantId uint64, userId uint64) error {
	return db.WithContext(ctx).
		Model(model).
		Where("assistant_id = ? AND status IN ?", assistantId, []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_INACTIVE}).
		Updates(map[string]interface{}{
			"status":     type_enums.RECORD_ARCHIEVE,
			"updated_by": userId,
		}).Error
}

func toProtoAudioProvider(audio *internal_assistant_entity.AssistantDeploymentAudio) *protos.DeploymentAudioProvider {
	if audio == nil {
		return nil
	}
	opts := make([]*protos.Metadata, 0, len(audio.AudioOptions))
	for _, v := range audio.AudioOptions {
		opts = append(opts, &protos.Metadata{Key: v.Key, Value: v.Value})
	}
	return &protos.DeploymentAudioProvider{
		AudioProvider: audio.AudioProvider,
		AudioOptions:  opts,
		Status:        string(type_enums.RECORD_ACTIVE),
		AudioType:     audio.AudioType,
	}
}
