package event

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"runtime"
	"testing"
	"time"
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
