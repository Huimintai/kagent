# go/adk — Go Agent Development Kit

Runtime for executing declarative agents. Receives CRD specs as mounted ConfigMap and runs agent loops with tool integrations.

## Sub-packages

| Package | Role |
|---------|------|
| `pkg/a2a` | Agent-to-Agent client |
| `pkg/agent` | Agent creation from config |
| `pkg/app` | Application initialization |
| `pkg/auth` | Auth token handling |
| `pkg/cli` | CLI interface |
| `pkg/config` | Configuration loading |
| `pkg/embedding` | Text embedding |
| `pkg/mcp` | MCP registry/connection |
| `pkg/memory` | Vector memory tools |
| `pkg/models` | LLM provider implementations |
| `pkg/runner` | Agent execution loop |
| `pkg/session` | Session management |
| `pkg/skills` | Skill discovery/execution |
| `pkg/taskstore` | Task persistence |
| `pkg/telemetry` | OpenTelemetry tracing |
| `pkg/tools` | Tool abstractions (A2A, ask-user, skills) |
| `cmd/` | Entry points |
| `examples/` | Example agents |

## Module Boundary

Imports `go/api` only. NEVER imports `go/core`.

## Integration

- Receives agent spec as a mounted ConfigMap (translated from CRD by controller)
- Connects to MCP servers for tool execution
- Reports session state back to platform via HTTP client
