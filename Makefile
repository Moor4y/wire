ifeq ($(OS),Windows_NT)
	EXEEXT := .exe
else
	EXEEXT :=
endif

BINARY := bin/orchestrator$(EXEEXT)
WORKFLOW := examples/demo-workflow.yaml
MCP_COMMAND ?= npx
MCP_ARGS ?= -y,@acme/mcp-demo-server

.PHONY: build test run clean

build:
	go build -o $(BINARY) ./cmd/orchestrator

test:
	go test ./...

run: build
	$(BINARY) -workflow $(WORKFLOW) -mcp-command $(MCP_COMMAND) -mcp-args "$(MCP_ARGS)"

clean:
	go clean
	$(RM) -r bin
	$(RM) orchestrator.db orchestrator.db-wal orchestrator.db-shm
