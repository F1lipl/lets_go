package event

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type ConsumerResolver interface {
	Consumers(eventType string) ([]string, bool)
}

type ConsumerRegistry struct {
	byEventType map[string][]string
}

func NewConsumerRegistry(subscriptions map[string][]string) (*ConsumerRegistry, error) {
	if len(subscriptions) == 0 {
		return nil, errors.New("at least one event subscription is required")
	}
	registry := &ConsumerRegistry{byEventType: make(map[string][]string, len(subscriptions))}
	for eventType, consumers := range subscriptions {
		eventType = strings.TrimSpace(eventType)
		if eventType == "" || len(eventType) > 64 {
			return nil, errors.New("event type must contain 1 to 64 characters")
		}
		if _, exists := registry.byEventType[eventType]; exists {
			return nil, fmt.Errorf("duplicate normalized event type %s", eventType)
		}
		if len(consumers) == 0 {
			return nil, fmt.Errorf("event type %s has no consumers", eventType)
		}
		seen := make(map[string]struct{}, len(consumers))
		copyOfConsumers := make([]string, 0, len(consumers))
		for _, consumer := range consumers {
			consumer = strings.TrimSpace(consumer)
			if consumer == "" || len(consumer) > 64 {
				return nil, fmt.Errorf("event type %s contains an invalid consumer", eventType)
			}
			if _, exists := seen[consumer]; exists {
				return nil, fmt.Errorf("event type %s contains duplicate consumer %s", eventType, consumer)
			}
			seen[consumer] = struct{}{}
			copyOfConsumers = append(copyOfConsumers, consumer)
		}
		registry.byEventType[eventType] = copyOfConsumers
	}
	return registry, nil
}

func (r *ConsumerRegistry) Consumers(eventType string) ([]string, bool) {
	consumers, ok := r.byEventType[eventType]
	if !ok {
		return nil, false
	}
	return append([]string(nil), consumers...), true
}

type DispatchResult struct {
	ScannedEvents    int
	DispatchedEvents int
	FailedEvents     int
	CreatedTasks     int
}

type DispatchRepository interface {
	FanOut(context.Context, int, ConsumerResolver) (DispatchResult, error)
}

type TaskNotifier interface {
	Notify()
}

type DispatcherConfig struct {
	PollInterval time.Duration
	ErrorBackoff time.Duration
	BatchSize    int
}

func DefaultDispatcherConfig() DispatcherConfig {
	return DispatcherConfig{
		PollInterval: 2 * time.Second,
		ErrorBackoff: time.Second,
		BatchSize:    50,
	}
}

type Dispatcher struct {
	repo     DispatchRepository
	registry ConsumerResolver
	notifier TaskNotifier
	config   DispatcherConfig
	wake     chan struct{}
}

func NewDispatcher(repo DispatchRepository, registry ConsumerResolver, notifier TaskNotifier, config DispatcherConfig) (*Dispatcher, error) {
	if repo == nil || registry == nil || notifier == nil {
		return nil, errors.New("dispatch repository, consumer registry and task notifier are required")
	}
	if config.PollInterval < 0 || config.ErrorBackoff < 0 || config.BatchSize < 0 {
		return nil, errors.New("dispatcher configuration cannot be negative")
	}
	defaults := DefaultDispatcherConfig()
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
		return nil, errors.New("dispatcher batch size cannot exceed 500")
	}
	return &Dispatcher{
		repo:     repo,
		registry: registry,
		notifier: notifier,
		config:   config,
		wake:     make(chan struct{}, 1),
	}, nil
}

// Notify must be called after the transaction that inserted an outbox event
// commits. Signals are coalesced because MySQL remains the source of truth.
func (d *Dispatcher) Notify() {
	select {
	case d.wake <- struct{}{}:
	default:
	}
}

func (d *Dispatcher) Run(ctx context.Context) error {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-d.wake:
		case <-timer.C:
		}

		result, err := d.repo.FanOut(ctx, d.config.BatchSize, d.registry)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			logx.WithContext(ctx).Errorf("event dispatch failed: %v", err)
			resetTimer(timer, d.config.ErrorBackoff)
			continue
		}
		if result.FailedEvents > 0 {
			logx.WithContext(ctx).Errorf("event dispatch rejected events=%d because no consumer was registered", result.FailedEvents)
		}
		if result.CreatedTasks > 0 {
			d.notifier.Notify()
		}
		if result.ScannedEvents == d.config.BatchSize {
			resetTimer(timer, 0)
		} else {
			resetTimer(timer, d.config.PollInterval)
		}
	}
}
