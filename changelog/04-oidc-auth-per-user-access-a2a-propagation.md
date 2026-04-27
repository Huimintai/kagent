# feat(go): add per-user agent access, MCP token forwarding, and A2A auth propagation

**Commit:** `68e3a60d`  
**Date:** 2026-04-23  
**Type:** feat (Go controller, HTTP server, auth)

## What

Implements the backend infrastructure for multi-user agent access control and
end-to-end authentication propagation:

### 1. Per-User Agent Access Control
Agents now carry two new Kubernetes annotations:
- `kagent.dev/user-id` — the owner's identity (email or ID)
- `kagent.dev/private-mode` — `"true"` (visible only to owner) or `"false"` (public)

These are persisted both as K8s annotations (source of truth for GitOps) and in the
database (for efficient filtering). The controller reconciler reads them during
upsert. Default: agents are private, owned by `admin@kagent.dev`.

### 2. A2A Auth Propagation (Go Side)
The Go remote A2A tool now reads the `Authorization` header from the parent session
context and forwards it to sub-agent A2A calls. This enables sub-agents to inherit
the user's bearer token for downstream service authentication without requiring
separate credential configuration per sub-agent.

### 3. MCP Token Header Forwarding
The authentication middleware now collects all `X-MCP-Token-*` headers from incoming
requests and forwards them upstream, enabling the UI's GitHub/enterprise OAuth tokens
to reach agent pods where they're needed for MCP server tools.

### 4. ADK Types & Database Model Updates
- `SessionTokenLabel` added to MCP server configs (HTTP and SSE)
- `AuthUrl` added to `EmbeddingConfig` for SAP AI Core embedding auth
- `UserID` and `PrivateMode` fields added to database Agent model

## Why

Enterprise deployments need multi-tenancy: multiple users sharing the same kagent
instance, each with their own agents and credentials. Without per-user access control,
any user could see/modify/delete any other user's agents. Without auth propagation,
sub-agents couldn't access user-specific resources (e.g., a GitHub MCP tool needing
the user's token to access their repos).

The annotation-based approach (rather than a separate RBAC system) keeps the
implementation simple and GitOps-compatible — agent ownership is visible in the
manifest YAML and can be managed through standard K8s tooling.

## Scope of Changes

| Area | Files |
|------|-------|
| **Go A2A Tool** | `go/adk/pkg/tools/remote_a2a_tool.go` — context-based auth propagation |
| **API Types** | `go/api/adk/types.go` — SessionTokenLabel, AuthUrl on EmbeddingConfig |
| **Database** | `go/api/database/models.go` — `UserID` and `PrivateMode` fields on Agent |
| **HTTP Types** | `go/api/httpapi/types.go` — UserID/PrivateMode in AgentResponse |
| **Auth Middleware** | `authn.go` — X-MCP-Token-* collection and forwarding |
| **HTTP Handlers** | `agents.go` — annotation read/write, `setAgentAccessMetadata` helper |
| **Utils** | `common.go` — annotation key constants and defaults |
| **Environment** | `kagent.go` — `KAGENT_DEFAULT_OCI_AUTH_SECRET` env var |
