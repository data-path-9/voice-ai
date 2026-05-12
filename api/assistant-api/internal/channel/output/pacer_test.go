package internal_channel_output

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type testPacerProvider struct {
	next []byte
	idle []byte
}

func (p *testPacerProvider) NextFrame() []byte { return p.next }
func (p *testPacerProvider) IdleFrame() []byte { return p.idle }

type testPacerConsumer struct {
	count atomic.Int32
	err   bool
}

func (c *testPacerConsumer) ConsumeFrame(_ []byte) error {
	c.count.Add(1)
	if c.err {
		return context.Canceled
	}
	return nil
}

func TestPacer_UsesIdleFrameWhenPrimaryEmpty(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	provider := &testPacerProvider{
		next: nil,
		idle: []byte{1, 2, 3},
	}
	consumer := &testPacerConsumer{}

	done := make(chan struct{})
	go func() {
		(&Pacer{
			FrameDuration: 10 * time.Millisecond,
			Provider:      provider,
			Consumer:      consumer,
		}).Run(ctx)
		close(done)
	}()

	time.Sleep(35 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("pacer did not stop after context cancellation")
	}

	if consumer.count.Load() == 0 {
		t.Fatal("expected pacer to deliver at least one idle frame")
	}
}

func TestPacer_StopsQuicklyOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	provider := &testPacerProvider{
		next: []byte{9},
	}
	consumer := &testPacerConsumer{}

	done := make(chan struct{})
	go func() {
		(&Pacer{
			FrameDuration: 200 * time.Millisecond,
			Provider:      provider,
			Consumer:      consumer,
		}).Run(ctx)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("pacer did not stop quickly after cancellation")
	}
}

func TestPacer_ReportsHealth(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	provider := &testPacerProvider{
		next: nil,
		idle: []byte{1},
	}
	consumer := &testPacerConsumer{err: true}
	health := NewHealthStats()

	done := make(chan struct{})
	go func() {
		(&Pacer{
			FrameDuration: 10 * time.Millisecond,
			Provider:      provider,
			Consumer:      consumer,
			Health:        health,
		}).Run(ctx)
		close(done)
	}()

	time.Sleep(35 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("pacer did not stop after context cancellation")
	}

	s := health.Snapshot()
	if s.Ticks == 0 {
		t.Fatal("expected non-zero ticks")
	}
	if s.IdleTicks == 0 {
		t.Fatal("expected idle ticks to be recorded")
	}
	if s.SendErrors == 0 {
		t.Fatal("expected send errors to be recorded")
	}
}
