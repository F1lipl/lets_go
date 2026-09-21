package event

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type schedulerTestClaimer struct {
	mu     sync.Mutex
	limits []int
	tasks  []ClaimedTask
	calls  chan struct{}
}

func (c *schedulerTestClaimer) Claim(_ context.Context, limit int, _ time.Duration) ([]ClaimedTask, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.calls != nil {
		select {
		case c.calls <- struct{}{}:
		default:
		}
	}
	c.limits = append(c.limits, limit)
	if len(c.tasks) == 0 {
		return nil, nil
	}
	if limit > len(c.tasks) {
		limit = len(c.tasks)
	}
	claimed := append([]ClaimedTask(nil), c.tasks[:limit]...)
	c.tasks = c.tasks[limit:]
	return claimed, nil
}

type schedulerTestWorker struct {
	ready chan struct{}
	done  chan struct{}

	mu        sync.Mutex
	capacity  int
	submitted []ClaimedTask
	notify    chan struct{}
	space     chan struct{}
}

func newSchedulerTestWorker(capacity int) *schedulerTestWorker {
	ready := make(chan struct{})
	close(ready)
	return &schedulerTestWorker{
		ready:    ready,
		done:     make(chan struct{}),
		capacity: capacity,
		notify:   make(chan struct{}, capacity),
		space:    make(chan struct{}, 1),
	}
}

func (w *schedulerTestWorker) Submit(_ context.Context, task ClaimedTask) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.submitted) >= w.capacity {
		return errors.New("worker queue overflow")
	}
	w.submitted = append(w.submitted, task)
	w.notify <- struct{}{}
	return nil
}

func (w *schedulerTestWorker) QueueStats() WorkerQueueStats {
	w.mu.Lock()
	defer w.mu.Unlock()
	queued := len(w.submitted)
	return WorkerQueueStats{Queued: queued, Capacity: w.capacity, Available: w.capacity - queued}
}

func (w *schedulerTestWorker) Ready() <-chan struct{} { return w.ready }
func (w *schedulerTestWorker) Done() <-chan struct{}  { return w.done }
func (w *schedulerTestWorker) SpaceAvailable() <-chan struct{} {
	return w.space
}

func TestSchedulerClaimsOnlyAvailableQueueCapacity(t *testing.T) {
	claimer := &schedulerTestClaimer{tasks: []ClaimedTask{
		{Context: TaskContext{EventId: "event-1", ConsumerName: "card"}, ClaimToken: "token-1"},
		{Context: TaskContext{EventId: "event-2", ConsumerName: "card"}, ClaimToken: "token-2"},
		{Context: TaskContext{EventId: "event-3", ConsumerName: "card"}, ClaimToken: "token-3"},
	}}
	worker := newSchedulerTestWorker(2)
	scheduler, err := NewScheduler(claimer, worker, SchedulerConfig{
		PollInterval:  10 * time.Millisecond,
		ErrorBackoff:  10 * time.Millisecond,
		LeaseDuration: time.Minute,
		MaxClaimBatch: 10,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- scheduler.Run(ctx) }()
	for i := 0; i < 2; i++ {
		select {
		case <-worker.notify:
		case <-time.After(time.Second):
			t.Fatal("scheduler did not submit claimed task")
		}
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("scheduler exit: %v", err)
	}

	claimer.mu.Lock()
	defer claimer.mu.Unlock()
	if len(claimer.limits) == 0 || claimer.limits[0] != 2 {
		t.Fatalf("first claim limit = %v, want 2", claimer.limits)
	}
	if len(claimer.tasks) != 1 {
		t.Fatalf("remaining tasks = %d, want 1", len(claimer.tasks))
	}
}

func TestSchedulerNotificationWakesImmediately(t *testing.T) {
	claimer := &schedulerTestClaimer{calls: make(chan struct{}, 4)}
	worker := newSchedulerTestWorker(1)
	scheduler, err := NewScheduler(claimer, worker, SchedulerConfig{
		PollInterval:  time.Hour,
		ErrorBackoff:  time.Second,
		LeaseDuration: time.Minute,
		MaxClaimBatch: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- scheduler.Run(ctx) }()
	select {
	case <-claimer.calls: // initial recovery scan completed
	case <-time.After(time.Second):
		t.Fatal("initial scheduler scan did not run")
	}

	claimer.mu.Lock()
	claimer.tasks = append(claimer.tasks, ClaimedTask{
		Context:    TaskContext{EventId: "event-new", ConsumerName: "card"},
		ClaimToken: "token-new",
	})
	claimer.mu.Unlock()
	started := time.Now()
	scheduler.Notify()
	select {
	case <-worker.notify:
		if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
			t.Fatalf("notification wake took %v", elapsed)
		}
	case <-time.After(time.Second):
		t.Fatal("scheduler did not wake after notification")
	}

	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("scheduler exit: %v", err)
	}
}
