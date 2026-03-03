package middleware

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/vigo999/ms-cli/agent/loop"
	"github.com/vigo999/ms-cli/ui/model"
)

// Runner is the interface for task runners.
type Runner interface {
	Execute(task loop.Task, eventCh chan<- model.Event) error
}

// Middleware is a function that wraps a Runner.
type Middleware func(Runner) Runner

// Chain chains multiple middleware together.
func Chain(middlewares ...Middleware) Middleware {
	return func(final Runner) Runner {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

// LoggingMiddleware logs all task executions.
func LoggingMiddleware(logger *log.Logger) Middleware {
	return func(next Runner) Runner {
		return &loggingRunner{next: next, logger: logger}
	}
}

type loggingRunner struct {
	next   Runner
	logger *log.Logger
}

func (r *loggingRunner) Execute(task loop.Task, eventCh chan<- model.Event) error {
	start := time.Now()
	r.logger.Printf("[EXECUTE] task=%q", task.Description)
	
	err := r.next.Execute(task, eventCh)
	
	duration := time.Since(start)
	if err != nil {
		r.logger.Printf("[ERROR] task=%q duration=%v error=%v", task.Description, duration, err)
	} else {
		r.logger.Printf("[COMPLETE] task=%q duration=%v", task.Description, duration)
	}
	return err
}

// RecoveryMiddleware recovers from panics.
func RecoveryMiddleware() Middleware {
	return func(next Runner) Runner {
		return &recoveryRunner{next: next}
	}
}

type recoveryRunner struct {
	next Runner
}

func (r *recoveryRunner) Execute(task loop.Task, eventCh chan<- model.Event) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("panic recovered: %v", rec)
			eventCh <- model.Event{
				Type:     model.ToolError,
				ToolName: "Runner",
				Message:  err.Error(),
			}
		}
	}()
	return r.next.Execute(task, eventCh)
}

// TimeoutMiddleware adds timeout to task execution.
func TimeoutMiddleware(timeout time.Duration) Middleware {
	return func(next Runner) Runner {
		return &timeoutRunner{next: next, timeout: timeout}
	}
}

type timeoutRunner struct {
	next    Runner
	timeout time.Duration
}

func (r *timeoutRunner) Execute(task loop.Task, eventCh chan<- model.Event) error {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	
	done := make(chan error, 1)
	go func() {
		done <- r.next.Execute(task, eventCh)
	}()
	
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("task execution timed out after %v", r.timeout)
	}
}

// MetricsMiddleware collects execution metrics.
func MetricsMiddleware() Middleware {
	return func(next Runner) Runner {
		return &metricsRunner{next: next}
	}
}

type metricsRunner struct {
	next     Runner
	taskCount int64
	totalTime time.Duration
}

func (r *metricsRunner) Execute(task loop.Task, eventCh chan<- model.Event) error {
	start := time.Now()
	err := r.next.Execute(task, eventCh)
	r.taskCount++
	r.totalTime += time.Since(start)
	return err
}

// GetStats returns execution statistics.
func (r *metricsRunner) GetStats() (count int64, avgTime time.Duration) {
	if r.taskCount == 0 {
		return 0, 0
	}
	return r.taskCount, r.totalTime / time.Duration(r.taskCount)
}
