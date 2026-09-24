package event

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var (
	ErrClaimLost             = errors.New("task claim lost")
	ErrWorkerStopped         = errors.New("worker is not running")
	ErrAlreadyRunning        = errors.New("worker can only run once")
	ErrWorkerShutdownTimeout = errors.New("worker shutdown timed out")
	ErrSuperseded            = errors.New("task superseded")
)

type permanentError struct{ error }

func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &permanentError{err}
}
func (e *permanentError) Unwrap() error { return e.error }
func IsPermanent(err error) bool        { var target *permanentError; return errors.As(err, &target) }

type Event struct {
	EventId       string
	EventType     string
	SchemaVersion uint64
	OccurredAt    time.Time
	Payload       json.RawMessage
}
type TaskContext struct {
	EventId       string
	ConsumerName  string
	EventType     string
	SchemaVersion uint64
	Payload       json.RawMessage
}
type Taskcontext = TaskContext

// ClaimedTask is created only after the scheduler commits its claim transaction.
type ClaimedTask struct {
	Context     TaskContext
	ClaimToken  string
	LockedUntil time.Time
}
type TaskHandler interface {
	Handle(context.Context, sqlx.Session, *TaskContext) error
}
type TaskHandlerFunc func(context.Context, sqlx.Session, *TaskContext) error

func (f TaskHandlerFunc) Handle(ctx context.Context, tx sqlx.Session, task *TaskContext) error {
	return f(ctx, tx, task)
}

type TaskRepository interface {
	Execute(context.Context, ClaimedTask, func(context.Context, sqlx.Session, *TaskContext) error) error
	ScheduleRetry(context.Context, ClaimedTask, error, time.Duration, bool) error
}

// TaskClaimer atomically transfers ready deliveries to the processing state.
// Claim must commit before returning tasks to the scheduler.
type TaskClaimer interface {
	Claim(context.Context, int, time.Duration) ([]ClaimedTask, error)
}

// RecoveryResult describes one committed recovery batch. A recovered task is
// ready to be claimed again; a failed task exhausted its configured attempts.
type RecoveryResult struct {
	Scanned   int
	Recovered int
	Failed    int
}

// TaskRecoverer transfers deliveries whose processing lease expired back to
// pending, or to failed when their attempt budget has been exhausted.
type TaskRecoverer interface {
	RecoverExpired(context.Context, int) (RecoveryResult, error)
}

type TaskController struct {
	taskMap map[string]TaskHandler
}

func NewTaskController() *TaskController {
	return &TaskController{taskMap: make(map[string]TaskHandler)}
}

// Register must only be called during service initialization, before the
// controller is handed to a running Worker. After startup taskMap is immutable,
// so Handle can safely perform lock-free concurrent reads.
func (c *TaskController) Register(name string, handler TaskHandler) error {
	if name == "" || handler == nil {
		return errors.New("consumer and handler are required")
	}
	if _, exists := c.taskMap[name]; exists {
		return fmt.Errorf("handler already registered: %s", name)
	}
	c.taskMap[name] = handler
	return nil
}
func (c *TaskController) Handle(ctx context.Context, tx sqlx.Session, task *TaskContext) error {
	if task == nil {
		return Permanent(errors.New("nil task"))
	}
	handler, ok := c.taskMap[task.ConsumerName]
	if !ok {
		return fmt.Errorf("handler not registered: %s", task.ConsumerName)
	}
	return handler.Handle(ctx, tx, task)
}
