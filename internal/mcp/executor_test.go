package mcpbridge

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"headless-orchestrator/internal/domain"
)

func TestMCPExecutor_ExecuteWrapsFailure(t *testing.T) {
	t.Parallel()

	// Invalid executable should fail quickly during stdio client startup.
	executor, err := NewMCPExecutor("command-that-does-not-exist", nil, nil, "test-client", "1.0.0", slog.Default())
	if err != nil {
		t.Fatalf("NewMCPExecutor() error = %v", err)
	}

	_, err = executor.Execute(context.Background(), domain.Step{
		ID:   "step-1",
		Type: "tool",
		With: map[string]interface{}{},
	}, map[string]interface{}{})
	if err == nil {
		t.Fatal("expected MCP execution failure, got nil")
	}

	var wrapped ErrMCPExecutionFailed
	if !errors.As(err, &wrapped) {
		t.Fatalf("expected ErrMCPExecutionFailed, got %T: %v", err, err)
	}
}
