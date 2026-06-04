// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package observability

import (
	"errors"
	"fmt"
	"time"

	"github.com/rapidaai/pkg/validator"
)

type RecordKind string

const (
	RecordKindEvent    RecordKind = "event"
	RecordKindMetric   RecordKind = "metric"
	RecordKindMetadata RecordKind = "metadata"
)

type Level string

const (
	LevelDebug   Level = "debug"
	LevelInfo    Level = "info"
	LevelWarning Level = "warning"
	LevelError   Level = "error"
	LevelFatal   Level = "fatal"
)

type Outcome string

const (
	OutcomeUnknown   Outcome = ""
	OutcomeSuccess   Outcome = "success"
	OutcomeFailure   Outcome = "failure"
	OutcomeCancelled Outcome = "cancelled"
	OutcomeSkipped   Outcome = "skipped"
)

// AttributeKey names are intentionally aligned with the current observe data
// keys where those names are already clear.
type AttributeKey string

const (
	AttrComponent      AttributeKey = "component"
	AttrType           AttributeKey = "type"
	AttrProvider       AttributeKey = "provider"
	AttrDirection      AttributeKey = "direction"
	AttrReason         AttributeKey = "reason"
	AttrError          AttributeKey = "error"
	AttrStage          AttributeKey = "stage"
	AttrDID            AttributeKey = "did"
	AttrCaller         AttributeKey = "caller"
	AttrCallee         AttributeKey = "callee"
	AttrContextID      AttributeKey = "context_id"
	AttrProviderCallID AttributeKey = "provider_call_id"
	AttrCodec          AttributeKey = "codec"
	AttrSampleRate     AttributeKey = "sample_rate"
	AttrMode           AttributeKey = "mode"
	AttrFrom           AttributeKey = "from"
	AttrTo             AttributeKey = "to"
	AttrDuration       AttributeKey = "duration_ms"
	AttrMessages       AttributeKey = "messages"
	AttrDigit          AttributeKey = "digit"
	AttrTarget         AttributeKey = "target"
	AttrOutboundCallID AttributeKey = "outbound_call_id"
	AttrStatus         AttributeKey = "status"
	AttrOldState       AttributeKey = "old_state"
	AttrNewState       AttributeKey = "new_state"
	AttrSource         AttributeKey = "source"
	AttrSpeakerID      AttributeKey = "speaker_id"
	AttrConfidence     AttributeKey = "confidence"
	AttrItemID         AttributeKey = "item_id"
	AttrRequestID      AttributeKey = "request_id"
	AttrHTTPURL        AttributeKey = "http_url"
	AttrHTTPMethod     AttributeKey = "http_method"
	AttrHTTPStatus     AttributeKey = "http_status"
	AttrRetryCount     AttributeKey = "retry_count"
	AttrInputText      AttributeKey = "input_text"
	AttrOutputText     AttributeKey = "output_text"
	AttrTranscript     AttributeKey = "transcript"
	AttrLanguage       AttributeKey = "language"
	AttrModel          AttributeKey = "model"
	AttrToolName       AttributeKey = "tool_name"
	AttrUsageCategory  AttributeKey = "usage_category"
)

type Attributes map[string]string

func (a Attributes) Clone() Attributes {
	copied := make(Attributes, len(a))
	for k, v := range a {
		copied[k] = v
	}
	return copied
}

func (a Attributes) Add(key AttributeKey, value string) Attributes {
	if a == nil {
		a = Attributes{}
	}
	if validator.NotBlank(value) {
		a[string(key)] = value
	}
	return a
}

type Data map[string]interface{}

func (d Data) Clone() Data {
	copied := make(Data, len(d))
	for k, v := range d {
		copied[k] = v
	}
	return copied
}

// Scope carries stable identifiers attached to every observability record.
type Scope struct {
	OrganizationID uint64
	ProjectID      uint64
	AssistantID    uint64
	ConversationID uint64
	ContextID      string
}

func (s Scope) WithRecord(record Scope) Scope {
	if validator.NotBlank(record.ContextID) {
		s.ContextID = record.ContextID
	}
	return s
}

// Record is the typed unit accepted by Recorder.
type Record interface {
	Kind() RecordKind
	Name() EventName
	Category() Category
	Level() Level
	Outcome() Outcome
	Title() string
	OccurredAt() time.Time
	ElapsedDuration() time.Duration
	Scope() Scope
	Attributes() Attributes
	Data() Data
	Validate() error
}

type BaseRecord struct {
	RecordName       EventName
	RecordScope      Scope
	RecordAttributes Attributes
	RecordData       Data
	RecordLevel      Level
	RecordOutcome    Outcome
	RecordTitle      string
	At               time.Time
	Elapsed          time.Duration
}

func (r BaseRecord) Name() EventName {
	return r.RecordName
}

func (r BaseRecord) Category() Category {
	return r.RecordName.Category()
}

func (r BaseRecord) Level() Level {
	if r.RecordLevel == "" {
		return LevelInfo
	}
	return r.RecordLevel
}

func (r BaseRecord) Outcome() Outcome {
	return r.RecordOutcome
}

func (r BaseRecord) Title() string {
	if r.RecordTitle != "" {
		return r.RecordTitle
	}
	return r.RecordName.String()
}

func (r BaseRecord) OccurredAt() time.Time {
	return r.At
}

func (r BaseRecord) ElapsedDuration() time.Duration {
	return r.Elapsed
}

func (r BaseRecord) Scope() Scope {
	return r.RecordScope
}

func (r BaseRecord) Attributes() Attributes {
	return r.RecordAttributes.Clone()
}

func (r BaseRecord) Data() Data {
	return r.RecordData.Clone()
}

type EventRecord struct {
	BaseRecord
}

func (r EventRecord) Kind() RecordKind {
	return RecordKindEvent
}

func (r EventRecord) Validate() error {
	return validateEventName(r.RecordName)
}

type CallEvent struct {
	BaseRecord
	Provider       string
	Direction      string
	Status         string
	Reason         string
	Error          string
	ProviderCallID string
}

func (e CallEvent) Kind() RecordKind {
	return RecordKindEvent
}

func (e CallEvent) Attributes() Attributes {
	attrs := e.BaseRecord.Attributes()
	attrs = attrs.Add(AttrProvider, e.Provider)
	attrs = attrs.Add(AttrDirection, e.Direction)
	attrs = attrs.Add(AttrStatus, e.Status)
	attrs = attrs.Add(AttrReason, e.Reason)
	attrs = attrs.Add(AttrError, e.Error)
	attrs = attrs.Add(AttrProviderCallID, e.ProviderCallID)
	return attrs
}

func (e CallEvent) Validate() error {
	return validateCategory(e.RecordName, CategoryCall)
}

type ConversationEvent struct {
	BaseRecord
	Status     string
	OldState   string
	NewState   string
	Source     string
	Transcript string
	Language   string
	SpeakerID  string
	ItemID     string
	Reason     string
	Error      string
}

func (e ConversationEvent) Kind() RecordKind {
	return RecordKindEvent
}

func (e ConversationEvent) Attributes() Attributes {
	attrs := e.BaseRecord.Attributes()
	attrs = attrs.Add(AttrStatus, e.Status)
	attrs = attrs.Add(AttrOldState, e.OldState)
	attrs = attrs.Add(AttrNewState, e.NewState)
	attrs = attrs.Add(AttrSource, e.Source)
	attrs = attrs.Add(AttrTranscript, e.Transcript)
	attrs = attrs.Add(AttrLanguage, e.Language)
	attrs = attrs.Add(AttrSpeakerID, e.SpeakerID)
	attrs = attrs.Add(AttrItemID, e.ItemID)
	attrs = attrs.Add(AttrReason, e.Reason)
	attrs = attrs.Add(AttrError, e.Error)
	return attrs
}

func (e ConversationEvent) Validate() error {
	return validateCategory(e.RecordName, CategoryConversation)
}

type TurnEvent struct {
	BaseRecord
	Status     string
	Reason     string
	Error      string
	InputText  string
	OutputText string
	Transcript string
}

func (e TurnEvent) Kind() RecordKind {
	return RecordKindEvent
}

func (e TurnEvent) Attributes() Attributes {
	attrs := e.BaseRecord.Attributes()
	attrs = attrs.Add(AttrStatus, e.Status)
	attrs = attrs.Add(AttrReason, e.Reason)
	attrs = attrs.Add(AttrError, e.Error)
	attrs = attrs.Add(AttrInputText, e.InputText)
	attrs = attrs.Add(AttrOutputText, e.OutputText)
	attrs = attrs.Add(AttrTranscript, e.Transcript)
	return attrs
}

func (e TurnEvent) Validate() error {
	return validateCategory(e.RecordName, CategoryTurn)
}

type ComponentEvent struct {
	BaseRecord
	Component string
	Provider  string
	Status    string
	Stage     string
	Reason    string
	Error     string
}

func (e ComponentEvent) Kind() RecordKind {
	return RecordKindEvent
}

func (e ComponentEvent) Attributes() Attributes {
	attrs := e.BaseRecord.Attributes()
	attrs = attrs.Add(AttrComponent, e.Component)
	attrs = attrs.Add(AttrProvider, e.Provider)
	attrs = attrs.Add(AttrStatus, e.Status)
	attrs = attrs.Add(AttrStage, e.Stage)
	attrs = attrs.Add(AttrReason, e.Reason)
	attrs = attrs.Add(AttrError, e.Error)
	return attrs
}

func (e ComponentEvent) Validate() error {
	return validateEventName(e.RecordName)
}

type AudioEvent struct {
	BaseRecord
	Provider   string
	Direction  string
	Codec      string
	SampleRate string
	Duration   string
	Status     string
	Error      string
}

func (e AudioEvent) Kind() RecordKind {
	return RecordKindEvent
}

func (e AudioEvent) Attributes() Attributes {
	attrs := e.BaseRecord.Attributes()
	attrs = attrs.Add(AttrProvider, e.Provider)
	attrs = attrs.Add(AttrDirection, e.Direction)
	attrs = attrs.Add(AttrCodec, e.Codec)
	attrs = attrs.Add(AttrSampleRate, e.SampleRate)
	attrs = attrs.Add(AttrDuration, e.Duration)
	attrs = attrs.Add(AttrStatus, e.Status)
	attrs = attrs.Add(AttrError, e.Error)
	return attrs
}

func (e AudioEvent) Validate() error {
	return validateCategory(e.RecordName, CategoryAudio)
}

type ProviderEvent struct {
	BaseRecord
	Provider   string
	Model      string
	Status     string
	Stage      string
	RequestID  string
	Transcript string
	Language   string
	Confidence string
	Reason     string
	Error      string
}

func (e ProviderEvent) Kind() RecordKind {
	return RecordKindEvent
}

func (e ProviderEvent) Attributes() Attributes {
	attrs := e.BaseRecord.Attributes()
	attrs = attrs.Add(AttrProvider, e.Provider)
	attrs = attrs.Add(AttrModel, e.Model)
	attrs = attrs.Add(AttrStatus, e.Status)
	attrs = attrs.Add(AttrStage, e.Stage)
	attrs = attrs.Add(AttrRequestID, e.RequestID)
	attrs = attrs.Add(AttrTranscript, e.Transcript)
	attrs = attrs.Add(AttrLanguage, e.Language)
	attrs = attrs.Add(AttrConfidence, e.Confidence)
	attrs = attrs.Add(AttrReason, e.Reason)
	attrs = attrs.Add(AttrError, e.Error)
	return attrs
}

func (e ProviderEvent) Validate() error {
	if err := validateEventName(e.RecordName); err != nil {
		return err
	}
	switch e.RecordName.Category() {
	case CategorySTT, CategoryTTS, CategoryLLM, CategoryVAD, CategoryEOS, CategoryDenoise:
		return nil
	default:
		return fmt.Errorf("observability: %q is not a provider event", e.RecordName)
	}
}

type TranscriptEvent struct {
	BaseRecord
	Provider   string
	Transcript string
	Language   string
	Status     string
	Error      string
}

func (e TranscriptEvent) Kind() RecordKind {
	return RecordKindEvent
}

func (e TranscriptEvent) Attributes() Attributes {
	attrs := e.BaseRecord.Attributes()
	attrs = attrs.Add(AttrProvider, e.Provider)
	attrs = attrs.Add(AttrTranscript, e.Transcript)
	attrs = attrs.Add(AttrLanguage, e.Language)
	attrs = attrs.Add(AttrStatus, e.Status)
	attrs = attrs.Add(AttrError, e.Error)
	return attrs
}

func (e TranscriptEvent) Validate() error {
	return validateCategory(e.RecordName, CategoryTranscript)
}

type ToolEvent struct {
	BaseRecord
	ToolName string
	Status   string
	Reason   string
	Error    string
}

func (e ToolEvent) Kind() RecordKind {
	return RecordKindEvent
}

func (e ToolEvent) Attributes() Attributes {
	attrs := e.BaseRecord.Attributes()
	attrs = attrs.Add(AttrToolName, e.ToolName)
	attrs = attrs.Add(AttrStatus, e.Status)
	attrs = attrs.Add(AttrReason, e.Reason)
	attrs = attrs.Add(AttrError, e.Error)
	return attrs
}

func (e ToolEvent) Validate() error {
	return validateCategory(e.RecordName, CategoryTool)
}

type WebhookEvent struct {
	BaseRecord
	Payload map[string]interface{}
}

func (e WebhookEvent) Kind() RecordKind {
	return RecordKindEvent
}

func (e WebhookEvent) Attributes() Attributes {
	return e.BaseRecord.Attributes()
}

func (e WebhookEvent) Data() Data {
	data := e.BaseRecord.Data()
	if data == nil {
		data = Data{}
	}
	if validator.NonNil(e.Payload) {
		data["payload"] = e.Payload
	}
	return data
}

func (e WebhookEvent) Validate() error {
	if err := validateEventName(e.RecordName); err != nil {
		return err
	}
	switch e.RecordName.Category() {
	case CategoryCall, CategoryConversation:
		return nil
	default:
		return fmt.Errorf("observability: %q is not a webhook event", e.RecordName)
	}
}

type UsageEvent struct {
	BaseRecord
	Component     string
	Provider      string
	UsageCategory string
	Duration      time.Duration
}

func (e UsageEvent) Kind() RecordKind {
	return RecordKindEvent
}

func (e UsageEvent) Attributes() Attributes {
	attrs := e.BaseRecord.Attributes()
	attrs = attrs.Add(AttrComponent, e.Component)
	attrs = attrs.Add(AttrProvider, e.Provider)
	attrs = attrs.Add(AttrUsageCategory, e.UsageCategory)
	if e.Duration > 0 {
		attrs = attrs.Add(AttrDuration, fmt.Sprintf("%d", e.Duration.Milliseconds()))
	}
	return attrs
}

func (e UsageEvent) Validate() error {
	if err := validateCategory(e.RecordName, CategoryUsage); err != nil {
		return err
	}
	if !validator.NotBlank(e.Component) {
		return errors.New("observability: usage component is required")
	}
	if e.Duration <= 0 {
		return errors.New("observability: usage duration must be greater than zero")
	}
	return nil
}

type ErrorEvent struct {
	BaseRecord
	Component string
	Stage     string
	Reason    string
	Error     string
}

func (e ErrorEvent) Kind() RecordKind {
	return RecordKindEvent
}

func (e ErrorEvent) Level() Level {
	if e.RecordLevel == "" {
		return LevelError
	}
	return e.RecordLevel
}

func (e ErrorEvent) Outcome() Outcome {
	if e.RecordOutcome == "" {
		return OutcomeFailure
	}
	return e.RecordOutcome
}

func (e ErrorEvent) Attributes() Attributes {
	attrs := e.BaseRecord.Attributes()
	attrs = attrs.Add(AttrComponent, e.Component)
	attrs = attrs.Add(AttrStage, e.Stage)
	attrs = attrs.Add(AttrReason, e.Reason)
	attrs = attrs.Add(AttrError, e.Error)
	return attrs
}

func (e ErrorEvent) Validate() error {
	return validateCategory(e.RecordName, CategoryError)
}

type Metric struct {
	Name        string
	Value       string
	Description string
	Attributes  Attributes
}

type MetricRecord struct {
	BaseRecord
	Metrics []Metric
}

func (r MetricRecord) Kind() RecordKind {
	return RecordKindMetric
}

func (r MetricRecord) Name() EventName {
	if r.RecordName == "" {
		return EventName("metric.recorded")
	}
	return r.RecordName
}

func (r MetricRecord) Title() string {
	if r.RecordTitle != "" {
		return r.RecordTitle
	}
	return r.Name().String()
}

func (r MetricRecord) Category() Category {
	return CategoryMetric
}

func (r MetricRecord) Validate() error {
	if len(r.Metrics) == 0 {
		return errors.New("observability: at least one metric is required")
	}
	for i, metric := range r.Metrics {
		if !validator.NotBlank(metric.Name) {
			return fmt.Errorf("observability: metric[%d] name is required", i)
		}
	}
	return nil
}

type Metadata struct {
	Key   string
	Value string
}

type MetadataRecord struct {
	BaseRecord
	Metadata []Metadata
}

func (r MetadataRecord) Kind() RecordKind {
	return RecordKindMetadata
}

func (r MetadataRecord) Name() EventName {
	if r.RecordName == "" {
		return EventName("metadata.recorded")
	}
	return r.RecordName
}

func (r MetadataRecord) Title() string {
	if r.RecordTitle != "" {
		return r.RecordTitle
	}
	return r.Name().String()
}

func (r MetadataRecord) Category() Category {
	return CategoryMetadata
}

func (r MetadataRecord) Validate() error {
	if len(r.Metadata) == 0 {
		return errors.New("observability: at least one metadata entry is required")
	}
	for i, metadata := range r.Metadata {
		if !validator.NotBlank(metadata.Key) {
			return fmt.Errorf("observability: metadata[%d] key is required", i)
		}
	}
	return nil
}

func validateEventName(name EventName) error {
	if !validator.NotBlank(name.String()) {
		return errors.New("observability: event name is required")
	}
	if !name.IsKnown() {
		return fmt.Errorf("observability: unknown event name %q", name)
	}
	return nil
}

func validateCategory(name EventName, category Category) error {
	if err := validateEventName(name); err != nil {
		return err
	}
	if !name.HasCategory(category) {
		return fmt.Errorf("observability: %q is not a %s event", name, category)
	}
	return nil
}
