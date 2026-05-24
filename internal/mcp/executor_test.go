package mcpbridge

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
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

func TestResolveValueWithState_StringInterpolation(t *testing.T) {
	t.Parallel()

	value, err := resolveValueWithState(
		"processed: ${state.message}",
		map[string]interface{}{"message": "hello-world"},
	)
	if err != nil {
		t.Fatalf("resolveValueWithState() error = %v", err)
	}
	if value != "processed: hello-world" {
		t.Fatalf("unexpected interpolated value: %v", value)
	}
}

func TestResolveValueWithState_MissingStateKey(t *testing.T) {
	t.Parallel()

	_, err := resolveValueWithState("${state.missing}", map[string]interface{}{"present": "yes"})
	if err == nil {
		t.Fatal("expected interpolation failure for missing key, got nil")
	}
}

func TestResolveValueWithState_NonStringNoOpAndNested(t *testing.T) {
	t.Parallel()

	input := map[string]interface{}{
		"count": 5,
		"nested": map[string]interface{}{
			"line": "value=${state.payload.message}",
		},
		"arr": []interface{}{
			"${state.simple}",
			123,
		},
	}

	value, err := resolveValueWithState(input, map[string]interface{}{
		"simple": "ok",
		"payload": map[string]interface{}{
			"message": "nested",
		},
	})
	if err != nil {
		t.Fatalf("resolveValueWithState() error = %v", err)
	}

	gotMap, ok := value.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", value)
	}

	expected := map[string]interface{}{
		"count": 5,
		"nested": map[string]interface{}{
			"line": "value=nested",
		},
		"arr": []interface{}{
			"ok",
			123,
		},
	}

	if !reflect.DeepEqual(gotMap, expected) {
		t.Fatalf("unexpected resolved structure:\n got: %#v\nwant: %#v", gotMap, expected)
	}
}
