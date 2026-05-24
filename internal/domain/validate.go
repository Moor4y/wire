package domain

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrDuplicateStepID signals an invalid workflow with repeated step IDs.
	ErrDuplicateStepID = errors.New("duplicate step id")
	// ErrStepDependencyMissing signals a dependency references an unknown step.
	ErrStepDependencyMissing = errors.New("step dependency does not exist")
	// ErrCircularDependency signals a cycle in the workflow dependency graph.
	ErrCircularDependency = errors.New("circular dependency detected")
)

// ValidateWorkflow enforces structural and graph-level constraints.
func ValidateWorkflow(wf *Workflow) error {
	if wf == nil {
		return errors.New("workflow is nil")
	}
	if len(wf.Steps) == 0 {
		return ErrEmptyWorkflow
	}

	stepsByID := make(map[string]Step, len(wf.Steps))
	for i, step := range wf.Steps {
		if strings.TrimSpace(step.ID) == "" {
			return fmt.Errorf("step[%d] has empty id", i)
		}
		if strings.TrimSpace(step.Type) == "" {
			return fmt.Errorf("step[%d] (%s) has empty type", i, step.ID)
		}
		if _, exists := stepsByID[step.ID]; exists {
			return fmt.Errorf("%w: %s", ErrDuplicateStepID, step.ID)
		}
		stepsByID[step.ID] = step
	}

	// Build adjacency map where edge A->B means A depends on B.
	adj := make(map[string]string, len(wf.Steps))
	for _, step := range wf.Steps {
		dep := strings.TrimSpace(step.Uses)
		if dep == "" {
			continue
		}
		if _, ok := stepsByID[dep]; !ok {
			return fmt.Errorf("%w: step %q uses %q", ErrStepDependencyMissing, step.ID, dep)
		}
		adj[step.ID] = dep
	}

	// DFS color states: 0=unvisited, 1=visiting, 2=visited.
	color := make(map[string]uint8, len(wf.Steps))
	path := make([]string, 0, len(wf.Steps))
	var visit func(stepID string) error
	visit = func(stepID string) error {
		switch color[stepID] {
		case 1:
			path = append(path, stepID)
			return fmt.Errorf("%w: %s", ErrCircularDependency, strings.Join(path, " -> "))
		case 2:
			return nil
		}

		color[stepID] = 1
		path = append(path, stepID)

		if dep, ok := adj[stepID]; ok {
			if err := visit(dep); err != nil {
				return err
			}
		}

		path = path[:len(path)-1]
		color[stepID] = 2
		return nil
	}

	for _, step := range wf.Steps {
		if err := visit(step.ID); err != nil {
			return err
		}
	}

	return nil
}
