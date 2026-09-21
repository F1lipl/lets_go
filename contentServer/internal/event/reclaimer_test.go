package event

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type reclaimerTestRepository struct {
	mu      sync.Mutex
	results []RecoveryResult
	calls   chan struct{}
}

func (r *reclaimerTestRepository) RecoverExpired(context.Context, int) (RecoveryResult, error) {
	select {
	case r.calls <- struct{}{}:
	default:
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.results) == 0 {
		return RecoveryResult{}, nil
	}
	result := r.results[0]
	r.results = r.results[1:]
	return result, nil
}

type reclaimerTestNotifier struct{ notified chan struct{} }

func (n *reclaimerTestNotifier) Notify() {
	select {
	case n.notified <- struct{}{}:
	default:
	}
}

func TestReclaimerRecoversImmediatelyAndNotifiesScheduler(t *testing.T) {
	repo := &reclaimerTestRepository{
		results: []RecoveryResult{{Scanned: 1, Recovered: 1}},
		calls:   make(chan struct{}, 2),
	}
	notifier := &reclaimerTestNotifier{notified: make(chan struct{}, 1)}
	reclaimer, err := NewReclaimer(repo, notifier, ReclaimerConfig{
		PollInterval: time.Hour,
		ErrorBackoff: time.Second,
		BatchSize:    10,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- reclaimer.Run(ctx) }()

	select {
	case <-repo.calls:
	case <-time.After(time.Second):
		t.Fatal("startup recovery scan did not run")
	}
	select {
	case <-notifier.notified:
	case <-time.After(time.Second):
		t.Fatal("scheduler was not notified after task recovery")
	}

	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("reclaimer exit: %v", err)
	}
}

func TestNewReclaimerRejectsOversizedBatch(t *testing.T) {
	_, err := NewReclaimer(
		&reclaimerTestRepository{},
		&reclaimerTestNotifier{},
		ReclaimerConfig{BatchSize: 501},
	)
	if err == nil {
		t.Fatal("expected oversized batch to be rejected")
	}
}
