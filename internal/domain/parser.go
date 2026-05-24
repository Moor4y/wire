package domain

import (
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

var (
	// ErrEmptyWorkflow indicates the parsed workflow had no executable steps.
	ErrEmptyWorkflow = errors.New("workflow has no steps")
)

// WorkflowParser defines parsing and validation behavior for workflow sources.
type WorkflowParser interface {
	Parse(data []byte) (*Workflow, error)
}

// YAMLWorkflowParser parses workflows from YAML documents.
type YAMLWorkflowParser struct{}

// NewYAMLWorkflowParser creates a parser implementation backed by yaml.v3.
func NewYAMLWorkflowParser() *YAMLWorkflowParser {
	return &YAMLWorkflowParser{}
}

// Parse converts YAML into a Workflow and performs defensive validation.
func (p *YAMLWorkflowParser) Parse(data []byte) (*Workflow, error) {
	var wf Workflow
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return nil, fmt.Errorf("failed to parse workflow yaml: %w", err)
	}

	if len(wf.Steps) == 0 {
		return nil, ErrEmptyWorkflow
	}

	// Normalize nil maps to avoid repeated nil checks downstream.
	for i := range wf.Steps {
		if wf.Steps[i].With == nil {
			wf.Steps[i].With = map[string]interface{}{}
		}
	}

	if err := ValidateWorkflow(&wf); err != nil {
		return nil, err
	}

	return &wf, nil
}
