package domain

import "testing"

func TestValidateWorkflow_DuplicateStepID(t *testing.T) {
	t.Parallel()

	wf := &Workflow{
		Steps: []Step{
			{ID: "a", Type: "tool_a"},
			{ID: "a", Type: "tool_b"},
		},
	}

	err := ValidateWorkflow(wf)
	if err == nil {
		t.Fatal("expected duplicate step id error, got nil")
	}
}

func TestValidateWorkflow_CircularDependency(t *testing.T) {
	t.Parallel()

	wf := &Workflow{
		Steps: []Step{
			{ID: "step-1", Type: "tool_a", Uses: "step-3"},
			{ID: "step-2", Type: "tool_b", Uses: "step-1"},
			{ID: "step-3", Type: "tool_c", Uses: "step-2"},
		},
	}

	err := ValidateWorkflow(wf)
	if err == nil {
		t.Fatal("expected circular dependency error, got nil")
	}
}
