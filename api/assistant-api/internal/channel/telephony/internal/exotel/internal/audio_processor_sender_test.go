package internal_exotel

import (
	"testing"

	internal_telephony_output "github.com/rapidaai/api/assistant-api/internal/channel/output"
)

func TestAudioProcessor_OutputHealthObserverRecordsTicks(t *testing.T) {
	audioProcessor, err := NewAudioProcessor(nil)
	if err != nil {
		t.Fatalf("NewAudioProcessor error: %v", err)
	}

	audioProcessor.OnTickHealth(internal_telephony_output.TickHealth{Active: true})
	audioProcessor.OnTickHealth(internal_telephony_output.TickHealth{Idle: true, SendError: true})

	stats := audioProcessor.OutputHealthSnapshot()
	if stats.Ticks != 2 {
		t.Fatalf("ticks=%d want=2", stats.Ticks)
	}
	if stats.ActiveTicks != 1 {
		t.Fatalf("activeTicks=%d want=1", stats.ActiveTicks)
	}
	if stats.IdleTicks != 1 {
		t.Fatalf("idleTicks=%d want=1", stats.IdleTicks)
	}
	if stats.SendErrors != 1 {
		t.Fatalf("sendErrors=%d want=1", stats.SendErrors)
	}
}
