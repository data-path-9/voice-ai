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

type Scope struct {
	ProjectID       uint64
	OrganizationID  uint64
	Name            string
	ScopeAttributes map[string]string
}

type LogRecord struct {
	ID         string
	Level      string
	Message    string
	Attributes map[string]string
	OccurredAt time.Time
}

func (LogRecord) isTelemetryRecord() {}

type EventRecord struct {
	ID         string
	Event      string
	Component  string
	Attributes map[string]string
	OccurredAt time.Time
}

func (EventRecord) isTelemetryRecord() {}

type MetricRecord struct {
	ID          string
	Name        string
	Value       string
	Description string
	Attributes  map[string]string
	OccurredAt  time.Time
}

func (MetricRecord) isTelemetryRecord() {}
