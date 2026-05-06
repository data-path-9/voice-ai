package lifecycle

import (
	"testing"

	type_enums "github.com/rapidaai/pkg/types/enums"
)

func TestMessageLifecycle_InterruptFromUnknown_Fails(t *testing.T) {
	l := NewMessageLifecycleWithState(Unknown, "ctx", type_enums.MessageMode(""), func() string { return "ctx2" })
	if err := l.Transition(Interrupt); err == nil {
		t.Fatalf("expected error when transitioning Unknown -> Interrupt")
	}
}

func TestMessageLifecycle_Interrupted_Succeeds(t *testing.T) {
	l := NewMessageLifecycleWithState(LLMGenerating, "ctx-old", type_enums.MessageMode(""), func() string { return "ctx-new" })
	if err := l.Transition(Interrupted); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := l.Current(); got != Interrupted {
		t.Fatalf("unexpected state, got=%v want=%v", got, Interrupted)
	}
	if got := l.ContextID(); got != "ctx-new" {
		t.Fatalf("unexpected context id, got=%s want=%s", got, "ctx-new")
	}
}

func TestMessageLifecycle_LLMGeneratingToLLMGenerated(t *testing.T) {
	l := NewMessageLifecycleWithState(LLMGenerating, "ctx", type_enums.MessageMode(""), func() string { return "ignored" })
	if err := l.Transition(LLMGenerated); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := l.Current(); got != LLMGenerated {
		t.Fatalf("unexpected state, got=%v want=%v", got, LLMGenerated)
	}
}
