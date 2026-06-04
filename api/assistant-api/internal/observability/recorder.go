// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package observability

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Recorder accepts typed observability records and fans them out to collectors.
type Recorder interface {
	Record(ctx context.Context, record Record) error
	Shutdown(ctx context.Context) error
}

type Config struct {
	Scope  Scope
	Clock  func() time.Time
	NewID  func() string
	Buffer int
}

type Envelope struct {
	ID         string
	Kind       RecordKind
	Name       EventName
	Category   Category
	Level      Level
	Outcome    Outcome
	Title      string
	Scope      Scope
	Attributes Attributes
	Data       Data
	Record     Record
	OccurredAt time.Time
	ReceivedAt time.Time
	Duration   time.Duration
}

type recorder struct {
	scope  Scope
	clock  func() time.Time
	newID  func() string
	fanout Collector
	queue  chan Envelope
	done   chan struct{}
	closed bool
	mu     sync.RWMutex
	errMu  sync.Mutex
	errs   []error
}

const defaultBufferSize = 1024

var (
	ErrRecorderClosed = errors.New("observability: recorder is closed")
	ErrBufferFull     = errors.New("observability: recorder buffer is full")
)

func New(cfg Config, collectors ...Collector) Recorder {
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	newID := cfg.NewID
	if newID == nil {
		newID = func() string { return uuid.NewString() }
	}
	buffer := cfg.Buffer
	if buffer <= 0 {
		buffer = defaultBufferSize
	}
	r := &recorder{
		scope:  cfg.Scope,
		clock:  clock,
		newID:  newID,
		fanout: NewCollectors(collectors...),
		queue:  make(chan Envelope, buffer),
		done:   make(chan struct{}),
	}
	go r.run()
	return r
}

func (r *recorder) Record(ctx context.Context, record Record) error {
	if record == nil {
		return errors.New("observability: record is nil")
	}
	if err := record.Validate(); err != nil {
		return err
	}
	now := r.clock()
	occurredAt := record.OccurredAt()
	if occurredAt.IsZero() {
		occurredAt = now
	}
	envelope := Envelope{
		ID:         r.newID(),
		Kind:       record.Kind(),
		Name:       record.Name(),
		Category:   record.Category(),
		Level:      record.Level(),
		Outcome:    record.Outcome(),
		Title:      record.Title(),
		Scope:      r.scope.WithRecord(record.Scope()),
		Attributes: record.Attributes(),
		Data:       record.Data(),
		Record:     record,
		OccurredAt: occurredAt,
		ReceivedAt: now,
		Duration:   record.ElapsedDuration(),
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.closed {
		return ErrRecorderClosed
	}
	select {
	case r.queue <- envelope:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrBufferFull
	}
}

func (r *recorder) run() {
	defer close(r.done)
	for envelope := range r.queue {
		if err := r.fanout.Collect(context.Background(), envelope); err != nil {
			r.addError(err)
		}
	}
}

func (r *recorder) Shutdown(ctx context.Context) error {
	r.mu.Lock()
	if !r.closed {
		r.closed = true
		close(r.queue)
	}
	r.mu.Unlock()

	select {
	case <-r.done:
	case <-ctx.Done():
		return ctx.Err()
	}

	if err := r.fanout.Shutdown(ctx); err != nil {
		r.addError(err)
	}
	return r.errors()
}

func (r *recorder) addError(err error) {
	r.errMu.Lock()
	defer r.errMu.Unlock()
	r.errs = append(r.errs, err)
}

func (r *recorder) errors() error {
	r.errMu.Lock()
	defer r.errMu.Unlock()
	return errors.Join(r.errs...)
}
