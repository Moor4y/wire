package mcpbridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"headless-orchestrator/internal/domain"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// ErrMCPExecutionFailed wraps external MCP failures so callers can halt one
// workflow run without destabilizing the daemon process.
type ErrMCPExecutionFailed struct {
	StepID string
	Cause  error
}

func (e ErrMCPExecutionFailed) Error() string {
	return fmt.Sprintf("mcp execution failed for step %q: %v", e.StepID, e.Cause)
}

func (e ErrMCPExecutionFailed) Unwrap() error {
	return e.Cause
}

// MCPExecutor implements engine.StepExecutor by calling external MCP tools.
type MCPExecutor struct {
	command       string
	args          []string
	env           []string
	clientName    string
	clientVersion string
	logger        *slog.Logger
}

// NewMCPExecutor configures an executor for stdio MCP server invocation.
func NewMCPExecutor(
	command string,
	args []string,
	env []string,
	clientName string,
	clientVersion string,
	logger *slog.Logger,
) (*MCPExecutor, error) {
	if strings.TrimSpace(command) == "" {
		return nil, errors.New("mcp command is required")
	}
	if strings.TrimSpace(clientName) == "" {
		clientName = "headless-orchestrator"
	}
	if strings.TrimSpace(clientVersion) == "" {
		clientVersion = "dev"
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &MCPExecutor{
		command:       command,
		args:          append([]string(nil), args...),
		env:           append([]string(nil), env...),
		clientName:    clientName,
		clientVersion: clientVersion,
		logger:        logger,
	}, nil
}

// Execute performs a single tool call over MCP stdio and converts the result
// into a map for state merge by the engine.
func (e *MCPExecutor) Execute(
	ctx context.Context,
	step domain.Step,
	state map[string]interface{},
) (map[string]interface{}, error) {
	_ = state // state is available for future parameter templating logic.

	c, err := client.NewStdioMCPClient(e.command, e.env, e.args...)
	if err != nil {
		return nil, e.wrapFailure(step.ID, "create stdio client", err)
	}
	defer c.Close()

	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    e.clientName,
		Version: e.clientVersion,
	}
	if _, err := c.Initialize(ctx, initReq); err != nil {
		return nil, e.wrapFailure(step.ID, "initialize client", err)
	}

	toolName := strings.TrimSpace(step.Type)
	if override, ok := step.With["tool"].(string); ok && strings.TrimSpace(override) != "" {
		toolName = strings.TrimSpace(override)
	}
	if toolName == "" {
		return nil, e.wrapFailure(step.ID, "resolve tool name", errors.New("step type/tool is empty"))
	}

	arguments := make(map[string]interface{}, len(step.With))
	for key, value := range step.With {
		// "tool" is orchestration metadata when present and not part of tool input.
		if key == "tool" {
			continue
		}
		arguments[key] = value
	}

	req := mcp.CallToolRequest{}
	req.Params.Name = toolName
	req.Params.Arguments = arguments

	res, err := c.CallTool(ctx, req)
	if err != nil {
		return nil, e.wrapFailure(step.ID, "call tool", err)
	}
	if res == nil {
		return nil, e.wrapFailure(step.ID, "call tool", errors.New("nil tool result"))
	}
	if res.IsError {
		return nil, e.wrapFailure(step.ID, "tool returned error", errors.New("mcp tool reported failure"))
	}

	out, err := decodeToolResultToMap(res)
	if err != nil {
		return nil, e.wrapFailure(step.ID, "decode tool result", err)
	}
	return out, nil
}

func (e *MCPExecutor) wrapFailure(stepID string, stage string, err error) error {
	wrapped := ErrMCPExecutionFailed{
		StepID: stepID,
		Cause:  fmt.Errorf("%s: %w", stage, err),
	}
	e.logger.Error("mcp step execution failed", "step_id", stepID, "stage", stage, "error", err)
	return wrapped
}

func decodeToolResultToMap(res *mcp.CallToolResult) (map[string]interface{}, error) {
	if res.StructuredContent != nil {
		switch typed := res.StructuredContent.(type) {
		case map[string]interface{}:
			return typed, nil
		default:
			blob, err := json.Marshal(typed)
			if err != nil {
				return nil, fmt.Errorf("marshal structured content: %w", err)
			}
			var out map[string]interface{}
			if err := json.Unmarshal(blob, &out); err == nil {
				return out, nil
			}
		}
	}

	texts := make([]string, 0, len(res.Content))
	for _, part := range res.Content {
		if textPart, ok := part.(mcp.TextContent); ok {
			texts = append(texts, textPart.Text)
		}
	}

	if len(texts) == 0 {
		return map[string]interface{}{
			"result": res.StructuredContent,
		}, nil
	}

	combined := strings.Join(texts, "\n")
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(combined), &parsed); err == nil {
		return parsed, nil
	}

	return map[string]interface{}{
		"text": combined,
	}, nil
}
