package internal_channel_output

import "testing"

func TestHealthStats_Snapshot(t *testing.T) {
	s := NewHealthStats()
	s.OnTickHealth(TickHealth{Active: true})
	s.OnTickHealth(TickHealth{Idle: true, LateTick: true})
	s.OnTickHealth(TickHealth{Idle: true, SendError: true})

	got := s.Snapshot()
	if got.Ticks != 3 {
		t.Fatalf("ticks=%d want=3", got.Ticks)
	}
	if got.ActiveTicks != 1 {
		t.Fatalf("active=%d want=1", got.ActiveTicks)
	}
	if got.IdleTicks != 2 {
		t.Fatalf("idle=%d want=2", got.IdleTicks)
	}
	if got.LateTicks != 1 {
		t.Fatalf("late=%d want=1", got.LateTicks)
	}
	if got.SendErrors != 1 {
		t.Fatalf("send_errors=%d want=1", got.SendErrors)
	}
	if got.IdleRatio < 0.66 || got.IdleRatio > 0.67 {
		t.Fatalf("idle_ratio=%f want~0.666", got.IdleRatio)
	}
}
