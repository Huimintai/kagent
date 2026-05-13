# kagent-adk - Main Agent Development Kit

Primary runtime package. Executes agents, manages MCP toolsets, integrates with LLM providers, and exposes an A2A server.

## Structure

| Directory | Purpose |
|-----------|---------|
| `src/` | Source code (namespace: `kagent.adk`) |
| `tests/` | Unit tests and fixtures |

## Key Features

- Agent execution via Google ADK framework
- MCP toolset integration (stdio + SSE transports)
- Multi-model support (OpenAI, Anthropic, Bedrock, Ollama, SAP AI Core)
- A2A protocol server for inter-agent communication
- Session and memory management
- Artifact handling (stage, return, session-scoped paths)
- Skills plugin system
- CLI entry point: `kagent-adk`

## Dependencies

- `kagent-core` (config, tracing, A2A)
- `kagent-skills` (skill loading)
- `agentsts-adk` / `agentsts-core` (token exchange)
- `google-adk` >=1.28.1 (agent framework)
- `openai`, `anthropic[vertex]`, `boto3`, `ollama` (model providers)
- `mcp` >=1.25.0 (tool protocol)
- `a2a-sdk` >=0.3.23 (agent-to-agent)
- `fastapi` + `uvicorn` (HTTP server)

## Entry Point

```
kagent-adk = "kagent.adk.cli:run_cli"
```
