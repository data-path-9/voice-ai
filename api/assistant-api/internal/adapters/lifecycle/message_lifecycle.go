// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package lifecycle

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

// MessageState tracks current message/turn processing lifecycle.
type MessageState int

const (
	Unknown       MessageState = 1
	Interrupt     MessageState = 6
	Interrupted   MessageState = 7
	LLMGenerating MessageState = 8
	LLMGenerated  MessageState = 5
)

func (s MessageState) String() string {
	switch s {
	case Unknown:
		return "Unknown"
	case LLMGenerated:
		return "LLMGenerated"
	case Interrupt:
		return "Interrupt"
	case Interrupted:
		return "Interrupted"
	case LLMGenerating:
		return "LLMGenerating"
	default:
		return "InvalidState"
	}
}

type MessageLifecycle interface {
	Current() MessageState
	CanBe(MessageState) bool
	Transition(MessageState) error
	ContextID() string
	SetContextID(string)
	Mode() type_enums.MessageMode
	SetMode(type_enums.MessageMode)
}

type messageLifecycle struct {
	mu            sync.RWMutex
	state         MessageState
	contextID     string
	mode          type_enums.MessageMode
	nextContextID func() string
}

func NewMessageLifecycle() MessageLifecycle {
	return NewMessageLifecycleWithState(Unknown, uuid.NewString(), type_enums.TextMode, uuid.NewString)
}

func NewMessageLifecycleWithState(
	initial MessageState,
	initialContextID string,
	initialMode type_enums.MessageMode,
	nextContextID func() string,
) MessageLifecycle {
	if initialContextID == "" {
		initialContextID = uuid.NewString()
	}
	if nextContextID == nil {
		nextContextID = uuid.NewString
	}
	if initialMode == "" {
		initialMode = type_enums.TextMode
	}
	return &messageLifecycle{
		state:         initial,
		contextID:     initialContextID,
		mode:          initialMode,
		nextContextID: nextContextID,
	}
}

func (l *messageLifecycle) Current() MessageState {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.state
}

func (l *messageLifecycle) CanBe(next MessageState) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return canTransitionMessage(l.state, next) == nil
}

func (l *messageLifecycle) Transition(next MessageState) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := canTransitionMessage(l.state, next); err != nil {
		return err
	}
	if next == Interrupted {
		nctx := l.nextContextID()
		if nctx == "" {
			return fmt.Errorf("Transition: generated empty context id")
		}
		l.contextID = nctx
	}
	l.state = next
	return nil
}

func (l *messageLifecycle) ContextID() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.contextID
}

func (l *messageLifecycle) SetContextID(id string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.contextID = id
}

func (l *messageLifecycle) Mode() type_enums.MessageMode {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.mode
}

func (l *messageLifecycle) SetMode(mode type_enums.MessageMode) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.mode = mode
}

func canTransitionMessage(current, next MessageState) error {
	switch next {
	case Unknown:
		return fmt.Errorf("Transition: cannot transition to Unknown state")
	case Interrupt:
		if current == Interrupted || current == Interrupt {
			return fmt.Errorf("Transition: cannot soft-interrupt from state %s", current)
		}
		if current == Unknown {
			return fmt.Errorf("Transition: nothing active to soft-interrupt in state %s", current)
		}
	case Interrupted:
		if current == Interrupted {
			return fmt.Errorf("Transition: already interrupted")
		}
	}
	return nil
}
