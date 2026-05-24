# headless-orchestrator

`headless-orchestrator` is a headless, MCP-native workflow execution engine built in Go.
It parses YAML workflows, validates graph safety constraints, executes steps deterministically, and persists execution logs in SQLite.

## Project Architecture

- `cmd/orchestrator`
  - CLI entrypoint that wires parser, execution engine, SQLite store, and MCP executor.
- `internal/domain`
  - Workflow domain model and YAML parser (`gopkg.in/yaml.v3`).
  - Defensive validation for duplicate step IDs, missing dependencies, and circular dependencies.
- `internal/engine`
  - Sequential execution engine and state merge logic.
  - Context-aware execution loop for timeout/cancellation handling.
- `internal/store`
  - Repository interfaces and SQLite implementation (`github.com/mattn/go-sqlite3`).
  - WAL mode and single-connection tuning to reduce lock contention.
- `internal/mcp`
  - MCP bridge executor (`github.com/mark3labs/mcp-go`) that calls tools over stdio.
  - Supports `${state.<key>}` interpolation for step arguments.

## Quick Start (CLI Binary)

### 1) Build the binary

```bash
go build -o bin/orchestrator ./cmd/orchestrator
```

### 2) Run the demo workflow

The demo expects an MCP server that exposes `emit_message` and `consume_message` tools.

```bash
./bin/orchestrator \
  -workflow ./examples/demo-workflow.yaml \
  -mcp-command npx \
  -mcp-args "-y,@acme/mcp-demo-server"
```

### 3) Outputs and state

- Execution metadata and step logs are stored in `orchestrator.db` by default.
- Final merged workflow state is emitted in the CLI logs as JSON.

## Workflow Notes

- Workflow YAML shape:
  - root `steps` list
  - each step has `id`, `type`, optional `uses`, optional `with`
- Steps execute in YAML order.
- `with` supports state interpolation using `${state.<key>}`.
  - Example: `${state.message}` uses the `message` key produced by prior steps.

## Makefile Shortcuts

Use the root Makefile for common tasks:

- `make build`
- `make test`
- `make run`
- `make clean`
