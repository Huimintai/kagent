# feat: add SAP AI Core model provider via Orchestration Service

**Commit:** `1e7df67b`  
**Date:** 2026-04-20  
**Type:** feat (Go ADK, CRDs)

## What

Adds SAP AI Core as a first-class model provider in the Go ADK. The implementation
talks to SAP AI Core's **Orchestration Service** (`/v2/completion` endpoint) rather
than direct model endpoints, providing a unified interface to all models available
through the orchestration layer (Claude, GPT, etc.).

Key capabilities:
- **OAuth2 client_credentials** token management with thread-safe caching and
  automatic invalidation on 401/403 errors
- **Automatic deployment URL resolution** by querying `/v2/lm/deployments` for
  the latest running "orchestration" scenario
- Full **streaming and non-streaming** response support
- **Tool/function calling** translation to the Orchestration Service template format
- Retry logic with exponential backoff for transient errors (401, 403, 502, etc.)

A new `SAPAICoreConfig` struct is added to the `ModelConfigSpec` CRD with fields:
`baseUrl`, `resourceGroup`, and `authUrl`.

## Why

Kagent deployments in SAP environments cannot always reach external LLM provider APIs
directly. SAP AI Core's Orchestration Service acts as a gateway that:
1. Centralizes model access behind SAP's auth infrastructure (client credentials flow)
2. Provides access to multiple model families through a single endpoint
3. Handles compliance, auditing, and quota management at the platform level

Without this, SAP customers would need direct API keys for each LLM provider, which
may not be available or permitted in enterprise environments.

## Scope of Changes

| Area | Files |
|------|-------|
| **Go ADK Models** | New `go/adk/pkg/models/sapaicore.go` (OAuth + deployment resolution), `sapaicore_adk.go` (LLM interface implementation) |
| **Agent Factory** | `go/adk/pkg/agent/agent.go` — added `SAPAICore` case to `CreateLLM()` |
| **CRDs** | `kagent.dev_modelconfigs.yaml`, `kagent.dev_modelproviderconfigs.yaml` — new `sapAICore` config block and provider enum |
| **API Types** | `go/api/v1alpha2/modelconfig_types.go` — `SAPAICoreConfig` struct, `ModelProviderSAPAICore` constant |
| **HTTP Handlers** | `modelconfig.go`, `modelproviderconfig.go`, `models.go` — SAPAICore create/update/list support |
| **Environment** | `go/core/pkg/env/providers.go` — new `SAP_AI_CORE_*` env vars |
| **Helm** | CRD templates mirroring base CRD changes |
