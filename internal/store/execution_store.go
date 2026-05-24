package store

import "context"

// ExecutionLog captures persisted step execution status events.
type ExecutionLog struct {
	ExecutionID string
	StepID      string
	Status      string
	Message     string
	CreatedAt   string
}

// ExecutionStore defines persistence operations for workflow executions.
type ExecutionStore interface {
	CreateExecution(ctx context.Context, executionID string, workflowID string) error
	UpdateStepStatus(ctx context.Context, executionID string, stepID string, status string, message string) error
	GetExecutionLogs(ctx context.Context, executionID string) ([]ExecutionLog, error)
}
