// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package variable

import (
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
)

// VariableSource is a point-in-time snapshot of a Communication's
// variable-resolvable state. All fields are copied at construction so the
// source is safe to use after the session disconnects.
type VariableSource struct {
	assistant    *AssistantInfo
	conversation *ConversationInfo
	histories    []ConversationMessageInfo
	args         map[string]any
	metadata     map[string]any
	options      map[string]any
	mode         string
	now          func() time.Time
}

// NewCommunicationSource snapshots the Communication state at call time.
// The returned VariableSource holds no reference to the Communication.
func NewCommunicationSource(c internal_type.Communication) *VariableSource {
	s := &VariableSource{now: time.Now}
	if c == nil {
		return s
	}

	if a := c.Assistant(); a != nil {
		s.assistant = &AssistantInfo{
			ID:          a.Id,
			VersionID:   a.AssistantProviderId,
			Name:        a.Name,
			Language:    a.Language,
			Description: a.Description,
		}
	}

	if conv := c.Conversation(); conv != nil {
		s.conversation = &ConversationInfo{
			ID:          conv.Id,
			Identifier:  conv.Identifier,
			Source:      string(conv.Source),
			Direction:   conv.Direction.String(),
			CreatedDate: time.Time(conv.CreatedDate),
		}
	}

	if msgs := c.GetHistories(); len(msgs) > 0 {
		s.histories = make([]ConversationMessageInfo, 0, len(msgs))
		for _, m := range msgs {
			s.histories = append(s.histories, ConversationMessageInfo{Role: m.Role(), Content: m.Content()})
		}
	}

	s.args = utils.CloneMap(c.GetArgs())
	s.metadata = utils.CloneMap(c.GetMetadata())
	s.options = utils.CloneMap(c.GetOptions())
	s.mode = c.GetMode().String()
	return s
}

// SourceOption configures a VariableSource via NewVariableSource.
type SourceOption func(*VariableSource)

// NewVariableSource builds a VariableSource from functional options.
// Useful in tests and code that constructs sources outside of a Communication.
func NewVariableSource(opts ...SourceOption) *VariableSource {
	s := &VariableSource{now: time.Now}
	for _, o := range opts {
		o(s)
	}
	return s
}

func WithAssistant(a *AssistantInfo) SourceOption { return func(s *VariableSource) { s.assistant = a } }
func WithConversation(c *ConversationInfo) SourceOption {
	return func(s *VariableSource) { s.conversation = c }
}
func WithHistories(h []ConversationMessageInfo) SourceOption {
	return func(s *VariableSource) { s.histories = h }
}
func WithArguments(a map[string]any) SourceOption { return func(s *VariableSource) { s.args = a } }
func WithMetadata(m map[string]any) SourceOption  { return func(s *VariableSource) { s.metadata = m } }
func WithOptions(o map[string]any) SourceOption   { return func(s *VariableSource) { s.options = o } }
func WithMode(m string) SourceOption              { return func(s *VariableSource) { s.mode = m } }

func WithClockFunc(now func() time.Time) SourceOption {
	return func(s *VariableSource) {
		if now != nil {
			s.now = now
		}
	}
}

// WithClock overrides the clock for tests.
func (s *VariableSource) WithClock(now func() time.Time) *VariableSource {
	if now != nil {
		s.now = now
	}
	return s
}

func (s *VariableSource) Assistant() *AssistantInfo            { return s.assistant }
func (s *VariableSource) Conversation() *ConversationInfo      { return s.conversation }
func (s *VariableSource) Histories() []ConversationMessageInfo { return s.histories }
func (s *VariableSource) Arguments() map[string]any            { return s.args }
func (s *VariableSource) Metadata() map[string]any             { return s.metadata }
func (s *VariableSource) Options() map[string]any              { return s.options }
func (s *VariableSource) Mode() string                         { return s.mode }
func (s *VariableSource) Now() time.Time                       { return s.now() }
