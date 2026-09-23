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
	shutdownGrace  time.Duration
	cleanupTimeout time.Duration
	retryDelay     time.Duration
	mu             sync.Mutex
	started        bool
	running        bool
	accepting      bool
	stopped        chan struct{}
	ready          chan struct{}
	draining       chan struct{}
	drainOnce      sync.Once
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
		shutdownGrace: 25 * time.Second, cleanupTimeout: 3 * time.Second, retryDelay: time.Second,
		stopped: make(chan struct{}), ready: make(chan struct{}), draining: make(chan struct{}),
		spaceAvailable: make(chan struct{}, 1)}, nil
}

// Run blocks until cancellation. Cancellation first stops new submissions and
// lets workers drain already claimed tasks. Only after shutdownGrace expires is
// the execution context canceled. Handlers must respect ctx and only perform
// short database work on the supplied transaction.
func (w *Worker) Run(ctx context.Context) error {
	w.mu.Lock()
	if w.started {
		w.mu.Unlock()
		return ErrAlreadyRunning
	}
	w.started, w.running, w.accepting = true, true, true
	close(w.ready)
	w.mu.Unlock()
	defer func() {
		// Close first so a submitter currently holding mu while waiting on a full
		// queue can unblock instead of deadlocking shutdown.
		close(w.stopped)
		w.mu.Lock()
		w.running, w.accepting = false, false
		w.mu.Unlock()
	}()

	executionCtx, forceStop := context.WithCancel(context.WithoutCancel(ctx))
	defer forceStop()
	var wg sync.WaitGroup
	for i := 0; i < w.count; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); w.work(executionCtx) }()
	}
	workersDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(workersDone)
	}()

	<-ctx.Done()
	w.stopAcceptingAndDrain()
	timer := time.NewTimer(w.shutdownGrace)
	defer timer.Stop()
	select {
	case <-workersDone:
	case <-timer.C:
		logx.WithContext(ctx).Errorf("worker graceful shutdown timed out after %s; canceling active handlers", w.shutdownGrace)
		forceStop()
		<-workersDone
	}
	return ctx.Err()
}

func (w *Worker) stopAcceptingAndDrain() {
	w.mu.Lock()
	w.accepting = false
	w.drainOnce.Do(func() { close(w.draining) })
	w.mu.Unlock()
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
	defer w.mu.Unlock()
	if !w.running || !w.accepting {
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
		case <-w.draining:
			w.drain(ctx)
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

func (w *Worker) drain(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case task := <-w.queue:
			w.notifySpaceAvailable()
			w.execute(ctx, task)
		default:
			return
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
