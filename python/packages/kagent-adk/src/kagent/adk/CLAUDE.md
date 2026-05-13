# kagent.adk - ADK Implementation

Core agent runtime modules and subpackages.

## Subpackages

| Directory | Purpose |
|-----------|---------|
| `artifacts/` | Artifact management (stage, return, session-scoped paths) |
| `converters/` | Event/part/request conversion between ADK and external formats |
| `models/` | LLM provider adapters (OpenAI, Anthropic, Bedrock, Ollama, SAP AI Core) |
| `tools/` | Built-in tools (bash, file, MCP OAuth, memory, skills, ask-user) |

## Top-Level Modules

| Module | Role |
|--------|------|
| `cli.py` | CLI entry point (`kagent-adk` command) |
| `_agent_executor.py` | Main agent execution loop |
| `_a2a.py` | A2A protocol server setup |
| `_mcp_toolset.py` | MCP tool connection and management |
| `_session_service.py` | Session persistence |
| `_memory_service.py` | Memory read/write |
| `_token.py` | Token source management |
| `_approval.py` | Human-in-the-loop approval flow |
| `_lifespan.py` | FastAPI lifespan management |
| `_llm_passthrough_plugin.py` | LLM call passthrough plugin |
| `_remote_a2a_tool.py` | Remote A2A agent invocation tool |
| `sandbox_code_executer.py` | Sandboxed code execution |
| `types.py` | Shared type definitions |
