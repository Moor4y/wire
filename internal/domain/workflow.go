package domain

// Workflow defines a workflow declaration loaded from YAML.
// Steps are executed in declared order by the engine once validated.
type Workflow struct {
	Steps []Step `yaml:"steps"`
}

// Step is a single executable unit in a workflow.
//
// ID: unique identifier used for dependency edges and status tracking.
// Type: logical type/tool name used by StepExecutor implementations.
// Uses: dependency edge to another step ID (optional).
// With: arbitrary parameters passed to the executor.
type Step struct {
	ID   string                 `yaml:"id"`
	Type string                 `yaml:"type"`
	Uses string                 `yaml:"uses,omitempty"`
	With map[string]interface{} `yaml:"with,omitempty"`
}
