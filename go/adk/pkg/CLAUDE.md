# go/adk/pkg — ADK Implementation Packages

All runtime packages for the Go Agent Development Kit.

## Packages

| Package | Role |
|---------|------|
| `a2a/` | Agent-to-Agent protocol client |
| `agent/` | Agent creation from declarative config |
| `app/` | Application initialization and lifecycle |
| `auth/` | Auth token acquisition and refresh |
| `cli/` | CLI interface for local agent execution |
| `config/` | Configuration loading from ConfigMap/files |
| `embedding/` | Text embedding for vector memory |
| `mcp/` | MCP server registry and connection management |
| `memory/` | Vector memory tools (pgvector-backed) |
| `models/` | LLM provider implementations (OpenAI, Anthropic, AI Core) |
| `runner/` | Agent execution loop (turn management, tool calls) |
| `session/` | Session state management |
| `skills/` | Skill discovery and execution |
| `taskstore/` | Task persistence (A2A task protocol) |
| `telemetry/` | OpenTelemetry tracing setup |
| `tools/` | Tool abstractions (A2A tools, ask-user, skill tools) |
