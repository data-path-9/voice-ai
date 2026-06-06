// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package providers

import (
	"context"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/telemetry"
)

// LoggingExporter logs telemetry records at INFO level.
type LoggingExporter struct {
	logger commons.Logger
}

func NewLoggingExporter(logger commons.Logger, _ LoggingConfig) *LoggingExporter {
	return &LoggingExporter{logger: logger}
}

func (e *LoggingExporter) Export(_ context.Context, scope telemetry.Scope, rec telemetry.Record) error {
	switch typed := rec.(type) {
	case telemetry.LogRecord:
		e.logger.Infof("[telemetry/log] scope=%s scopeAttributes=%v message=%s level=%s attributes=%v",
			scope.Name, scope.ScopeAttributes, typed.Message, typed.Level, typed.Attributes)
	case telemetry.EventRecord:
		e.logger.Infof("[telemetry/event] scope=%s scopeAttributes=%v event=%s component=%s attributes=%v",
			scope.Name, scope.ScopeAttributes, typed.Event, typed.Component, typed.Attributes)
	case telemetry.MetricRecord:
		e.logger.Infof("[telemetry/metric] scope=%s scopeAttributes=%v name=%s value=%s",
			scope.Name, scope.ScopeAttributes, typed.Name, typed.Value)
	}
	return nil
}

func (e *LoggingExporter) Close(_ context.Context) error { return nil }
