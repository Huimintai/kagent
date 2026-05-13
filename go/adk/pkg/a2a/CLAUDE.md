# pkg/a2a/

Agent-to-Agent (A2A) protocol support for the Go ADK runtime.

`agentcard.go` provides `EnrichAgentCard()` which populates an A2A AgentCard with skills derived from a Google ADK agent. `consts.go` defines protocol constants. `converter.go` / `executor.go` bridge between A2A protocol messages and ADK agent execution. `hitl.go` implements human-in-the-loop approval flows.

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `server/` | A2A HTTP server wrapper (`A2AServer`) with health endpoints, graceful shutdown, and OpenTelemetry instrumentation |
