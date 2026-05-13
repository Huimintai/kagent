# python/ - UV Workspace

Python agent runtimes and SDKs managed as a UV workspace.

## Workspace Config

- **Manager:** UV (pyproject.toml at root)
- **Python:** >=3.10
- **Lint:** Ruff (line-length=120, targets: E, F, W, B, Q, I, ASYNC, T20)
- **Test:** pytest + pytest-asyncio (asyncio_mode=auto)

## Members

| Path | Package | Description |
|------|---------|-------------|
| `packages/kagent-adk` | kagent-adk 0.3.0 | Main ADK runtime (agent execution, MCP, models) |
| `packages/kagent-core` | kagent-core 0.1.0 | Shared library (config, A2A, tracing) |
| `packages/kagent-skills` | kagent-skills 0.1.0 | Skill discovery and loading |
| `packages/kagent-langgraph` | kagent-langgraph 0.1.0 | LangGraph integration with A2A |
| `packages/kagent-openai` | kagent-openai 0.1.0 | OpenAI Agents SDK integration |
| `packages/agentsts-adk` | agentsts-adk 0.1.0 | ADK-specific STS integration |
| `packages/agentsts-core` | agentsts-core 0.1.0 | RFC 8693 token exchange client |
| `samples/adk/*` | sample agents | ADK-based sample agents |
| `samples/langgraph/*` | sample agents | LangGraph-based samples |
| `samples/openai/*` | sample agents | OpenAI-based samples |

## Dependency Graph

```
kagent-adk
  ├── kagent-core
  ├── kagent-skills
  ├── agentsts-adk
  │     └── agentsts-core
  └── (google-adk, openai, anthropic, mcp, a2a-sdk)

kagent-langgraph
  └── kagent-core

kagent-openai
  ├── kagent-core
  └── kagent-skills
```

## Commands

| Task | Command |
|------|---------|
| Install all | `uv sync` |
| Run tests | `uv run pytest` |
| Lint | `uv run ruff check .` |
| Format | `uv run ruff format .` |
