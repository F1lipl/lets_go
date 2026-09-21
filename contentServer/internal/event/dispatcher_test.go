package event

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type dispatcherTestRepository struct {
	mu      sync.Mutex
	results []DispatchResult
	calls   chan struct{}
}

func (r *dispatcherTestRepository) FanOut(_ context.Context, _ int, _ ConsumerResolver) (DispatchResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	select {
	case r.calls <- struct{}{}:
	default:
	}
	if len(r.results) == 0 {
		return DispatchResult{}, nil
	}
	result := r.results[0]
	r.results = r.results[1:]
	return result, nil
}

type dispatcherTestNotifier struct {
	notified chan struct{}
}

func (n *dispatcherTestNotifier) Notify() {
	select {
	case n.notified <- struct{}{}:
	default:
	}
}

func TestConsumerRegistryIsImmutable(t *testing.T) {
	source := map[string][]string{
		"PostPublished": {"post_card", "post_search"},
	}
	registry, err := NewConsumerRegistry(source)
	if err != nil {
		t.Fatal(err)
	}
	source["PostPublished"][0] = "changed"
	consumers, ok := registry.Consumers("PostPublished")
	if !ok || len(consumers) != 2 || consumers[0] != "post_card" {
		t.Fatalf("unexpected consumers: %v", consumers)
	}
	consumers[0] = "changed-again"
	again, _ := registry.Consumers("PostPublished")
	if again[0] != "post_card" {
		t.Fatal("caller mutated registry contents")
	}
	if _, err := NewConsumerRegistry(map[string][]string{"PostPublished": {"post_card", "post_card"}}); err == nil {
		t.Fatal("duplicate consumer was accepted")
	}
}

func TestDispatcherNotificationTriggersFanOutAndScheduler(t *testing.T) {
	repo := &dispatcherTestRepository{calls: make(chan struct{}, 4)}
	registry, err := NewConsumerRegistry(map[string][]string{"PostPublished": {"post_card"}})
	if err != nil {
		t.Fatal(err)
	}
	notifier := &dispatcherTestNotifier{notified: make(chan struct{}, 1)}
	dispatcher, err := NewDispatcher(repo, registry, notifier, DispatcherConfig{
		PollInterval: time.Hour,
		ErrorBackoff: time.Second,
		BatchSize:    10,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- dispatcher.Run(ctx) }()
	select {
	case <-repo.calls:
	case <-time.After(time.Second):
		t.Fatal("initial dispatch scan did not run")
	}

	repo.mu.Lock()
	repo.results = append(repo.results, DispatchResult{
		ScannedEvents:    1,
		DispatchedEvents: 1,
		CreatedTasks:     2,
	})
	repo.mu.Unlock()
	started := time.Now()
	dispatcher.Notify()
	select {
	case <-notifier.notified:
		if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
			t.Fatalf("dispatch notification took %v", elapsed)
		}
	case <-time.After(time.Second):
		t.Fatal("dispatcher did not notify scheduler")
	}

	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("dispatcher exit: %v", err)
	}
}
