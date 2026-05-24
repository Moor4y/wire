package store

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestSQLiteExecutionStore_CreateAndGetLogs(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "engine.db")
	ctx := context.Background()

	s, err := NewSQLiteExecutionStore(ctx, dbPath)
	if err != nil {
		if strings.Contains(err.Error(), "requires cgo") {
			t.Skipf("sqlite unavailable in this environment: %v", err)
		}
		t.Fatalf("NewSQLiteExecutionStore() error = %v", err)
	}
	defer s.Close()

	if got := s.db.Stats().MaxOpenConnections; got != 1 {
		t.Fatalf("expected MaxOpenConnections=1, got %d", got)
	}

	if err := s.CreateExecution(ctx, "exec-1", "wf-1"); err != nil {
		t.Fatalf("CreateExecution() error = %v", err)
	}
	if err := s.UpdateStepStatus(ctx, "exec-1", "step-1", "running", "start"); err != nil {
		t.Fatalf("UpdateStepStatus(running) error = %v", err)
	}
	if err := s.UpdateStepStatus(ctx, "exec-1", "step-1", "completed", "done"); err != nil {
		t.Fatalf("UpdateStepStatus(completed) error = %v", err)
	}

	logs, err := s.GetExecutionLogs(ctx, "exec-1")
	if err != nil {
		t.Fatalf("GetExecutionLogs() error = %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(logs))
	}
	if logs[0].Status != "running" || logs[1].Status != "completed" {
		t.Fatalf("unexpected status sequence: %#v", logs)
	}
}
