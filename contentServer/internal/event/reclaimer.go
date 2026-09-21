package event

import (
	"context"
	"errors"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReclaimerConfig struct {
	PollInterval time.Duration
	ErrorBackoff time.Duration
	BatchSize    int
}

func DefaultReclaimerConfig() ReclaimerConfig {
	return ReclaimerConfig{
		PollInterval: 5 * time.Second,
		ErrorBackoff: time.Second,
		BatchSize:    50,
	}
}

// Reclaimer periodically returns deliveries abandoned in processing to the
// ready queue. The repository owns the transaction and multi-instance claim
// coordination; this component only controls timing and scheduler wakeups.
type Reclaimer struct {
	repo     TaskRecoverer
	notifier TaskNotifier
	config   ReclaimerConfig
}

func NewReclaimer(repo TaskRecoverer, notifier TaskNotifier, config ReclaimerConfig) (*Reclaimer, error) {
	if repo == nil || notifier == nil {
		return nil, errors.New("task recoverer and notifier are required")
	}
	if config.PollInterval < 0 || config.ErrorBackoff < 0 || config.BatchSize < 0 {
		return nil, errors.New("reclaimer configuration cannot be negative")
	}
	defaults := DefaultReclaimerConfig()
	if config.PollInterval == 0 {
		config.PollInterval = defaults.PollInterval
	}
	if config.ErrorBackoff == 0 {
		config.ErrorBackoff = defaults.ErrorBackoff
	}
	if config.BatchSize == 0 {
		config.BatchSize = defaults.BatchSize
	}
	if config.BatchSize > 500 {
		return nil, errors.New("reclaimer batch size cannot exceed 500")
	}
	return &Reclaimer{repo: repo, notifier: notifier, config: config}, nil
}

// Run performs an immediate startup scan and then uses a timer as the normal
// recovery path. Full batches are drained without waiting for the next tick.
func (r *Reclaimer) Run(ctx context.Context) error {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}

		result, err := r.repo.RecoverExpired(ctx, r.config.BatchSize)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			logx.WithContext(ctx).Errorf("expired task recovery failed: %v", err)
			resetTimer(timer, r.config.ErrorBackoff)
			continue
		}
		if result.Recovered > 0 {
			r.notifier.Notify()
		}
		if result.Failed > 0 {
			logx.WithContext(ctx).Errorf("expired tasks exhausted attempts count=%d", result.Failed)
		}
		if result.Scanned == r.config.BatchSize {
			resetTimer(timer, 0)
		} else {
			resetTimer(timer, r.config.PollInterval)
		}
	}
}
