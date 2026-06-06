// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package providers

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/rapidaai/pkg/telemetry"
)

// OTLPExporter converts telemetry records to OTEL spans and ships
// them to any OTLP-compatible backend via the configured endpoint.
type OTLPExporter struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
	once     sync.Once
}

// NewOTLPExporter creates an OTLPExporter connected to the given OTLP endpoint.
func NewOTLPExporter(ctx context.Context, cfg OTLPConfig) (*OTLPExporter, error) {
	spanExporter, err := newOTLPSpanExporter(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("telemetry/otlp: create span exporter: %w", err)
	}

	res, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("rapida-voice-assistant"),
			semconv.ServiceVersion("1.0"),
			semconv.TelemetrySDKLanguageGo,
		),
	)

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(spanExporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
	)

	return &OTLPExporter{
		provider: provider,
		tracer:   provider.Tracer("rapida.voice"),
	}, nil
}

func (e *OTLPExporter) Export(ctx context.Context, scope telemetry.Scope, rec telemetry.Record) error {
	switch typed := rec.(type) {
	case telemetry.LogRecord:
		t := typed.OccurredAt
		if t.IsZero() {
			t = time.Now()
		}
		attrs := []attribute.KeyValue{
			attribute.String("rapida.telemetry.kind", "log"),
			attribute.String("rapida.log.level", typed.Level),
			attribute.String("rapida.log.message", typed.Message),
			attribute.Int64("rapida.project.id", int64(scope.ProjectID)),
			attribute.Int64("rapida.organization.id", int64(scope.OrganizationID)),
			attribute.String("rapida.scope", scope.Name),
		}
		for k, v := range scope.ScopeAttributes {
			attrs = append(attrs, attribute.String("rapida.scope."+k, v))
		}
		for k, v := range typed.Attributes {
			attrs = append(attrs, attribute.String("rapida.attribute."+k, v))
		}
		_, span := e.tracer.Start(ctx, "rapida.voice.log",
			trace.WithTimestamp(t),
			trace.WithAttributes(attrs...),
		)
		span.End(trace.WithTimestamp(t))
	case telemetry.EventRecord:
		t := typed.OccurredAt
		if t.IsZero() {
			t = time.Now()
		}
		attrs := []attribute.KeyValue{
			attribute.String("rapida.telemetry.kind", "event"),
			attribute.String("rapida.event", typed.Event),
			attribute.String("rapida.component", typed.Component),
			attribute.Int64("rapida.project.id", int64(scope.ProjectID)),
			attribute.Int64("rapida.organization.id", int64(scope.OrganizationID)),
			attribute.String("rapida.scope", scope.Name),
		}
		for k, v := range scope.ScopeAttributes {
			attrs = append(attrs, attribute.String("rapida.scope."+k, v))
		}
		for k, v := range typed.Attributes {
			attrs = append(attrs, attribute.String("rapida.attribute."+k, v))
		}
		_, span := e.tracer.Start(ctx, "rapida.voice.event."+typed.Event,
			trace.WithTimestamp(t),
			trace.WithAttributes(attrs...),
		)
		span.End(trace.WithTimestamp(t))
	case telemetry.MetricRecord:
		t := typed.OccurredAt
		if t.IsZero() {
			t = time.Now()
		}
		attrs := []attribute.KeyValue{
			attribute.String("rapida.telemetry.kind", "metric"),
			attribute.String("rapida.metric.name", typed.Name),
			attribute.String("rapida.metric.value", typed.Value),
			attribute.String("rapida.metric.description", typed.Description),
			attribute.Int64("rapida.project.id", int64(scope.ProjectID)),
			attribute.Int64("rapida.organization.id", int64(scope.OrganizationID)),
			attribute.String("rapida.scope", scope.Name),
		}
		for k, v := range scope.ScopeAttributes {
			attrs = append(attrs, attribute.String("rapida.scope."+k, v))
		}
		for k, v := range typed.Attributes {
			attrs = append(attrs, attribute.String("rapida.attribute."+k, v))
		}
		_, span := e.tracer.Start(ctx, "rapida.voice.metric."+typed.Name,
			trace.WithTimestamp(t),
			trace.WithAttributes(attrs...),
		)
		span.End(trace.WithTimestamp(t))
	}
	return nil
}

// Close flushes the batch processor and releases OTLP resources.
func (e *OTLPExporter) Close(ctx context.Context) error {
	var err error
	e.once.Do(func() {
		if ferr := e.provider.ForceFlush(ctx); ferr != nil {
			err = ferr
		}
		if serr := e.provider.Shutdown(ctx); serr != nil && err == nil {
			err = serr
		}
	})
	return err
}

func newOTLPSpanExporter(ctx context.Context, cfg OTLPConfig) (sdktrace.SpanExporter, error) {
	headers := parseOTLPHeaders(cfg.Headers)
	switch strings.ToLower(cfg.Protocol) {
	case "grpc":
		opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(cfg.Endpoint)}
		if len(headers) > 0 {
			opts = append(opts, otlptracegrpc.WithHeaders(headers))
		}
		if cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
			opts = append(opts, otlptracegrpc.WithInsecure()) //nolint:staticcheck
		}
		return otlptracegrpc.New(ctx, opts...)
	default:
		opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(cfg.Endpoint)}
		if len(headers) > 0 {
			opts = append(opts, otlptracehttp.WithHeaders(headers))
		}
		if cfg.Insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		return otlptracehttp.New(ctx, opts...)
	}
}

func parseOTLPHeaders(pairs []string) map[string]string {
	headers := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		if k, v, ok := strings.Cut(pair, "="); ok {
			headers[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return headers
}
