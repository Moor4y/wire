package mcpbridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"headless-orchestrator/internal/domain"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

var statePlaceholderPattern = regexp.MustCompile(`\$\{state\.([a-zA-Z0-9_.-]+)\}`)

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
		resolved, err := resolveValueWithState(value, state)
		if err != nil {
			return nil, e.wrapFailure(step.ID, "resolve step arguments", fmt.Errorf("with.%s: %w", key, err))
		}
		arguments[key] = resolved
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

func resolveValueWithState(value interface{}, state map[string]interface{}) (interface{}, error) {
	switch typed := value.(type) {
	case string:
		return resolveStringWithState(typed, state)
	case map[string]interface{}:
		resolved := make(map[string]interface{}, len(typed))
		for key, inner := range typed {
			out, err := resolveValueWithState(inner, state)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", key, err)
			}
			resolved[key] = out
		}
		return resolved, nil
	case []interface{}:
		resolved := make([]interface{}, len(typed))
		for idx, inner := range typed {
			out, err := resolveValueWithState(inner, state)
			if err != nil {
				return nil, fmt.Errorf("[%d]: %w", idx, err)
			}
			resolved[idx] = out
		}
		return resolved, nil
	default:
		return value, nil
	}
}

func resolveStringWithState(input string, state map[string]interface{}) (string, error) {
	matches := statePlaceholderPattern.FindAllStringSubmatch(input, -1)
	if len(matches) == 0 {
		return input, nil
	}

	resolved := input
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		path := match[1]
		val, ok := lookupStatePath(state, path)
		if !ok {
			return "", fmt.Errorf("missing state value for path %q", path)
		}
		resolved = strings.ReplaceAll(resolved, match[0], fmt.Sprint(val))
	}

	return resolved, nil
}

func lookupStatePath(state map[string]interface{}, path string) (interface{}, bool) {
	if state == nil || strings.TrimSpace(path) == "" {
		return nil, false
	}

	segments := strings.Split(path, ".")
	var current interface{} = state
	for _, segment := range segments {
		asMap, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		next, exists := asMap[segment]
		if !exists {
			return nil, false
		}
		current = next
	}

	return current, true
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
