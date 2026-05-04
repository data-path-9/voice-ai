// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package analysis

import (
	"context"
	"encoding/json"
	"fmt"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	endpoint_client "github.com/rapidaai/pkg/clients/endpoint"
	endpoint_client_builders "github.com/rapidaai/pkg/clients/endpoint/builders"
	"github.com/rapidaai/pkg/commons"
	rapida_types "github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

// Executor defines analysis runtime behavior.
type Executor interface {
	Init(ctx context.Context, communication internal_type.Communication)
	Execute(ctx context.Context, packet internal_type.RunAnalysisPacket) error
	Close(ctx context.Context)
}

type runtimeExecutor struct {
	logger       commons.Logger
	deployment   endpoint_client.DeploymentServiceClient
	inputBuilder endpoint_client_builders.InputInvokeBuilder
	onPacket     func(ctx context.Context, pkts ...internal_type.Packet) error
}

// NewExecutor creates an analysis executor.
func NewExecutor(logger commons.Logger) Executor {
	return &runtimeExecutor{
		logger:       logger,
		inputBuilder: endpoint_client_builders.NewInputInvokeBuilder(logger),
	}
}

// Init wires live communication dependencies required by executor.
func (e *runtimeExecutor) Init(_ context.Context, communication internal_type.Communication) {
	e.deployment = communication.DeploymentCaller()
	e.onPacket = communication.OnPacket
}

// Execute runs one analysis and pushes metadata via callback packet.
func (e *runtimeExecutor) Execute(ctx context.Context, packet internal_type.RunAnalysisPacket) error {
	response, err := e.deployment.Invoke(
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

	return e.onPacket(ctx, internal_type.ConversationMetadataPacket{
		ContextID: packet.ConversationID,
		Metadata:  protoMetadata,
	})
}

// Close releases executor dependencies.
func (e *runtimeExecutor) Close(_ context.Context) {
	e.deployment = nil
	e.onPacket = nil
}
