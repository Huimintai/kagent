# Python ADK Fork Customizations

This document maps every custom module in `kagent-adk` relative to upstream `google-adk`.

**Package:** `python/packages/kagent-adk/src/kagent/adk/`
**Upstream:** `google-adk` (pip: `google-adk>=1.25.0`)
**Relationship:** kagent-adk subclasses/wraps upstream — does not patch or vendor it.

---

## Sync Risk Legend

| Level | Meaning |
|-------|---------|
| **HIGH** | Subclasses or overrides upstream class — API changes in google-adk may break this |
| **MEDIUM** | Depends on upstream types/interfaces but doesn't subclass — type signature changes may break |
| **LOW** | Additive module — no upstream dependency beyond BaseTool/BaseLlm base classes |

---

## Core Executor

### `_agent_executor.py` (~743 lines) — Sync Risk: HIGH

**Extends:** `google.adk.a2a.executor.a2a_agent_executor.A2aAgentExecutor`

The most critical customization. Overrides the upstream executor with:

| Customization | Why | Upstream Behavior |
|---------------|-----|-------------------|
| Per-request Runner lifecycle | MCP toolset cleanup across requests; prevents connection leaks | Caches single Runner instance |
| `_resolve_runner()` / `_safe_close_runner()` | Creates fresh Runner per request, isolated cleanup with anyio cancel scope | No equivalent |
| OpenTelemetry span attributes | `set_kagent_span_attributes()` / `clear_kagent_span_attributes()` — tracks user_id, task_id, session_id, invocation_id | Basic tracing only |
| Ollama JSON parse error handling | User-friendly messages when model doesn't support function calling | Generic error |
| CancelledError handling | Task cancellation cleanup with proper status event | Not handled |
| Partial event filtering | During streaming, only non-partial events accumulate into final result | All events accumulated |
| Session naming | Extracts name from first TextPart, truncates to 20 chars + "..." | No session naming |
| Request header forwarding | Stores request headers in session state; maps `X-MCP-Token-*` → `mcp_token:*` | No header forwarding |
| Invocation ID tracking | Adds `invocation_id` to final event metadata for telemetry correlation | No invocation tracking |
| HITL resume handling | `_find_pending_confirmations()`, `_build_confirmation_payload()`, `_process_hitl_decision()` | Basic confirmation only |

**Key overridden methods:**
- `_handle_request()` — Main request handler with header mapping and HITL resume logic
- `_prepare_session()` — Session creation with name extraction and source tagging

**What to check on upstream sync:**
- `A2aAgentExecutor.__init__()` signature — we pass custom converters via adapter functions
- `_handle_request()` / `_prepare_session()` — we override these entirely
- `Runner` constructor and `close()` — our per-request lifecycle depends on these
- Event converter and request converter callback signatures

---

### `_remote_a2a_tool.py` (~505 lines) — Sync Risk: LOW (custom replacement)

**Replaces:** upstream `AgentTool(RemoteA2aAgent(...))` pairing. Does NOT subclass upstream.

| Class | Purpose |
|-------|---------|
| `KAgentRemoteA2ATool(BaseTool)` | Remote A2A agent invocation with HITL propagation |
| `KAgentRemoteA2AToolset(BaseToolset)` | Wrapper for proper httpx client cleanup on Runner.close() |
| `_SubagentInterceptor` | Injects `x-user-id` and `x-kagent-source` headers on every outgoing request |
| `SubagentSessionProvider(Protocol)` | Protocol for tools that expose a `subagent_session_id` property |

**Custom features not in upstream:**
- **HITL propagation** — surfaces subagent approval requests to parent agent UI
- **Two-phase call/resume** — Phase 1: initial call with HITL pause; Phase 2: forward user decision
- **Live activity viewing** — subagent session ID tracked before tool runs, UI polls it
- **User isolation** — authenticated user ID + Authorization header forwarded to subagent
- **Batch decisions** — mixed approve/reject per tool in a single response
- **Usage metadata extraction** — collects subagent's final LLM usage for parent display

**What to check on upstream sync:**
- If upstream adds HITL to `AgentTool`/`RemoteA2aAgent`, evaluate merging vs keeping custom
- `BaseTool` and `BaseToolset` interfaces — we depend on `run_async()` and `close()` contracts
- A2A client/transport APIs (`a2a.client`, `ClientFactory`) — we use these directly

**Architecture doc:** `docs/architecture/a2a-subagents.md`

---

### `_approval.py` (~64 lines) — Sync Risk: LOW

**Uses:** `google.adk.tools.tool_context.ToolContext.request_confirmation()` (stable public API)

Single function `make_approval_callback()` that creates a `before_tool_callback` for HITL tool approval. Uses ADK's native confirmation mechanism — low risk because it calls a public API rather than subclassing.

**Architecture doc:** `docs/architecture/human-in-the-loop.md`

---

### `_a2a.py` (~224 lines) — Sync Risk: MEDIUM

**Uses:** `google.adk.apps.App`, `google.adk.runners.Runner`, `google.adk.agents.BaseAgent`

`KAgentApp` is the main application factory. Wires together:
- Custom `A2aAgentExecutor` (our override)
- `KAgentSessionService` for HTTP-based session persistence
- `KagentMemoryService` for long-term memory
- `KAgentTokenService` for K8s token management
- Health check and thread dump endpoints
- `kagent.core.a2a` components for task store switching

**What to check on upstream sync:**
- `App` constructor and `build()` method — we override how the FastAPI app is composed
- `A2aAgentExecutorConfig` — our custom config class may conflict with upstream changes

---

## Services

### `_session_service.py` (~185 lines) — Sync Risk: MEDIUM

**Extends:** `google.adk.sessions.base_session_service.BaseSessionService`

HTTP-based session persistence via kagent controller API (`/api/sessions`). Replaces upstream's in-memory default.

| Method | Custom Behavior |
|--------|----------------|
| `create_session()` | POST to controller, supports source tagging ("agent" for subagent sessions) |
| `get_session()` | GET with event ordering (asc) for ADK delta calculations |
| `list_sessions()` | GET with pagination (limit parameter) |
| `delete_session()` | DELETE via controller API |

**What to check on upstream sync:** `BaseSessionService` interface changes (method signatures, new required methods).

---

### `_memory_service.py` (~657 lines) — Sync Risk: MEDIUM

**Extends:** `google.adk.memory.BaseMemoryService`

Full semantic memory system with:
- Multi-provider embedding (OpenAI, Azure, Ollama, Google, Bedrock, SAP AI Core)
- LLM-based session summarization before embedding
- Batch storage via `/api/memories/sessions/batch`
- Vector search with `min_score` threshold
- Matryoshka embedding dimension truncation to 768 + L2 normalization
- TTL support for memory entries
- Async background task scheduling for non-blocking saves

**What to check on upstream sync:** `BaseMemoryService` interface, `SearchMemoryResponse`, `MemoryEntry` types.

---

### `_mcp_toolset.py` (~70 lines) — Sync Risk: HIGH

**Extends:** `google.adk.tools.mcp_tool.mcp_toolset.McpToolset`

Wraps upstream MCP toolset with enhanced error handling:
- `CancelledError` enrichment with context information
- Anyio cross-task cancel scope error suppression
- `is_anyio_cross_task_cancel_scope_error()` helper

**What to check on upstream sync:** `McpToolset` class changes, especially `__init__()` and error handling paths.

---

### `_token.py` (~81 lines) — Sync Risk: LOW

No upstream equivalent. `KAgentTokenService` reads K8s service account tokens from `/var/run/secrets/tokens/kagent-token` with periodic refresh (60s). Injects Bearer token and agent name header into httpx requests.

---

### `_lifespan.py` (~37 lines) — Sync Risk: LOW

No upstream equivalent. `LifespanManager` composes multiple FastAPI lifespans by recursively nesting context managers. Enables multiple startup/shutdown lifecycle handlers.

---

## Converters

### `converters/event_converter.py` (~355 lines) — Sync Risk: MEDIUM

**Depends on:** `google.adk.events.event.Event`, `google.adk.agents.invocation_context.InvocationContext`

Main converter: `convert_event_to_a2a_events()`. Transforms ADK events into A2A `TaskStatusUpdateEvent` / `TaskArtifactUpdateEvent`.

Custom features:
- Partial event filtering (only non-partial accumulate into final)
- Long-running tool metadata marking
- Subagent session ID stamping on function_call DataParts
- Error code → human-readable message mapping
- Artifact ID generation with version tracking

**What to check on upstream sync:** `Event` class fields, `InvocationContext` properties, new event types.

---

### `converters/part_converter.py` (~100+ lines) — Sync Risk: MEDIUM

Bidirectional conversion: A2A parts <-> GenAI parts. Handles DataPart type detection via metadata keys (`function_call`, `function_response`, etc.), file URI handling, base64 encoding.

**Depends on:** `google.genai.types`, `a2a.types`

---

### `converters/request_converter.py` (~36 lines) — Sync Risk: LOW

`convert_a2a_request_to_adk_run_args()` — extracts user ID from request context, configures streaming mode.

---

### `converters/error_mappings.py` (~50 lines) — Sync Risk: LOW

Maps Gemini `FinishReason` codes to user-friendly error messages. Safety, token limit, and function call error handling.

---

## Model Providers

### `models/_anthropic.py` (~44 lines) — Sync Risk: HIGH

**Extends:** `google.adk.models.anthropic_llm.AnthropicLlm`

Adds API key passthrough from request bearer token, custom base_url, custom headers.

---

### `models/_openai.py` (~80+ lines) — Sync Risk: MEDIUM

**Extends:** `google.adk.models.BaseLlm`

Custom OpenAI implementation (not subclassing upstream's OpenAI LLM). Adds:
- GDCH token exchange for OAuth flows
- Thought signature handling for Gemini deep research
- Custom TLS/SSL context creation
- API key passthrough

Also includes `AzureOpenAI(BaseLlm)`.

---

### `models/_sap_ai_core.py` (~80+ lines) — Sync Risk: LOW

**Extends:** `google.adk.models.BaseLlm` (base class only)

Fully custom SAP AI Core provider via Orchestration Service:
- OAuth token caching with 2-min buffer
- Dynamic deployment URL resolution by scenario
- Bearer token injection with AI-Resource-Group header

No upstream equivalent.

---

### `models/_bedrock.py` (~60+ lines) — Sync Risk: LOW

**Extends:** `google.adk.models.BaseLlm` (base class only)

AWS Bedrock via Converse API. Boto3 integration, tool use payload conversion, multi-model support.

---

### `models/_ollama.py` — Sync Risk: LOW

Ollama SDK integration via `create_ollama_llm()`. Option type conversion from string config.

---

### `models/_token_source.py` — Sync Risk: LOW

Token auth abstraction. Shared between providers that use bearer token auth.

---

## Custom Tools

All tools extend `google.adk.tools.base_tool.BaseTool`. Sync risk is **LOW** for all — they are purely additive.

| Tool | File | Purpose |
|------|------|---------|
| `AskUserTool` | `tools/ask_user_tool.py` | Interactive Q&A with predefined choices; two-phase via request_confirmation |
| `SaveMemoryTool` | `tools/memory_tools.py` | Save content to long-term semantic memory |
| `LoadMemoryTool` | `tools/memory_tools.py` | Search and load memories via similarity |
| `PrefetchMemoryTool` | `tools/prefetch_memory_tool.py` | Inject relevant memories on first message |
| `InitiateMcpOAuthTool` | `tools/mcp_oauth_tools.py` | Start OAuth flow for MCP server auth |
| `CompleteMcpOAuthTool` | `tools/mcp_oauth_tools.py` | Complete OAuth flow and store token |
| `SetMcpTokenTool` | `tools/set_mcp_token_tool.py` | Per-user MCP token storage in session |
| `SkillsTool` | `tools/skill_tool.py` | Discover and load skills from /skills directory |
| `SkillsToolset` | `tools/skills_toolset.py` | Toolset wrapper for skills integration |
| `SkillsPlugin` | `tools/skills_plugin.py` | Plugin for skills discovery |
| `BashTool` | `tools/bash_tool.py` | Shell command execution |
| `FileReadTool` / `FileWriteTool` | `tools/file_tools.py` | File read/write/delete operations |
| `MCP toolset wrapper` | `tools/mcp_toolset.py` | MCP tool discovery and execution |

---

## Configuration Types

### `types.py` (~655 lines) — Sync Risk: LOW

Defines the entire agent configuration model (`AgentConfig`) that mirrors the Go ADK types. This file is the Python-side contract for `config.json`.

Key classes:
- `AgentConfig` — main config with `to_agent()` factory method
- LLM configs: `OpenAI`, `AzureOpenAI`, `Anthropic`, `Ollama`, `Gemini`, `Bedrock`, `SAPAICore`, etc.
- `HttpMcpServerConfig`, `SseMcpServerConfig` — tool connection configs
- `RemoteAgentConfig` — agent-to-agent references
- `MemoryConfig`, `ContextConfig`, `EmbeddingConfig`, `NetworkConfig`

**Dual maintenance required:** changes here must match `go/api/adk/types.go`.

---

## Upstream Sync Checklist (Python ADK)

1. **Before syncing google-adk:**
   - Check `A2aAgentExecutor` API changes (constructor, `_handle_request`, `_prepare_session`)
   - Check `Runner` constructor/close changes (per-request lifecycle depends on this)
   - Check `BaseTool`, `BaseToolset`, `ToolContext` interface changes
   - Check `BaseSessionService` and `BaseMemoryService` interface changes
   - Check `McpToolset` changes (our `_mcp_toolset.py` subclasses it)
   - Check event converter callback signatures

2. **After syncing:**
   - Run `make -C python lint` and `make -C python test`
   - Test HITL flow end-to-end (most fragile custom code path)
   - Test subagent calling (KAgentRemoteA2ATool)
   - Test memory save/load cycle
   - Test each LLM provider (at least OpenAI + one custom like SAP AI Core)

3. **If upstream adds competing features:**
   - Memory service — evaluate merging if upstream adds persistence
   - HITL on subagents — evaluate if upstream `AgentTool` adds `input_required` handling
   - Session naming — adopt upstream if they add it
