package event

import (
	"context"
	"errors"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

var ErrPipelineAlreadyRunning = errors.New("event pipeline can only run once")

type TaskStore interface {
	TaskRepository
	TaskClaimer
	TaskRecoverer
}

type EventPipelineConfig struct {
	WorkerCount        int
	WorkerDrainTimeout time.Duration
	Scheduler          SchedulerConfig
	Dispatcher         DispatcherConfig
	Reclaimer          ReclaimerConfig
}

func DefaultEventPipelineConfig() EventPipelineConfig {
	return EventPipelineConfig{
		WorkerCount:        4,
		WorkerDrainTimeout: 25 * time.Second,
		Scheduler:          DefaultSchedulerConfig(),
		Dispatcher:         DefaultDispatcherConfig(),
		Reclaimer:          DefaultReclaimerConfig(),
	}
}

// EventPipeline owns the lifecycle of the local outbox processing stages. Its
// fields remain private so callers can only run the complete pipeline and wake
// the dispatcher after an outbox-producing transaction commits.
type EventPipeline struct {
	dispatcher *Dispatcher
	scheduler  *Scheduler
	reclaimer  *Reclaimer
	worker     *Worker

	mu      sync.Mutex
	started bool
	ready   chan struct{}
	done    chan struct{}
}

func NewEventPipeline(
	dispatchRepo DispatchRepository,
	taskStore TaskStore,
	registry ConsumerResolver,
	handler TaskHandler,
	config EventPipelineConfig,
) (*EventPipeline, error) {
	if dispatchRepo == nil || taskStore == nil || registry == nil || handler == nil {
		return nil, errors.New("dispatch repository, task store, consumer registry and task handler are required")
	}
	if config.WorkerCount < 0 || config.WorkerDrainTimeout < 0 {
		return nil, errors.New("worker configuration cannot be negative")
	}
	defaults := DefaultEventPipelineConfig()
	if config.WorkerCount == 0 {
		config.WorkerCount = defaults.WorkerCount
	}
	if config.WorkerDrainTimeout == 0 {
		config.WorkerDrainTimeout = defaults.WorkerDrainTimeout
	}

	worker, err := NewWorker(config.WorkerCount, taskStore, handler)
	if err != nil {
		return nil, err
	}
	worker.shutdownGrace = config.WorkerDrainTimeout
	scheduler, err := NewScheduler(taskStore, worker, config.Scheduler)
	if err != nil {
		return nil, err
	}
	dispatcher, err := NewDispatcher(dispatchRepo, registry, scheduler, config.Dispatcher)
	if err != nil {
		return nil, err
	}
	reclaimer, err := NewReclaimer(taskStore, scheduler, config.Reclaimer)
	if err != nil {
		return nil, err
	}
	return &EventPipeline{
		dispatcher: dispatcher,
		scheduler:  scheduler,
		reclaimer:  reclaimer,
		worker:     worker,
		ready:      make(chan struct{}),
		done:       make(chan struct{}),
	}, nil
}

// NotifyOutboxCommitted is non-blocking and must only be called after the
// transaction that inserted the outbox row has committed.
func (p *EventPipeline) NotifyOutboxCommitted() {
	p.dispatcher.Notify()
}

func (p *EventPipeline) Ready() <-chan struct{} { return p.ready }
func (p *EventPipeline) Done() <-chan struct{}  { return p.done }

func (p *EventPipeline) Run(ctx context.Context) error {
	p.mu.Lock()
	if p.started {
		p.mu.Unlock()
		return ErrPipelineAlreadyRunning
	}
	p.started = true
	p.mu.Unlock()
	defer close(p.done)

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		return p.worker.Run(groupCtx)
	})

	select {
	case <-p.worker.Ready():
	case <-groupCtx.Done():
		return group.Wait()
	}

	group.Go(func() error {
		return p.scheduler.Run(groupCtx)
	})
	group.Go(func() error {
		return p.dispatcher.Run(groupCtx)
	})
	group.Go(func() error {
		return p.reclaimer.Run(groupCtx)
	})
	close(p.ready)

	return group.Wait()
}
