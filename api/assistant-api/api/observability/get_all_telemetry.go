// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package observability_api

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rapidaai/pkg/exceptions"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

func (api *observabilityGrpcApi) GetAllTelemetry(
	ctx context.Context,
	request *protos.GetAllTelemetryRequest,
) (*protos.GetAllTelemetryResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		api.logger.Errorf("unauthenticated request for GetAllTelemetry")
		return exceptions.AuthenticationError[protos.GetAllTelemetryResponse]()
	}

	if api.opensearch == nil {
		return &protos.GetAllTelemetryResponse{Code: 200, Success: true}, nil
	}

	page := int(request.GetPaginate().GetPage())
	if page < 1 {
		page = 1
	}
	size := int(request.GetPaginate().GetPageSize())
	if size < 1 || size > 100 {
		size = 20
	}
	from := (page - 1) * size

	indices := []string{"rapida-logs-*", "rapida-events-*", "rapida-metrics-*"}
	filter := []interface{}{
		map[string]interface{}{"term": map[string]interface{}{"organizationId": *iAuth.GetCurrentOrganizationId()}},
		map[string]interface{}{"term": map[string]interface{}{"projectId": *iAuth.GetCurrentProjectId()}},
	}
	must := []interface{}{}
	timeRange := map[string]interface{}{}

	for _, criteria := range request.GetCriterias() {
		key := strings.TrimSpace(criteria.GetKey())
		value := strings.TrimSpace(criteria.GetValue())
		if key == "" || value == "" {
			continue
		}

		switch key {
		case "kind":
			switch strings.ToLower(value) {
			case "log":
				indices = []string{"rapida-logs-*"}
				filter = append(filter, map[string]interface{}{"term": map[string]interface{}{"kind": "log"}})
			case "event":
				indices = []string{"rapida-events-*"}
				filter = append(filter, map[string]interface{}{"term": map[string]interface{}{"kind": "event"}})
			case "metric":
				indices = []string{"rapida-metrics-*"}
				filter = append(filter, map[string]interface{}{"term": map[string]interface{}{"kind": "metric"}})
			}
		case "id", "scope", "event", "name", "level":
			filter = append(filter, map[string]interface{}{"term": map[string]interface{}{key: value}})
		case "assistantId":
			filter = append(filter, map[string]interface{}{"term": map[string]interface{}{"scopeAttributes.assistantId": value}})
		case "assistantConversationId", "conversationId":
			filter = append(filter, map[string]interface{}{"term": map[string]interface{}{"scopeAttributes.assistantConversationId": value}})
		case "messageId":
			filter = append(filter, map[string]interface{}{"term": map[string]interface{}{"scopeAttributes.messageId": value}})
		case "messageRole":
			filter = append(filter, map[string]interface{}{"term": map[string]interface{}{"scopeAttributes.messageRole": value}})
		case "traceId":
			filter = append(filter, map[string]interface{}{"term": map[string]interface{}{"context.traceId": value}})
		case "occurredAtFrom", "from", "start":
			timeRange["gte"] = value
		case "occurredAtTo", "to", "end":
			timeRange["lte"] = value
		case "search", "q":
			must = append(must, map[string]interface{}{
				"multi_match": map[string]interface{}{
					"query":  value,
					"fields": []string{"message", "event", "name", "description", "level"},
				},
			})
		default:
			if strings.HasPrefix(key, "attributes.") || strings.HasPrefix(key, "scopeAttributes.") || strings.HasPrefix(key, "context.") {
				filter = append(filter, map[string]interface{}{"term": map[string]interface{}{key: value}})
			}
		}
	}
	if len(timeRange) > 0 {
		filter = append(filter, map[string]interface{}{"range": map[string]interface{}{"occurredAt": timeRange}})
	}

	orderColumn := "occurredAt"
	orderDirection := "desc"
	if order := request.GetOrder(); order != nil {
		switch order.GetColumn() {
		case "occurredAt", "kind", "scope", "event", "name", "level":
			orderColumn = order.GetColumn()
		}
		if strings.EqualFold(order.GetOrder(), "asc") {
			orderDirection = "asc"
		}
	}

	boolQuery := map[string]interface{}{"filter": filter}
	if len(must) > 0 {
		boolQuery["must"] = must
	}
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": boolQuery,
		},
		"sort": []interface{}{
			map[string]interface{}{orderColumn: map[string]interface{}{"order": orderDirection}},
		},
		"from": from,
		"size": size,
	}

	body, _ := json.Marshal(query)
	hits := api.opensearch.Search(ctx, indices, string(body))
	if hits.Error() != nil {
		api.logger.Errorf("unable to query telemetry: %v", hits.Error())
		return exceptions.BadRequestError[protos.GetAllTelemetryResponse]("Unable to get telemetry.")
	}

	records := make([]*protos.ObservabilityRecord, 0, len(hits.Hits.Hits))
	for _, hit := range hits.Hits.Hits {
		src, _ := hit["_source"].(map[string]interface{})
		if src == nil {
			continue
		}

		strVal := func(v interface{}) string {
			if v == nil {
				return ""
			}
			if f, ok := v.(float64); ok {
				return strconv.FormatUint(uint64(f), 10)
			}
			return fmt.Sprintf("%v", v)
		}
		uintVal := func(v interface{}) uint64 {
			if f, ok := v.(float64); ok {
				return uint64(f)
			}
			if s, ok := v.(string); ok {
				out, _ := strconv.ParseUint(s, 10, 64)
				return out
			}
			return 0
		}
		mapVal := func(v interface{}) map[string]string {
			out := map[string]string{}
			if raw, ok := v.(map[string]interface{}); ok {
				for key, value := range raw {
					out[key] = strVal(value)
				}
			}
			return out
		}
		timestampVal := func(v interface{}) *timestamppb.Timestamp {
			if value, ok := v.(string); ok {
				if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
					return timestamppb.New(t)
				}
				if t, err := time.Parse(time.RFC3339, value); err == nil {
					return timestamppb.New(t)
				}
			}
			return nil
		}

		switch strVal(src["kind"]) {
		case "log":
			records = append(records, &protos.ObservabilityRecord{
				Record: &protos.ObservabilityRecord_Log{
					Log: &protos.ObservabilityLogRecord{
						Id:              strVal(src["id"]),
						Kind:            protos.ObservabilityRecordKind_OBSERVABILITY_RECORD_KIND_LOG,
						Level:           strVal(src["level"]),
						Message:         strVal(src["message"]),
						ProjectId:       uintVal(src["projectId"]),
						OrganizationId:  uintVal(src["organizationId"]),
						Scope:           strVal(src["scope"]),
						ScopeAttributes: mapVal(src["scopeAttributes"]),
						Attributes:      mapVal(src["attributes"]),
						Context:         mapVal(src["context"]),
						OccurredAt:      timestampVal(src["occurredAt"]),
					},
				},
			})
		case "event":
			records = append(records, &protos.ObservabilityRecord{
				Record: &protos.ObservabilityRecord_Event{
					Event: &protos.ObservabilityEventRecord{
						Id:              strVal(src["id"]),
						Kind:            protos.ObservabilityRecordKind_OBSERVABILITY_RECORD_KIND_EVENT,
						Event:           strVal(src["event"]),
						Component:       strVal(src["component"]),
						ProjectId:       uintVal(src["projectId"]),
						OrganizationId:  uintVal(src["organizationId"]),
						Scope:           strVal(src["scope"]),
						ScopeAttributes: mapVal(src["scopeAttributes"]),
						Attributes:      mapVal(src["attributes"]),
						Context:         mapVal(src["context"]),
						OccurredAt:      timestampVal(src["occurredAt"]),
					},
				},
			})
		case "metric":
			records = append(records, &protos.ObservabilityRecord{
				Record: &protos.ObservabilityRecord_Metric{
					Metric: &protos.ObservabilityMetricRecord{
						Id:              strVal(src["id"]),
						Kind:            protos.ObservabilityRecordKind_OBSERVABILITY_RECORD_KIND_METRIC,
						Name:            strVal(src["name"]),
						Value:           strVal(src["value"]),
						Description:     strVal(src["description"]),
						ProjectId:       uintVal(src["projectId"]),
						OrganizationId:  uintVal(src["organizationId"]),
						Scope:           strVal(src["scope"]),
						ScopeAttributes: mapVal(src["scopeAttributes"]),
						Attributes:      mapVal(src["attributes"]),
						Context:         mapVal(src["context"]),
						OccurredAt:      timestampVal(src["occurredAt"]),
					},
				},
			})
		}
	}

	return &protos.GetAllTelemetryResponse{
		Code:    200,
		Success: true,
		Data:    records,
		Paginated: &protos.Paginated{
			TotalItem:   uint32(hits.Hits.Total.Value),
			CurrentPage: uint32(page),
		},
	}, nil
}
