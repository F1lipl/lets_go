package event

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type pipelineTestTaskStore struct {
	mu       sync.Mutex
	ready    bool
	claimed  bool
	executed chan struct{}
}

func (s *pipelineTestTaskStore) makeReady() {
	s.mu.Lock()
	s.ready = true
	s.mu.Unlock()
}

func (s *pipelineTestTaskStore) Claim(_ context.Context, limit int, _ time.Duration) ([]ClaimedTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 || !s.ready || s.claimed {
		return nil, nil
	}
	s.claimed = true
	return []ClaimedTask{{
		Context:    TaskContext{EventId: "event-1", ConsumerName: "post_card"},
		ClaimToken: "claim-1",
	}}, nil
}

func (s *pipelineTestTaskStore) Execute(ctx context.Context, task ClaimedTask, apply func(context.Context, sqlx.Session, *TaskContext) error) error {
	if err := apply(ctx, nil, &task.Context); err != nil {
		return err
	}
	select {
	case s.executed <- struct{}{}:
	default:
	}
	return nil
}

func (*pipelineTestTaskStore) ScheduleRetry(context.Context, ClaimedTask, error, time.Duration, bool) error {
	return nil
}

func (*pipelineTestTaskStore) RecoverExpired(context.Context, int) (RecoveryResult, error) {
	return RecoveryResult{}, nil
}

type pipelineTestDispatchRepository struct {
	mu    sync.Mutex
	armed bool
	calls chan struct{}
	store *pipelineTestTaskStore
}

func (r *pipelineTestDispatchRepository) arm() {
	r.mu.Lock()
	r.armed = true
	r.mu.Unlock()
}

func (r *pipelineTestDispatchRepository) FanOut(context.Context, int, ConsumerResolver) (DispatchResult, error) {
	select {
	case r.calls <- struct{}{}:
	default:
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.armed {
		return DispatchResult{}, nil
	}
	r.armed = false
	r.store.makeReady()
	return DispatchResult{ScannedEvents: 1, DispatchedEvents: 1, CreatedTasks: 1}, nil
}

func TestEventPipelineRunsAllStages(t *testing.T) {
	store := &pipelineTestTaskStore{executed: make(chan struct{}, 1)}
	dispatchRepo := &pipelineTestDispatchRepository{
		calls: make(chan struct{}, 4),
		store: store,
	}
	registry, err := NewConsumerRegistry(map[string][]string{
		"PostPublished": {"post_card"},
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := TaskHandlerFunc(func(context.Context, sqlx.Session, *TaskContext) error {
		return nil
	})
	pipeline, err := NewEventPipeline(dispatchRepo, store, registry, handler, EventPipelineConfig{
		WorkerCount: 1,
		Scheduler: SchedulerConfig{
			PollInterval:  time.Hour,
			ErrorBackoff:  time.Second,
			LeaseDuration: time.Minute,
			MaxClaimBatch: 1,
		},
		Dispatcher: DispatcherConfig{
			PollInterval: time.Hour,
			ErrorBackoff: time.Second,
			BatchSize:    1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- pipeline.Run(ctx) }()
	select {
	case <-pipeline.Ready():
	case <-time.After(time.Second):
		t.Fatal("pipeline did not become ready")
	}
	select {
	case <-dispatchRepo.calls:
	case <-time.After(time.Second):
		t.Fatal("initial dispatcher scan did not run")
	}

	dispatchRepo.arm()
	pipeline.NotifyOutboxCommitted()
	select {
	case <-store.executed:
	case <-time.After(time.Second):
		t.Fatal("event did not pass through dispatcher, scheduler and worker")
	}

	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("pipeline exit: %v", err)
	}
	select {
	case <-pipeline.Done():
	default:
		t.Fatal("pipeline done signal was not closed")
	}
	if err := pipeline.Run(context.Background()); !errors.Is(err, ErrPipelineAlreadyRunning) {
		t.Fatalf("second pipeline run: %v", err)
	}
}
