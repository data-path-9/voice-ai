// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package lifecycle

import (
	"fmt"
	"sync"
)

type SessionState uint8

const (
	StateNew SessionState = iota
	StateInitializing
	StateReady
	StateSwitching
	StateDisconnecting
	StateDisconnected
	StateFailed
)

func (s SessionState) String() string {
	switch s {
	case StateNew:
		return "new"
	case StateInitializing:
		return "initializing"
	case StateReady:
		return "ready"
	case StateSwitching:
		return "switching"
	case StateDisconnecting:
		return "disconnecting"
	case StateDisconnected:
		return "disconnected"
	case StateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

type SessionEvent uint8

const (
	EventConnectRequested SessionEvent = iota + 1
	EventInitializationCompleted
	EventInitializationFailed
	EventSwitchRequested
	EventSwitchCompleted
	EventSwitchFailedRecoverable
	EventSwitchFailedFatal
	EventDisconnectRequested
	EventDisconnectCompleted
)

func (e SessionEvent) String() string {
	switch e {
	case EventConnectRequested:
		return "connect_requested"
	case EventInitializationCompleted:
		return "initialization_completed"
	case EventInitializationFailed:
		return "initialization_failed"
	case EventSwitchRequested:
		return "switch_requested"
	case EventSwitchCompleted:
		return "switch_completed"
	case EventSwitchFailedRecoverable:
		return "switch_failed_recoverable"
	case EventSwitchFailedFatal:
		return "switch_failed_fatal"
	case EventDisconnectRequested:
		return "disconnect_requested"
	case EventDisconnectCompleted:
		return "disconnect_completed"
	default:
		return "unknown"
	}
}

type SessionLifecycle interface {
	Current() SessionState
	CanBe(SessionEvent) bool
	Transition(SessionEvent) error
}

type sessionLifecycle struct {
	mu    sync.RWMutex
	state SessionState
}

func NewSessionLifecycle() SessionLifecycle {
	return NewSessionLifecycleWithState(StateNew)
}

func NewSessionLifecycleWithState(initial SessionState) SessionLifecycle {
	return &sessionLifecycle{state: initial}
}

func (l *sessionLifecycle) Current() SessionState {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.state
}

func (l *sessionLifecycle) CanBe(event SessionEvent) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	_, err := nextSessionState(l.state, event)
	return err == nil
}

func (l *sessionLifecycle) Transition(event SessionEvent) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	next, err := nextSessionState(l.state, event)
	if err != nil {
		return err
	}
	l.state = next
	return nil
}

func nextSessionState(current SessionState, event SessionEvent) (SessionState, error) {
	switch current {
	case StateNew:
		switch event {
		case EventConnectRequested:
			return StateInitializing, nil
		case EventDisconnectRequested:
			return StateDisconnecting, nil
		}
	case StateInitializing:
		switch event {
		case EventInitializationCompleted:
			return StateReady, nil
		case EventInitializationFailed:
			return StateFailed, nil
		case EventDisconnectRequested:
			return StateDisconnecting, nil
		}
	case StateReady:
		switch event {
		case EventSwitchRequested:
			return StateSwitching, nil
		case EventDisconnectRequested:
			return StateDisconnecting, nil
		}
	case StateSwitching:
		switch event {
		case EventSwitchCompleted:
			return StateReady, nil
		case EventSwitchFailedRecoverable:
			return StateReady, nil
		case EventSwitchFailedFatal:
			return StateFailed, nil
		case EventDisconnectRequested:
			return StateDisconnecting, nil
		}
	case StateFailed:
		switch event {
		case EventDisconnectRequested:
			return StateDisconnecting, nil
		}
	case StateDisconnecting:
		if event == EventDisconnectCompleted {
			return StateDisconnected, nil
		}
	case StateDisconnected:
	}

	return current, fmt.Errorf("invalid session lifecycle transition: state=%s event=%s", current.String(), event.String())
}
