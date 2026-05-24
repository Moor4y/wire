package engine

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"headless-orchestrator/internal/domain"
)

type blockingExecutor struct{}

func (b blockingExecutor) Execute(ctx context.Context, step domain.Step, state map[string]interface{}) (map[string]interface{}, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

type passThroughExecutor struct{}

func (p passThroughExecutor) Execute(ctx context.Context, step domain.Step, state map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		step.ID: step.Type,
	}, nil
}

func TestExecutionEngine_RunCancelled(t *testing.T) {
	t.Parallel()

	e, err := NewExecutionEngine(blockingExecutor{}, nil, slog.Default())
	if err != nil {
		t.Fatalf("NewExecutionEngine() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err = e.Run(ctx, "exec-cancel", "wf-cancel", &domain.Workflow{
		Steps: []domain.Step{{ID: "s1", Type: "tool"}},
	}, nil)
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation error, got %v", err)
	}
}

func TestExecutionEngine_StateMerge(t *testing.T) {
	t.Parallel()

	e, err := NewExecutionEngine(passThroughExecutor{}, nil, slog.Default())
	if err != nil {
		t.Fatalf("NewExecutionEngine() error = %v", err)
	}

	out, err := e.Run(context.Background(), "exec-merge", "wf-merge", &domain.Workflow{
		Steps: []domain.Step{
			{ID: "s1", Type: "alpha"},
			{ID: "s2", Type: "beta"},
		},
	}, map[string]interface{}{"seed": "ok"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if out["seed"] != "ok" || out["s1"] != "alpha" || out["s2"] != "beta" {
		t.Fatalf("unexpected merged state: %#v", out)
	}
}
