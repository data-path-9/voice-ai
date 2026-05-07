// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_analysis_endpoint

import (
	"context"
	"encoding/json"
	"fmt"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	endpoint_client_builders "github.com/rapidaai/pkg/clients/endpoint/builders"
	"github.com/rapidaai/pkg/commons"
	rapida_types "github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type runtimeExecutor struct {
	logger       commons.Logger
	callback     internal_type.Callback
	caller       internal_type.InternalCaller
	inputBuilder endpoint_client_builders.InputInvokeBuilder
}

// NewExecutor creates a fully wired endpoint-based analysis executor.
func NewExecutor(
	logger commons.Logger,
	_ context.Context,
	callback internal_type.Callback,
	caller internal_type.InternalCaller,
) (internal_type.AnalysisExecutor, error) {
	return &runtimeExecutor{
		logger:       logger,
		callback:     callback,
		caller:       caller,
		inputBuilder: endpoint_client_builders.NewInputInvokeBuilder(logger),
	}, nil
}

// Execute runs one analysis and pushes metadata via callback packet.
func (e *runtimeExecutor) Execute(ctx context.Context, packet internal_type.ExecuteAnalysisPacket) error {
	response, err := e.caller.DeploymentCaller().Invoke(
		ctx,
		packet.Auth,
		e.inputBuilder.Invoke(
			&protos.EndpointDefinition{
				EndpointId: packet.Analysis.GetEndpointId(),
				Version:    packet.Analysis.GetEndpointVersion(),
			},
			e.inputBuilder.Arguments(packet.Arguments, nil),
			nil,
			nil,
		),
	)
	if err != nil {
		return err
	}
	if !response.GetSuccess() || len(response.GetData()) == 0 {
		return fmt.Errorf("empty response from endpoint")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(response.GetData()[0]), &parsed); err != nil {
		parsed = map[string]interface{}{"result": response.GetData()[0]}
	}

	metadata := map[string]interface{}{
		fmt.Sprintf("analysis.%s", packet.Analysis.GetName()): parsed,
	}
	metadataList := rapida_types.NewMetadataList(metadata)
	protoMetadata := make([]*protos.Metadata, 0, len(metadataList))
	for _, item := range metadataList {
		protoMetadata = append(protoMetadata, &protos.Metadata{Key: item.Key, Value: item.Value})
	}

	e.callback.OnPacket(ctx, internal_type.ConversationMetadataPacket{
		ContextID: packet.ConversationID,
		Metadata:  protoMetadata,
	})
	return nil
}

// Close releases executor dependencies.
func (e *runtimeExecutor) Close(_ context.Context) error {
	e.callback = nil
	e.caller = nil
	return nil
}
