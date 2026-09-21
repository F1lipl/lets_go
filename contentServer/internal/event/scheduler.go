package event

import (
	"context"
	"errors"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type TaskSubmitter interface {
	Submit(context.Context, ClaimedTask) error
	QueueStats() WorkerQueueStats
	Ready() <-chan struct{}
	Done() <-chan struct{}
	SpaceAvailable() <-chan struct{}
}

type SchedulerConfig struct {
	PollInterval  time.Duration
	ErrorBackoff  time.Duration
	LeaseDuration time.Duration
	MaxClaimBatch int
}

func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		PollInterval:  2 * time.Second,
		ErrorBackoff:  time.Second,
		LeaseDuration: 30 * time.Second,
		MaxClaimBatch: 32,
	}
}

type Scheduler struct {
	claimer TaskClaimer
	worker  TaskSubmitter
	config  SchedulerConfig
	wake    chan struct{}
}

func NewScheduler(claimer TaskClaimer, worker TaskSubmitter, config SchedulerConfig) (*Scheduler, error) {
	if claimer == nil || worker == nil {
		return nil, errors.New("task claimer and worker are required")
	}
	if config.PollInterval < 0 || config.ErrorBackoff < 0 || config.LeaseDuration < 0 || config.MaxClaimBatch < 0 {
		return nil, errors.New("scheduler configuration cannot be negative")
	}
	defaults := DefaultSchedulerConfig()
	if config.PollInterval == 0 {
		config.PollInterval = defaults.PollInterval
	}
	if config.ErrorBackoff == 0 {
		config.ErrorBackoff = defaults.ErrorBackoff
	}
	if config.LeaseDuration == 0 {
		config.LeaseDuration = defaults.LeaseDuration
	}
	if config.MaxClaimBatch == 0 {
		config.MaxClaimBatch = defaults.MaxClaimBatch
	}
	return &Scheduler{claimer: claimer, worker: worker, config: config, wake: make(chan struct{}, 1)}, nil
}

// Notify wakes the scheduler after a dispatcher commits new delivery rows.
// It never blocks the dispatcher; repeated notifications are intentionally
// coalesced because the database, rather than the signal, is the source of work.
func (s *Scheduler) Notify() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// Run waits for worker startup, claims only as many tasks as the worker queue
// can currently accept, and submits them after the claim transaction commits.
func (s *Scheduler) Run(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.worker.Done():
		return ErrWorkerStopped
	case <-s.worker.Ready():
	}

	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.worker.Done():
			return ErrWorkerStopped
		case <-s.wake:
		case <-s.worker.SpaceAvailable():
		case <-timer.C:
		}

		available := s.worker.QueueStats().Available
		if available <= 0 {
			resetTimer(timer, s.config.PollInterval)
			continue
		}
		if available > s.config.MaxClaimBatch {
			available = s.config.MaxClaimBatch
		}

		tasks, err := s.claimer.Claim(ctx, available, s.config.LeaseDuration)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			logx.WithContext(ctx).Errorf("task claim failed: %v", err)
			resetTimer(timer, s.config.ErrorBackoff)
			continue
		}

		for _, task := range tasks {
			if err := s.worker.Submit(ctx, task); err != nil {
				// The committed claim is intentionally left in processing. If the
				// process is stopping, lease recovery will make it available again.
				if ctx.Err() != nil {
					return ctx.Err()
				}
				return err
			}
		}

		if len(tasks) == available {
			// The database may have more ready work. Recheck immediately; queue
			// capacity still bounds the next claim.
			resetTimer(timer, 0)
		} else {
			resetTimer(timer, s.config.PollInterval)
		}
	}
}

func resetTimer(timer *time.Timer, delay time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(delay)
}
