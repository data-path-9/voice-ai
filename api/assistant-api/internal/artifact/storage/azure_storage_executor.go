// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_artifact_storage

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/configs"
	storage_files "github.com/rapidaai/pkg/storages/file-storage"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
)

const (
	azureStorageOptionCredentialIDKey   = "credential_id"
	azureStorageOptionContainerKey      = "container"
	azureStorageOptionPrefixKey         = "prefix"
	azureStorageOptionTimeoutSecondsKey = "timeout_seconds"

	azureStorageDefaultArtifactPushTimeout = 30 * time.Second
)

type azureStorageExecutor struct {
	logger        commons.Logger
	configuration *internal_assistant_entity.AssistantConfiguration
	caller        internal_type.InternalCaller
	auth          types.SimplePrinciple
	onPacket      func(context.Context, ...internal_type.Packet) error
}

func NewAzureStorageExecutor(
	logger commons.Logger,
	configuration *internal_assistant_entity.AssistantConfiguration,
	caller internal_type.InternalCaller,
	auth types.SimplePrinciple,
	onPacket func(context.Context, ...internal_type.Packet) error,
) internal_type.ArtifactPushExecutor {
	return &azureStorageExecutor{
		logger:        logger,
		configuration: configuration,
		caller:        caller,
		auth:          auth,
		onPacket:      onPacket,
	}
}

func (e *azureStorageExecutor) Name() string {
	configurationName, _ := e.configuration.GetOptions().GetString("name")
	if configurationName == "" {
		configurationName = fmt.Sprintf("%d", e.configuration.Id)
	}
	return fmt.Sprintf("artifact-push-%s-%s", e.configuration.Provider, configurationName)
}

func (e *azureStorageExecutor) Options() utils.Option {
	return e.configuration.GetOptions()
}

func (e *azureStorageExecutor) Arguments() (map[string]string, error) {
	return map[string]string{}, nil
}

func (e *azureStorageExecutor) Close(context.Context) error {
	return nil
}

func (e *azureStorageExecutor) Execute(ctx context.Context, input internal_type.ArtifactPushInput) (internal_type.ArtifactPushOutput, error) {
	pushStartedAt := time.Now()
	options := e.Options()
	output := internal_type.ArtifactPushOutput{
		Provider:        e.configuration.Provider,
		ConfigurationID: e.configuration.Id,
		Results:         make([]internal_type.ArtifactPushResult, 0, len(input.Artifacts)),
	}

	containerName, _ := options.GetString(azureStorageOptionContainerKey)
	accountName, _ := options.GetString("account_name")
	accountKey, _ := options.GetString("account_key")
	connectionString, _ := options.GetString("connection_string")

	credentialID, _ := options.GetUint64(azureStorageOptionCredentialIDKey)
	if credentialID != 0 {
		credential, err := e.caller.VaultCaller().GetCredential(ctx, e.auth, credentialID)
		if err != nil {
			executeErr := fmt.Errorf("artifact push storage: get credential %d for %s: %w", credentialID, input.ContextID, err)
			_ = e.onPacket(ctx, internal_type.ObservabilityLogRecordPacket{
				ContextID: input.ContextID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: "External artifact push failed",
					Attributes: observability.Attributes{
						"component":        observability.ComponentStorage.String(),
						"operation":        "push_artifact",
						"provider":         e.configuration.Provider,
						"configuration_id": fmt.Sprintf("%d", e.configuration.Id),
						"context_id":       input.ContextID,
						"artifact_count":   fmt.Sprintf("%d", len(input.Artifacts)),
						"pushed_count":     fmt.Sprintf("%d", len(output.Results)),
						"duration_ms":      fmt.Sprintf("%d", time.Since(pushStartedAt).Milliseconds()),
						"error":            executeErr.Error(),
						"error_type":       fmt.Sprintf("%T", executeErr),
					},
				},
			})
			return output, executeErr
		}
		credentialValues := credential.GetValue().AsMap()
		if value, ok := credentialValues[azureStorageOptionContainerKey]; containerName == "" && ok {
			containerName = fmt.Sprintf("%v", value)
		}
		if value, ok := credentialValues["account_name"]; accountName == "" && ok {
			accountName = fmt.Sprintf("%v", value)
		}
		if value, ok := credentialValues["account_key"]; accountKey == "" && ok {
			accountKey = fmt.Sprintf("%v", value)
		}
		if value, ok := credentialValues["connection_string"]; connectionString == "" && ok {
			connectionString = fmt.Sprintf("%v", value)
		}
	}
	destinationAssetStoreConfig := configs.AssetStoreConfig{
		StorageType:       string(configs.AZURE),
		StoragePathPrefix: containerName,
		AzureAuth: &configs.AzureConfig{
			AccountName:      accountName,
			AccountKey:       accountKey,
			ConnectionString: connectionString,
		},
	}
	if !validator.NotBlank(destinationAssetStoreConfig.StoragePathPrefix) {
		executeErr := fmt.Errorf("artifact push storage: container is required for %s", e.configuration.Provider)
		_ = e.onPacket(ctx, internal_type.ObservabilityLogRecordPacket{
			ContextID: input.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: "External artifact push failed",
				Attributes: observability.Attributes{
					"component":        observability.ComponentStorage.String(),
					"operation":        "push_artifact",
					"provider":         e.configuration.Provider,
					"configuration_id": fmt.Sprintf("%d", e.configuration.Id),
					"context_id":       input.ContextID,
					"artifact_count":   fmt.Sprintf("%d", len(input.Artifacts)),
					"pushed_count":     fmt.Sprintf("%d", len(output.Results)),
					"duration_ms":      fmt.Sprintf("%d", time.Since(pushStartedAt).Milliseconds()),
					"error":            executeErr.Error(),
					"error_type":       fmt.Sprintf("%T", executeErr),
				},
			},
		})
		return output, executeErr
	}
	if !validator.NotBlank(destinationAssetStoreConfig.AzureAuth.ConnectionString) &&
		(!validator.NotBlank(destinationAssetStoreConfig.AzureAuth.AccountName) || !validator.NotBlank(destinationAssetStoreConfig.AzureAuth.AccountKey)) {
		executeErr := fmt.Errorf("artifact push storage: connection_string or account_name/account_key is required for %s", e.configuration.Provider)
		_ = e.onPacket(ctx, internal_type.ObservabilityLogRecordPacket{
			ContextID: input.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: "External artifact push failed",
				Attributes: observability.Attributes{
					"component":        observability.ComponentStorage.String(),
					"operation":        "push_artifact",
					"provider":         e.configuration.Provider,
					"configuration_id": fmt.Sprintf("%d", e.configuration.Id),
					"context_id":       input.ContextID,
					"artifact_count":   fmt.Sprintf("%d", len(input.Artifacts)),
					"pushed_count":     fmt.Sprintf("%d", len(output.Results)),
					"duration_ms":      fmt.Sprintf("%d", time.Since(pushStartedAt).Milliseconds()),
					"error":            executeErr.Error(),
					"error_type":       fmt.Sprintf("%T", executeErr),
				},
			},
		})
		return output, executeErr
	}

	pushTimeout := azureStorageDefaultArtifactPushTimeout
	if configuredTimeoutSeconds, _ := options.GetUint32(azureStorageOptionTimeoutSecondsKey); configuredTimeoutSeconds > 0 {
		pushTimeout = time.Duration(configuredTimeoutSeconds) * time.Second
	}
	pushContext, cancelPushContext := context.WithTimeout(ctx, pushTimeout)
	defer cancelPushContext()

	destinationStorage := storage_files.NewStorage(destinationAssetStoreConfig, e.logger)
	configuredPrefix, _ := options.GetString(azureStorageOptionPrefixKey)

	for _, artifact := range input.Artifacts {
		artifactFileName := artifact.Name
		if filepath.Ext(artifactFileName) == "" {
			switch artifact.ContentType {
			case "audio/wav":
				artifactFileName += ".wav"
			case "application/json":
				artifactFileName += ".json"
			case "text/plain":
				artifactFileName += ".txt"
			}
		}
		destinationObjectKey := strings.Join([]string{input.ContextID, artifact.Type, artifactFileName}, "/")
		if configuredPrefix != "" {
			destinationObjectKey = configuredPrefix + "/" + destinationObjectKey
		}

		storageResult := destinationStorage.Store(pushContext, destinationObjectKey, artifact.Content)
		if storageResult.Error != nil {
			executeErr := fmt.Errorf("artifact push storage: push artifact %q to %q: %w", artifact.Name, destinationObjectKey, storageResult.Error)
			_ = e.onPacket(ctx, internal_type.ObservabilityLogRecordPacket{
				ContextID: input.ContextID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: "External artifact push failed",
					Attributes: observability.Attributes{
						"component":        observability.ComponentStorage.String(),
						"operation":        "push_artifact",
						"provider":         e.configuration.Provider,
						"configuration_id": fmt.Sprintf("%d", e.configuration.Id),
						"context_id":       input.ContextID,
						"artifact_count":   fmt.Sprintf("%d", len(input.Artifacts)),
						"pushed_count":     fmt.Sprintf("%d", len(output.Results)),
						"duration_ms":      fmt.Sprintf("%d", time.Since(pushStartedAt).Milliseconds()),
						"error":            executeErr.Error(),
						"error_type":       fmt.Sprintf("%T", executeErr),
					},
				},
			})
			return output, executeErr
		}
		output.Results = append(output.Results, internal_type.ArtifactPushResult{
			Name:           artifact.Name,
			Type:           artifact.Type,
			ContentType:    artifact.ContentType,
			DestinationKey: destinationObjectKey,
			CompletePath:   storageResult.CompletePath,
			StorageType:    string(storageResult.StorageType),
		})
	}

	_ = e.onPacket(ctx, internal_type.ObservabilityLogRecordPacket{
		ContextID: input.ContextID,
		Scope:     internal_type.ObservabilityRecordScopeConversation,
		Record: observability.RecordLog{
			Level:   observability.LevelInfo,
			Message: "External artifact push completed",
			Attributes: observability.Attributes{
				"component":        observability.ComponentStorage.String(),
				"operation":        "push_artifact",
				"provider":         e.configuration.Provider,
				"configuration_id": fmt.Sprintf("%d", e.configuration.Id),
				"context_id":       input.ContextID,
				"artifact_count":   fmt.Sprintf("%d", len(input.Artifacts)),
				"pushed_count":     fmt.Sprintf("%d", len(output.Results)),
				"duration_ms":      fmt.Sprintf("%d", time.Since(pushStartedAt).Milliseconds()),
			},
		},
	})
	return output, nil
}
