// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_artifact_storage

import (
	"context"
	"testing"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
)

func testLogger(t *testing.T) commons.Logger {
	t.Helper()
	logger, err := commons.NewApplicationLogger()
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	return logger
}

func testConfig(provider string, options map[string]interface{}) *internal_assistant_entity.AssistantConfiguration {
	cfg := &internal_assistant_entity.AssistantConfiguration{
		Provider:          provider,
		ConfigurationType: internal_assistant_entity.AssistantConfigurationTypeStorage,
		Enabled:           true,
	}
	for key, value := range options {
		metadata := gorm_models.NewMetadata(key, value)
		cfg.Options = append(cfg.Options, &internal_assistant_entity.AssistantConfigurationOption{
			Metadata: *metadata,
		})
	}
	return cfg
}

func TestAWSExecutor_ExecuteEmitsObservabilityLog(t *testing.T) {
	configuration := testConfig("aws", nil)
	var packets []internal_type.Packet
	onPacket := func(_ context.Context, pkts ...internal_type.Packet) error {
		packets = append(packets, pkts...)
		return nil
	}
	exec := &awsExecutor{
		configuration: configuration,
		onPacket:      onPacket,
	}

	input := internal_type.ArtifactPushInput{
		ContextID: "ctx-logs",
		Artifacts: []internal_type.ArtifactPushArtifact{
			{Name: "payload", Type: "json", ContentType: "application/json", Content: []byte(`{"ok":true}`)},
		},
	}
	if _, err := exec.Execute(context.Background(), input); err == nil {
		t.Fatalf("execute error = nil, want destination config error")
	}

	if len(packets) != 1 {
		t.Fatalf("observability packets = %d, want 1", len(packets))
	}
	logPacket, ok := packets[0].(internal_type.ObservabilityLogRecordPacket)
	if !ok {
		t.Fatalf("observability packet type = %T, want ObservabilityLogRecordPacket", packets[0])
	}
	if logPacket.Scope != internal_type.ObservabilityRecordScopeConversation {
		t.Fatalf("scope = %q, want conversation", logPacket.Scope)
	}
	if logPacket.Record.Level != observability.LevelError {
		t.Fatalf("level = %q, want error", logPacket.Record.Level)
	}
	if got, want := logPacket.Record.Attributes["component"], observability.ComponentStorage.String(); got != want {
		t.Fatalf("component = %q, want %q", got, want)
	}
	if got, want := logPacket.Record.Attributes["operation"], "push_artifact"; got != want {
		t.Fatalf("operation = %q, want %q", got, want)
	}
	if got, want := logPacket.Record.Attributes["pushed_count"], "0"; got != want {
		t.Fatalf("pushed_count = %q, want %q", got, want)
	}
	if logPacket.Record.Attributes["error"] == "" {
		t.Fatalf("error attribute is empty")
	}
}
