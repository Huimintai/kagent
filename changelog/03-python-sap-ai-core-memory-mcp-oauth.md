# feat(python): add SAP AI Core provider, memory embeddings, and MCP OAuth token support

**Commit:** `ab81e991`  
**Date:** 2026-04-20  
**Type:** feat (Python ADK)

## What

Three major features added to the Python ADK:

### 1. SAP AI Core LLM Provider
Full Python implementation mirroring the Go ADK version:
- OAuth2 client_credentials flow with thread-safe token caching
- Orchestration Service template format (chat messages + tool definitions)
- Streaming SSE parsing with proper event boundary handling
- Tool/function calling with result serialization
- Error recovery with automatic token refresh on auth failures

### 2. Multi-Provider Memory Embeddings
Extends the memory service with real embedding generation across **6 providers**:
- OpenAI, Azure OpenAI, Ollama, SAP AI Core, Google/Vertex AI, Amazon Bedrock

All embeddings are L2-normalized and truncated to 768 dimensions using Matryoshka
representation learning, ensuring consistent vector sizes regardless of the
upstream model's native dimensionality.

### 3. MCP OAuth Token Support
Enables per-user, per-session OAuth token injection for MCP servers:
- `set_mcp_token` — an in-chat tool that lets users store personal access tokens
  in session state
- `MCPOAuthToolWrapper` — intercepts MCP tool calls and injects stored tokens as
  `Authorization: Bearer` headers
- Agent executor auto-maps `X-MCP-Token-*` HTTP headers to session state keys
- `_remote_a2a_tool.py` — propagates `Authorization` header from parent session
  to sub-agent A2A calls

## Why

**SAP AI Core**: Completes the provider support on the Python runtime side. The Go
ADK handles CRD reconciliation, but agent pods run the Python runtime — both need
the provider implementation for end-to-end functionality.

**Memory Embeddings**: Without real embeddings, the memory service could only do
keyword-based retrieval. Vector embeddings enable semantic search across agent
memory, significantly improving recall quality. Supporting multiple providers
ensures compatibility with whatever model infrastructure the deployment has.

**MCP OAuth**: In multi-user deployments, MCP servers often require user-specific
authentication (e.g., GitHub repos, Azure DevOps). Sharing a single service
account token would give all users access to the same resources. Session-scoped
tokens ensure each user authenticates with their own credentials.

## Scope of Changes

| Area | Files |
|------|-------|
| **Python Models** | New `_sap_ai_core.py` (542 lines) — full LLM implementation |
| **Memory Service** | New `_memory_service.py` (233 lines) — multi-provider embeddings |
| **MCP Tools** | New `mcp_oauth_tools.py` (467 lines), `set_mcp_token_tool.py` (105 lines) |
| **Agent Executor** | `_agent_executor.py` — auto-maps `X-MCP-Token-*` headers to session state |
| **A2A Protocol** | `_remote_a2a_tool.py` — auth header propagation to sub-agents |
| **Types/Config** | `types.py` — `auth_url` on EmbeddingConfig, `SessionTokenLabel`, model factory registration |
| **Tests** | New tests for A2A auth propagation and `set_mcp_token` tool |
