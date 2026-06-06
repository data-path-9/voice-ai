// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package telemetry

import "time"

type Record interface {
	isTelemetryRecord()
}

type CommonRecord struct {
	ID              string
	ProjectID       uint64
	OrganizationID  uint64
	Scope           string
	ScopeAttributes map[string]string
	Attributes      map[string]string
	OccurredAt      time.Time
}

type LogRecord struct {
	CommonRecord
	Level   string
	Message string
}

func (LogRecord) isTelemetryRecord() {}

type EventRecord struct {
	CommonRecord
	Event     string
	Component string
}

func (EventRecord) isTelemetryRecord() {}

type MetricRecord struct {
	CommonRecord
	Name        string
	Value       string
	Description string
}

func (MetricRecord) isTelemetryRecord() {}
