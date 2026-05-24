package engine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"headless-orchestrator/internal/domain"
)

// StepExecutor executes a workflow step using a provided immutable snapshot of state.
// It returns key/value pairs that are merged into the global state.
type StepExecutor interface {
	Execute(ctx context.Context, step domain.Step, state map[string]interface{}) (map[string]interface{}, error)
}

// ExecutionStore is the minimal persistence contract used by the engine.
// The SQLite implementation in internal/store satisfies this interface.
type ExecutionStore interface {
	CreateExecution(ctx context.Context, executionID string, workflowID string) error
	UpdateStepStatus(ctx context.Context, executionID string, stepID string, status string, message string) error
}

// ExecutionEngine orchestrates validated workflow execution.
type ExecutionEngine struct {
	executor StepExecutor
	store    ExecutionStore
	logger   *slog.Logger
}

// NewExecutionEngine creates an execution engine with injected interfaces.
func NewExecutionEngine(stepExecutor StepExecutor, store ExecutionStore, logger *slog.Logger) (*ExecutionEngine, error) {
	if stepExecutor == nil {
		return nil, errors.New("step executor is required")
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &ExecutionEngine{
		executor: stepExecutor,
		store:    store,
		logger:   logger,
	}, nil
}

// Run executes all steps sequentially.
//
// Defensive behavior:
//   - Runs execution logic in a goroutine and respects ctx cancellation/timeouts.
//   - Passes a state snapshot to each step to reduce mutation races/state corruption.
//   - Merges step outputs with explicit overwrite semantics.
func (e *ExecutionEngine) Run(
	ctx context.Context,
	executionID string,
	workflowID string,
	workflow *domain.Workflow,
	initialState map[string]interface{},
) (map[string]interface{}, error) {
	if workflow == nil {
		return nil, errors.New("workflow is nil")
	}

	type result struct {
		state map[string]interface{}
		err   error
	}
	resultCh := make(chan result, 1)

	go func() {
		state := cloneMap(initialState)
		if state == nil {
			state = map[string]interface{}{}
		}

		if e.store != nil {
			if err := e.store.CreateExecution(ctx, executionID, workflowID); err != nil {
				resultCh <- result{err: fmt.Errorf("create execution: %w", err)}
				return
			}
		}

		for _, step := range workflow.Steps {
			select {
			case <-ctx.Done():
				resultCh <- result{err: ctx.Err()}
				return
			default:
			}

			if e.store != nil {
				if err := e.store.UpdateStepStatus(ctx, executionID, step.ID, "running", "step started"); err != nil {
					resultCh <- result{err: fmt.Errorf("step %s running status: %w", step.ID, err)}
					return
				}
			}

			output, err := e.executor.Execute(ctx, step, cloneMap(state))
			if err != nil {
				e.logger.Error("step execution failed", "step_id", step.ID, "step_type", step.Type, "error", err)
				if e.store != nil {
					_ = e.store.UpdateStepStatus(ctx, executionID, step.ID, "failed", err.Error())
				}
				resultCh <- result{err: fmt.Errorf("step %s (%s) execution failed: %w", step.ID, step.Type, err)}
				return
			}

			mergeState(state, output)
			if e.store != nil {
				if err := e.store.UpdateStepStatus(ctx, executionID, step.ID, "completed", "step completed"); err != nil {
					resultCh <- result{err: fmt.Errorf("step %s completed status: %w", step.ID, err)}
					return
				}
			}
		}

		resultCh <- result{state: state}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case out := <-resultCh:
		if out.err != nil {
			return nil, out.err
		}
		return out.state, nil
	}
}

func mergeState(global map[string]interface{}, update map[string]interface{}) {
	if global == nil || update == nil {
		return
	}
	for key, value := range update {
		global[key] = value
	}
}

func cloneMap(in map[string]interface{}) map[string]interface{} {
	if in == nil {
		return nil
	}
	out := make(map[string]interface{}, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
