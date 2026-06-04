// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package collectors

import (
	"context"
	"strings"
	"testing"

	assistant_config "github.com/rapidaai/api/assistant-api/config"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_telemetry_entity "github.com/rapidaai/api/assistant-api/internal/entity/telemetry"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/configs"
	"github.com/rapidaai/pkg/connectors"
)

type openSearchStub struct {
	bodies []string
}

func (o *openSearchStub) Connect(context.Context) error {
	return nil
}

func (o *openSearchStub) Name() string {
	return "opensearch-stub"
}

func (o *openSearchStub) IsConnected(context.Context) bool {
	return true
}

func (o *openSearchStub) Disconnect(context.Context) error {
	return nil
}

func (o *openSearchStub) VectorSearch(context.Context, string, []float64, map[string]interface{}, *connectors.VectorSearchOptions) ([]map[string]interface{}, error) {
	return nil, nil
}

func (o *openSearchStub) HybridSearch(context.Context, string, string, []float64, map[string]interface{}, *connectors.VectorSearchOptions) ([]map[string]interface{}, error) {
	return nil, nil
}

func (o *openSearchStub) TextSearch(context.Context, string, string, map[string]interface{}, *connectors.VectorSearchOptions) ([]map[string]interface{}, error) {
	return nil, nil
}

func (o *openSearchStub) Search(context.Context, []string, string) *connectors.SearchResponse {
	return nil
}

func (o *openSearchStub) SearchWithCount(context.Context, []string, string) *connectors.SearchResponseWithCount {
	return nil
}

func (o *openSearchStub) Persist(context.Context, string, string, string) error {
	return nil
}

func (o *openSearchStub) Update(context.Context, string, string, string) error {
	return nil
}

func (o *openSearchStub) Bulk(_ context.Context, body string) error {
	o.bodies = append(o.bodies, body)
	return nil
}

func TestDefaultTelemetryEnabled(t *testing.T) {
	if DefaultTelemetryEnabled(nil) {
		t.Fatal("nil config should not enable default telemetry")
	}
	if DefaultTelemetryEnabled(&assistant_config.AssistantConfig{}) {
		t.Fatal("missing telemetry config should not enable default telemetry")
	}
	if DefaultTelemetryEnabled(&assistant_config.AssistantConfig{
		TelemetryConfig: &configs.TelemetryConfig{TelemetryType: "unknown"},
	}) {
		t.Fatal("unknown telemetry type should not enable default telemetry")
	}
	if !DefaultTelemetryEnabled(&assistant_config.AssistantConfig{
		TelemetryConfig: &configs.TelemetryConfig{
			TelemetryType: string(configs.LOGGING),
			Logging:       &configs.TelemetryLoggingConfig{},
		},
	}) {
		t.Fatal("valid telemetry type should enable default telemetry")
	}
}

func TestAppendDefault_AppendsTelemetryWhenDefaultTelemetryEnabled(t *testing.T) {
	collectors, err := AppendDefault(context.Background(), nil, DefaultConfig{
		Logger: testLogger(t),
		AssistantConfig: &assistant_config.AssistantConfig{
			TelemetryConfig: &configs.TelemetryConfig{
				TelemetryType: string(configs.LOGGING),
				Logging:       &configs.TelemetryLoggingConfig{},
			},
		},
	})
	if err != nil {
		t.Fatalf("AppendDefault returned error: %v", err)
	}
	if len(collectors) != 1 {
		t.Fatalf("expected telemetry collector only, got %d", len(collectors))
	}
}

func TestAppendDefault_AppendsTimelineForOpenSearchTelemetry(t *testing.T) {
	opensearch := &openSearchStub{}
	collectors, err := AppendDefault(context.Background(), nil, DefaultConfig{
		Logger:     testLogger(t),
		OpenSearch: opensearch,
		AssistantConfig: &assistant_config.AssistantConfig{
			TelemetryConfig: &configs.TelemetryConfig{
				TelemetryType: string(configs.OPENSEARCH),
				OpenSearch:    &configs.TelemetryOpenSearchConfig{IndexPrefix: "voice-ai"},
			},
		},
	})
	if err != nil {
		t.Fatalf("AppendDefault returned error: %v", err)
	}
	if len(collectors) != 2 {
		t.Fatalf("expected telemetry and timeline collectors, got %d", len(collectors))
	}

	err = collectors[1].Collect(context.Background(), observability.Envelope{
		ID:   "evt-1",
		Kind: observability.RecordKindEvent,
		Name: observability.CallRinging,
	})
	if err != nil {
		t.Fatalf("timeline Collect returned error: %v", err)
	}
	if len(opensearch.bodies) != 1 || !strings.Contains(opensearch.bodies[0], `"voice-ai-`) {
		t.Fatalf("timeline did not use telemetry opensearch index prefix: %v", opensearch.bodies)
	}
}

func TestAppendDefault_AppendsTimelineForAssistantOpenSearchProvider(t *testing.T) {
	collectors, err := AppendDefault(context.Background(), nil, DefaultConfig{
		Logger:     testLogger(t),
		OpenSearch: &openSearchStub{},
		AssistantTelemetryProvider: []*internal_telemetry_entity.AssistantTelemetryProvider{
			{ProviderType: string(configs.OPENSEARCH), Enabled: true},
			{ProviderType: string(configs.LOGGING), Enabled: false},
		},
	})
	if err != nil {
		t.Fatalf("AppendDefault returned error: %v", err)
	}
	if len(collectors) != 2 {
		t.Fatalf("expected telemetry and timeline collectors, got %d", len(collectors))
	}
}

func TestAppendDefault_SkipsInactiveConfig(t *testing.T) {
	collectors, err := AppendDefault(context.Background(), nil, DefaultConfig{
		Logger: testLogger(t),
		AssistantConfig: &assistant_config.AssistantConfig{
			TelemetryConfig: &configs.TelemetryConfig{},
		},
	})
	if err != nil {
		t.Fatalf("AppendDefault returned error: %v", err)
	}
	if len(collectors) != 0 {
		t.Fatalf("expected no collectors, got %d", len(collectors))
	}
}

func TestAppendDefault_SkipsUnknownDefaultTelemetryType(t *testing.T) {
	collectors, err := AppendDefault(context.Background(), nil, DefaultConfig{
		Logger: testLogger(t),
		AssistantConfig: &assistant_config.AssistantConfig{
			TelemetryConfig: &configs.TelemetryConfig{TelemetryType: "unknown"},
		},
	})
	if err != nil {
		t.Fatalf("AppendDefault returned error: %v", err)
	}
	if len(collectors) != 0 {
		t.Fatalf("expected no collectors, got %d", len(collectors))
	}
}

func TestAppendDefault_ReturnsValidationErrorForMissingLogger(t *testing.T) {
	_, err := AppendDefault(context.Background(), nil, DefaultConfig{
		AssistantConfig: &assistant_config.AssistantConfig{
			TelemetryConfig: &configs.TelemetryConfig{
				TelemetryType: string(configs.LOGGING),
				Logging:       &configs.TelemetryLoggingConfig{},
			},
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestAppendDefault_ReturnsValidationErrorForEmptyAssistantProvider(t *testing.T) {
	_, err := AppendDefault(context.Background(), nil, DefaultConfig{
		Logger: testLogger(t),
		AssistantTelemetryProvider: []*internal_telemetry_entity.AssistantTelemetryProvider{
			{ProviderType: "", Enabled: true},
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestAppendDefault_ReturnsFactoryErrorForUnknownAssistantProvider(t *testing.T) {
	_, err := AppendDefault(context.Background(), nil, DefaultConfig{
		Logger: testLogger(t),
		AssistantTelemetryProvider: []*internal_telemetry_entity.AssistantTelemetryProvider{
			{ProviderType: "unknown", Enabled: true},
		},
	})
	if err == nil {
		t.Fatal("expected factory error")
	}
}

func TestAppendDefault_AppendsWebhookCollector(t *testing.T) {
	collectors, err := AppendDefault(context.Background(), nil, DefaultConfig{
		Logger: testLogger(t),
		AssistantWebhooks: []*internal_assistant_entity.AssistantWebhook{
			{Provider: internal_assistant_entity.AssistantWebhookProviderHTTP},
		},
	})
	if err != nil {
		t.Fatalf("AppendDefault returned error: %v", err)
	}
	if len(collectors) != 1 {
		t.Fatalf("expected webhook collector only, got %d", len(collectors))
	}
}

func testLogger(t *testing.T) commons.Logger {
	t.Helper()

	logger, err := commons.NewApplicationLogger(
		commons.Name("observability-collectors-test"),
		commons.Level("error"),
		commons.EnableFile(false),
	)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	return logger
}

var _ connectors.OpenSearchConnector = (*openSearchStub)(nil)
