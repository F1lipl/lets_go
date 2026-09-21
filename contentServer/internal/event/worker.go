package event

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type Worker struct {
	count          int
	queue          chan ClaimedTask
	repo           TaskRepository
	handler        TaskHandler
	timeout        time.Duration
	cleanupTimeout time.Duration
	retryDelay     time.Duration
	mu             sync.Mutex
	started        bool
	running        bool
	stopped        chan struct{}
	ready          chan struct{}
	spaceAvailable chan struct{}
}

type WorkerQueueStats struct {
	Queued    int
	Capacity  int
	Available int
}

func NewWorker(count int, repo TaskRepository, handler TaskHandler) (*Worker, error) {
	if count <= 0 || repo == nil || handler == nil {
		return nil, errors.New("positive worker count, repository and handler are required")
	}
	return &Worker{count: count, queue: make(chan ClaimedTask, count),
		repo: repo, handler: handler, timeout: 10 * time.Second,
		cleanupTimeout: 3 * time.Second, retryDelay: time.Second, stopped: make(chan struct{}), ready: make(chan struct{}),
		spaceAvailable: make(chan struct{}, 1)}, nil
}

// Run blocks until cancellation and waits for all workers. Call once.
// Handlers must respect ctx and only perform short database work on the supplied tx.
func (w *Worker) Run(ctx context.Context) error {
	w.mu.Lock()
	if w.started {
		w.mu.Unlock()
		return ErrAlreadyRunning
	}
	w.started, w.running = true, true
	close(w.ready)
	w.mu.Unlock()
	defer func() { w.mu.Lock(); w.running = false; close(w.stopped); w.mu.Unlock() }()
	var wg sync.WaitGroup
	for i := 0; i < w.count; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); w.work(ctx) }()
	}
	wg.Wait()
	return ctx.Err()
}

// Ready is closed once Run has started. Done closes after all workers exit.
func (w *Worker) Ready() <-chan struct{} { return w.ready }
func (w *Worker) Done() <-chan struct{}  { return w.stopped }
func (w *Worker) SpaceAvailable() <-chan struct{} {
	return w.spaceAvailable
}

// QueueStats is an instantaneous observation intended for scheduling and
// metrics. Scheduler must remain the only producer when it relies on Available.
func (w *Worker) QueueStats() WorkerQueueStats {
	queued := len(w.queue)
	capacity := cap(w.queue)
	return WorkerQueueStats{
		Queued:    queued,
		Capacity:  capacity,
		Available: capacity - queued,
	}
}

// Submit applies backpressure. It does not claim tasks or start transactions.
// On rejection the scheduler must release the claim or let it expire.
func (w *Worker) Submit(ctx context.Context, task ClaimedTask) error {
	if task.Context.EventId == "" || task.Context.ConsumerName == "" || task.ClaimToken == "" {
		return errors.New("task key and claim token are required")
	}
	w.mu.Lock()
	running := w.running
	w.mu.Unlock()
	if !running {
		return ErrWorkerStopped
	}
	task.Context.Payload = append([]byte(nil), task.Context.Payload...)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-w.stopped:
		return ErrWorkerStopped
	case w.queue <- task:
		return nil
	}
}

func (w *Worker) work(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case task := <-w.queue:
			w.notifySpaceAvailable()
			if ctx.Err() != nil {
				return
			} // queued claims are recovered by lease expiry
			w.execute(ctx, task)
		}
	}
}

func (w *Worker) notifySpaceAvailable() {
	select {
	case w.spaceAvailable <- struct{}{}:
	default:
	}
}

func (w *Worker) execute(parent context.Context, task ClaimedTask) {
	ctx, cancel := context.WithTimeout(parent, w.timeout)
	err := w.executeTransaction(ctx, task)
	cancel()
	if err == nil || errors.Is(err, ErrClaimLost) {
		return
	}
	logx.WithContext(parent).Errorf("task failed event=%s consumer=%s: %v", task.Context.EventId, task.Context.ConsumerName, err)
	cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(parent), w.cleanupTimeout)
	defer cleanupCancel()
	if retryErr := w.repo.ScheduleRetry(cleanupCtx, task, err, w.retryDelay, IsPermanent(err)); retryErr != nil && !errors.Is(retryErr, ErrClaimLost) {
		logx.WithContext(cleanupCtx).Errorf("task retry update failed event=%s consumer=%s; awaiting lease recovery: %v", task.Context.EventId, task.Context.ConsumerName, retryErr)
	}
}
func (w *Worker) executeTransaction(ctx context.Context, task ClaimedTask) (err error) {
	// Protect the pool from repository panics as well; transaction cleanup belongs to the repository.
	defer func() {
		if v := recover(); v != nil {
			err = Permanent(fmt.Errorf("task execution panic: %v", v))
		}
	}()
	return w.repo.Execute(ctx, task, func(ctx context.Context, tx sqlx.Session, taskContext *TaskContext) (err error) {
		// Recover inside the callback so the repository sees an error and rolls back.
		defer func() {
			if v := recover(); v != nil {
				err = Permanent(fmt.Errorf("task handler panic: %v", v))
			}
		}()
		return w.handler.Handle(ctx, tx, taskContext)
	})
}
