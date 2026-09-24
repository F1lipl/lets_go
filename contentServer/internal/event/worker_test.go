package event

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type testRepo struct{ results chan string }

func (r *testRepo) Execute(ctx context.Context, task ClaimedTask, fn func(context.Context, sqlx.Session, *TaskContext) error) error {
	if task.Context.EventId == "lost" {
		r.results <- "lost"
		return ErrClaimLost
	}
	err := fn(ctx, nil, &task.Context)
	if err == nil {
		r.results <- "success"
	}
	return err
}
func (r *testRepo) ScheduleRetry(ctx context.Context, task ClaimedTask, cause error, _ time.Duration, permanent bool) error {
	if ctx.Err() != nil {
		return errors.New("cleanup context already canceled")
	}
	if permanent {
		r.results <- "permanent"
	} else {
		r.results <- "retry"
	}
	return nil
}
func TestWorkerContinuesAfterErrorsAndPanic(t *testing.T) {
	repo := &testRepo{results: make(chan string, 10)}
	handler := TaskHandlerFunc(func(ctx context.Context, _ sqlx.Session, task *TaskContext) error {
		switch task.EventId {
		case "error":
			return errors.New("temporary failure")
		case "panic":
			panic("handler bug")
		case "timeout":
			<-ctx.Done()
			return ctx.Err()
		default:
			return nil
		}
	})
	worker, err := NewWorker(1, repo, handler)
	if err != nil {
		t.Fatal(err)
	}
	worker.timeout = 20 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()
	deadline := time.Now().Add(time.Second)
	for {
		worker.mu.Lock()
		running := worker.running
		worker.mu.Unlock()
		if running {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("worker did not start")
		}
		runtime.Gosched()
	}
	for _, id := range []string{"error", "panic", "timeout", "lost", "ok"} {
		submitCtx, stop := context.WithTimeout(ctx, time.Second)
		err := worker.Submit(submitCtx, ClaimedTask{Context: TaskContext{EventId: id, ConsumerName: "test"}, ClaimToken: "token"})
		stop()
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, want := range []string{"retry", "permanent", "retry", "lost", "success"} {
		select {
		case got := <-repo.results:
			if got != want {
				t.Fatalf("got %s, want %s", got, want)
			}
		case <-time.After(time.Second):
			t.Fatal("worker stopped processing")
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not exit")
	}
	if err := worker.Run(ctx); !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("second run: %v", err)
	}
	if err := worker.Submit(context.Background(), ClaimedTask{Context: TaskContext{EventId: "x", ConsumerName: "test"}, ClaimToken: "x"}); !errors.Is(err, ErrWorkerStopped) {
		t.Fatalf("submit after stop: %v", err)
	}
}

func TestControllerRegistration(t *testing.T) {
	c := NewTaskController()
	if err := c.Handle(context.Background(), nil, &TaskContext{ConsumerName: "missing"}); err == nil {
		t.Fatal("missing handler succeeded")
	}
	h := TaskHandlerFunc(func(context.Context, sqlx.Session, *TaskContext) error { return nil })
	if err := c.Register("test", h); err != nil {
		t.Fatal(err)
	}
	if err := c.Register("test", h); err == nil {
		t.Fatal("duplicate handler accepted")
	}
}

func TestWorkerDrainsClaimedTasksBeforeExit(t *testing.T) {
	repo := &testRepo{results: make(chan string, 4)}
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var startedOnce sync.Once
	handler := TaskHandlerFunc(func(ctx context.Context, _ sqlx.Session, task *TaskContext) error {
		if task.EventId != "first" {
			return nil
		}
		startedOnce.Do(func() { close(firstStarted) })
		select {
		case <-releaseFirst:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	worker, err := NewWorker(1, repo, handler)
	if err != nil {
		t.Fatal(err)
	}
	worker.shutdownGrace = time.Second
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()
	<-worker.Ready()

	for _, id := range []string{"first", "second"} {
		if err := worker.Submit(ctx, ClaimedTask{Context: TaskContext{EventId: id, ConsumerName: "test"}, ClaimToken: id}); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first task did not start")
	}

	cancel()
	select {
	case <-worker.draining:
	case <-time.After(time.Second):
		t.Fatal("worker did not enter draining state")
	}
	if err := worker.Submit(context.Background(), ClaimedTask{Context: TaskContext{EventId: "late", ConsumerName: "test"}, ClaimToken: "late"}); !errors.Is(err, ErrWorkerStopped) {
		t.Fatalf("submit during drain: %v", err)
	}
	// Cancellation starts draining but must not cancel the active handler while
	// the grace period is still available.
	select {
	case result := <-repo.results:
		t.Fatalf("active task finished before release: %s", result)
	case <-time.After(20 * time.Millisecond):
	}
	close(releaseFirst)
	for i := 0; i < 2; i++ {
		select {
		case result := <-repo.results:
			if result != "success" {
				t.Fatalf("drained task result = %s", result)
			}
		case <-time.After(time.Second):
			t.Fatal("claimed task was not drained")
		}
	}
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("worker exit: %v", err)
	}
}

func TestWorkerCancelsActiveHandlerAfterGracePeriod(t *testing.T) {
	repo := &testRepo{results: make(chan string, 2)}
	handlerStarted := make(chan struct{})
	handlerCanceled := make(chan struct{})
	handler := TaskHandlerFunc(func(ctx context.Context, _ sqlx.Session, _ *TaskContext) error {
		close(handlerStarted)
		<-ctx.Done()
		close(handlerCanceled)
		return ctx.Err()
	})
	worker, err := NewWorker(1, repo, handler)
	if err != nil {
		t.Fatal(err)
	}
	worker.shutdownGrace = 20 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()
	<-worker.Ready()
	if err := worker.Submit(ctx, ClaimedTask{Context: TaskContext{EventId: "active", ConsumerName: "test"}, ClaimToken: "active"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-handlerStarted:
	case <-time.After(time.Second):
		t.Fatal("handler did not start")
	}
	cancel()
	select {
	case <-handlerCanceled:
	case <-time.After(time.Second):
		t.Fatal("handler was not canceled after shutdown grace period")
	}
	select {
	case result := <-repo.results:
		if result != "retry" {
			t.Fatalf("canceled task result = %s", result)
		}
	case <-time.After(time.Second):
		t.Fatal("canceled task was not scheduled for retry")
	}
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("worker exit: %v", err)
	}
}

func TestWorkerShutdownReleasesSubmitBlockedByFullQueue(t *testing.T) {
	repo := &testRepo{results: make(chan string, 4)}
	handlerStarted := make(chan struct{})
	releaseHandler := make(chan struct{})
	handler := TaskHandlerFunc(func(ctx context.Context, _ sqlx.Session, task *TaskContext) error {
		if task.EventId != "active" {
			return nil
		}
		close(handlerStarted)
		select {
		case <-releaseHandler:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	worker, err := NewWorker(1, repo, handler)
	if err != nil {
		t.Fatal(err)
	}
	worker.shutdownGrace = time.Second
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()
	<-worker.Ready()

	if err := worker.Submit(ctx, ClaimedTask{Context: TaskContext{EventId: "active", ConsumerName: "test"}, ClaimToken: "active"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-handlerStarted:
	case <-time.After(time.Second):
		t.Fatal("active handler did not start")
	}
	if err := worker.Submit(ctx, ClaimedTask{Context: TaskContext{EventId: "queued", ConsumerName: "test"}, ClaimToken: "queued"}); err != nil {
		t.Fatal(err)
	}

	blockedSubmit := make(chan error, 1)
	go func() {
		blockedSubmit <- worker.Submit(context.Background(), ClaimedTask{
			Context:    TaskContext{EventId: "blocked", ConsumerName: "test"},
			ClaimToken: "blocked",
		})
	}()
	select {
	case err := <-blockedSubmit:
		t.Fatalf("submit unexpectedly completed while queue was full: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	cancel()
	select {
	case err := <-blockedSubmit:
		if !errors.Is(err, ErrWorkerStopped) {
			t.Fatalf("blocked submit returned %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown did not release blocked submit")
	}
	close(releaseHandler)
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("worker exit: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("worker did not finish draining")
	}
}

func TestWorkerReturnsWhenHandlerIgnoresForcedCancellation(t *testing.T) {
	repo := &testRepo{results: make(chan string, 2)}
	handlerStarted := make(chan struct{})
	handlerFinished := make(chan struct{})
	releaseHandler := make(chan struct{})
	handler := TaskHandlerFunc(func(context.Context, sqlx.Session, *TaskContext) error {
		close(handlerStarted)
		<-releaseHandler
		close(handlerFinished)
		return nil
	})
	worker, err := NewWorker(1, repo, handler)
	if err != nil {
		t.Fatal(err)
	}
	worker.shutdownGrace = 10 * time.Millisecond
	worker.forceStopGrace = 10 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()
	<-worker.Ready()
	if err := worker.Submit(ctx, ClaimedTask{Context: TaskContext{EventId: "stuck", ConsumerName: "test"}, ClaimToken: "stuck"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-handlerStarted:
	case <-time.After(time.Second):
		t.Fatal("handler did not start")
	}

	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) || !errors.Is(err, ErrWorkerShutdownTimeout) {
			t.Fatalf("worker exit = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("worker Run remained blocked after the forced-stop deadline")
	}
	select {
	case <-worker.Done():
		t.Fatal("Done closed while the handler was still running")
	default:
	}

	close(releaseHandler)
	select {
	case <-handlerFinished:
	case <-time.After(time.Second):
		t.Fatal("handler did not finish after release")
	}
	select {
	case <-worker.Done():
	case <-time.After(time.Second):
		t.Fatal("Done was not closed after the remaining worker exited")
	}
}
