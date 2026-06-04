// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package timeline

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/validator"
)

const defaultIndexPrefix = "rapida-timeline"

type Config struct {
	Logger      commons.Logger
	OpenSearch  connectors.OpenSearchConnector
	IndexPrefix string
}

type Collector struct {
	logger      commons.Logger
	opensearch  connectors.OpenSearchConnector
	indexPrefix string
}

func New(cfg Config) observability.Collector {
	if !validator.NonNil(cfg.OpenSearch) {
		return observability.NoopCollector{}
	}
	indexPrefix := strings.TrimSpace(cfg.IndexPrefix)
	if !validator.NotBlank(indexPrefix) {
		indexPrefix = defaultIndexPrefix
	}
	return &Collector{
		logger:      cfg.Logger,
		opensearch:  cfg.OpenSearch,
		indexPrefix: indexPrefix,
	}
}

func (c *Collector) Collect(ctx context.Context, envelope observability.Envelope) error {
	if !validator.NonNil(c) || !validator.NonNil(c.opensearch) {
		return nil
	}
	doc := newDocument(envelope)
	if !validator.NotBlank(doc.ID) {
		doc.ID = uuid.NewString()
	}
	return c.bulk(ctx, c.index(doc.OccurredAt), doc)
}

func (c *Collector) Shutdown(context.Context) error {
	return nil
}

func (c *Collector) index(at time.Time) string {
	if at.IsZero() {
		at = time.Now().UTC()
	}
	return c.indexPrefix + "-" + at.UTC().Format("20060102")
}

func (c *Collector) bulk(ctx context.Context, index string, doc document) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`{ "index": { "_index": "%s", "_id": "%s" } }`, index, doc.ID))
	sb.WriteByte('\n')
	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	sb.Write(body)
	sb.WriteByte('\n')
	if err := c.opensearch.Bulk(ctx, sb.String()); err != nil {
		if validator.NonNil(c.logger) {
			c.logger.Errorf("observability/timeline/opensearch: bulk index error: %v", err)
		}
		return err
	}
	return nil
}

type document struct {
	ID                      string                 `json:"id"`
	Kind                    string                 `json:"kind"`
	Name                    string                 `json:"name"`
	Category                string                 `json:"category"`
	Level                   string                 `json:"level"`
	Outcome                 string                 `json:"outcome"`
	Title                   string                 `json:"title"`
	ProjectID               uint64                 `json:"projectId"`
	OrganizationID          uint64                 `json:"organizationId"`
	AssistantID             uint64                 `json:"assistantId"`
	AssistantConversationID uint64                 `json:"assistantConversationId"`
	ContextID               string                 `json:"contextId"`
	Attributes              map[string]string      `json:"attributes,omitempty"`
	Data                    map[string]interface{} `json:"data,omitempty"`
	OccurredAt              time.Time              `json:"occurredAt"`
	ReceivedAt              time.Time              `json:"receivedAt"`
	DurationMs              int64                  `json:"durationMs,omitempty"`
}

func newDocument(envelope observability.Envelope) document {
	occurredAt := envelope.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = envelope.ReceivedAt
	}
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	return document{
		ID:                      envelope.ID,
		Kind:                    string(envelope.Kind),
		Name:                    envelope.Name.String(),
		Category:                envelope.Category.String(),
		Level:                   string(envelope.Level),
		Outcome:                 string(envelope.Outcome),
		Title:                   envelope.Title,
		ProjectID:               envelope.Scope.ProjectID,
		OrganizationID:          envelope.Scope.OrganizationID,
		AssistantID:             envelope.Scope.AssistantID,
		AssistantConversationID: envelope.Scope.ConversationID,
		ContextID:               envelope.Scope.ContextID,
		Attributes:              envelope.Attributes.Clone(),
		Data:                    envelope.Data.Clone(),
		OccurredAt:              occurredAt,
		ReceivedAt:              envelope.ReceivedAt,
		DurationMs:              envelope.Duration.Milliseconds(),
	}
}
