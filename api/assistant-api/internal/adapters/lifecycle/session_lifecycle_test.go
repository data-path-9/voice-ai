package lifecycle

import "testing"

func TestSessionLifecycle_HappyPath(t *testing.T) {
	l := NewSessionLifecycle()

	if err := l.Transition(EventConnectRequested); err != nil {
		t.Fatalf("connect transition failed: %v", err)
	}
	if got := l.Current(); got != StateInitializing {
		t.Fatalf("state mismatch: got=%v want=%v", got, StateInitializing)
	}

	if err := l.Transition(EventInitializationCompleted); err != nil {
		t.Fatalf("init complete transition failed: %v", err)
	}
	if got := l.Current(); got != StateReady {
		t.Fatalf("state mismatch: got=%v want=%v", got, StateReady)
	}

	if err := l.Transition(EventSwitchRequested); err != nil {
		t.Fatalf("switch requested transition failed: %v", err)
	}
	if got := l.Current(); got != StateSwitching {
		t.Fatalf("state mismatch: got=%v want=%v", got, StateSwitching)
	}

	if err := l.Transition(EventSwitchCompleted); err != nil {
		t.Fatalf("switch completed transition failed: %v", err)
	}
	if got := l.Current(); got != StateReady {
		t.Fatalf("state mismatch: got=%v want=%v", got, StateReady)
	}

	if err := l.Transition(EventDisconnectRequested); err != nil {
		t.Fatalf("disconnect requested transition failed: %v", err)
	}
	if got := l.Current(); got != StateDisconnecting {
		t.Fatalf("state mismatch: got=%v want=%v", got, StateDisconnecting)
	}

	if err := l.Transition(EventDisconnectCompleted); err != nil {
		t.Fatalf("disconnect completed transition failed: %v", err)
	}
	if got := l.Current(); got != StateDisconnected {
		t.Fatalf("state mismatch: got=%v want=%v", got, StateDisconnected)
	}
}

func TestSessionLifecycle_FatalSwitchFailure(t *testing.T) {
	l := NewSessionLifecycleWithState(StateReady)
	if err := l.Transition(EventSwitchRequested); err != nil {
		t.Fatalf("switch requested transition failed: %v", err)
	}
	if err := l.Transition(EventSwitchFailedFatal); err != nil {
		t.Fatalf("fatal switch failure transition failed: %v", err)
	}
	if got := l.Current(); got != StateFailed {
		t.Fatalf("state mismatch: got=%v want=%v", got, StateFailed)
	}
}

func TestSessionLifecycle_InvalidTransition(t *testing.T) {
	l := NewSessionLifecycleWithState(StateReady)
	if err := l.Transition(EventInitializationCompleted); err == nil {
		t.Fatalf("expected invalid transition error")
	}
}
