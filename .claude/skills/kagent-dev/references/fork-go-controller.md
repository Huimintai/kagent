# Go Controller Fork Customizations

This document maps every custom component in the Go controller layer relative to upstream kagent-dev/kagent.

**Source:** `go/` workspace (modules: `api`, `core`, `adk`)
**Upstream:** `github.com/kagent-dev/kagent` (main branch)
**Customization level:** ~80% custom at the controller layer; CRDs and translator are heavily extended.

---

## Sync Risk Legend

| Level | Meaning |
|-------|---------|
| **HIGH** | Core types or logic that diverge significantly from upstream — merge conflicts likely |
| **MEDIUM** | Extended with new fields/endpoints but structure mostly aligned — targeted merge needed |
| **LOW** | Additive files or standard K8s patterns — minimal upstream conflict |

---

## CRD Types (`go/api/v1alpha2/`)

### `agent_types.go` (~568 lines) — Sync Risk: HIGH

The primary CRD definition. Key types:

| Type | Purpose |
|------|---------|
| `Agent` | Main CRD resource |
| `AgentSpec` | Top-level spec (Type: Declarative or BYO) |
| `DeclarativeAgentSpec` | Runtime config, system message, model, tools, memory, context, A2A |
| `SharedDeploymentSpec` | Deployment fields shared by Declarative + BYO |
| `Tool` / `McpServerTool` | Tool references with approval and header support |
| `A2AConfig` | Agent-to-agent server configuration |
| `SkillForAgent` | OCI and Git-based skill fetching with auth |
| `MemorySpec` / `ContextConfig` | Long-term memory and event compression |

**Fork-specific fields not in upstream:**
- `MemorySpec` — TTL-based memory with embedding provider config
- `ContextConfig` — event history compaction with LLM summarization
- `A2AConfig.Skills` — A2A skill advertisement
- `SkillForAgent` — Git + OCI with auth secrets
- `McpServerTool.RequireApproval` — per-tool HITL governance
- `McpServerTool.Headers` — runtime header injection from Secrets/ConfigMaps
- `DeclarativeAgentSpec.PromptTemplate` — Go text/template for system messages
- `AllowedNamespaces` — cross-namespace reference control (Gateway API pattern)

**XValidations:**
- Type must be Declarative or BYO
- systemMessage vs systemMessageFrom mutually exclusive
- ServiceAccountName vs ServiceAccountConfig mutually exclusive

---

### `modelconfig_types.go` (~425 lines) — Sync Risk: HIGH

LLM provider configuration CRD.

| Provider | Config Struct | Fork-Specific |
|----------|--------------|---------------|
| OpenAI | `OpenAIConfig` | TokenExchange (GDCH), DefaultHeaders |
| Anthropic | `AnthropicConfig` | BaseURL override |
| AzureOpenAI | `AzureOpenAIConfig` | Standard |
| Ollama | `OllamaConfig` | Options map |
| Gemini | (via base) | Standard |
| GeminiVertexAI | `GeminiVertexAIConfig` | Credential mounting |
| AnthropicVertexAI | `AnthropicVertexAIConfig` | Region/project |
| Bedrock | `BedrockConfig` | Region, access key |
| **SAPAICore** | `SAPAICoreConfig` | **Fully custom** — ResourceGroup, AuthURL, OAuth2 |

**Fork-specific features:**
- `APIKeyPassthrough` — forward Bearer tokens directly to LLM providers
- `TLSConfig` — DisableVerify, CACertSecretRef, DisableSystemCAs
- `TokenExchangeConfig` — GDCH service account integration
- `DefaultHeaders` — global headers per model config
- Secret hash tracking — detect underlying secret changes for pod restarts

---

### `remotemcpserver_types.go` (~138 lines) — Sync Risk: MEDIUM

| Feature | Custom? |
|---------|---------|
| Protocol selection (SSE / STREAMABLE_HTTP) | Extended |
| HeadersFrom (Secret/ConfigMap resolution) | Fork-specific |
| Custom timeouts (Timeout, SseReadTimeout) | Fork-specific |
| TerminateOnClose | Fork-specific |
| AllowedNamespaces | Fork-specific |
| Tool discovery in status | Fork-specific |

---

### `common_types.go` (~165 lines) — Sync Risk: MEDIUM

Shared types: `AllowedNamespaces`, `ValueSource`, `ValueRef`. Gateway API-inspired bidirectional namespace handshake pattern.

---

### Other files in v1alpha2/

| File | Purpose | Sync Risk |
|------|---------|-----------|
| `modelproviderconfig_types.go` | Provider-level config | MEDIUM |
| `agentobject.go` | Interface for agent objects | LOW |
| `groupversion_info.go` | API registration | LOW |
| `zz_generated.deepcopy.go` | Generated — do not edit | N/A |

---

## ADK Config Types (`go/api/adk/types.go`)

### `types.go` (~576 lines) — Sync Risk: HIGH

Bridge between K8s CRDs and Python/Go ADK runtime. Serialized as `config.json` in a Secret, mounted into agent pods.

Key types: `AgentConfig`, model types (`OpenAI`, `AzureOpenAI`, `Anthropic`, `Gemini`, `Bedrock`, `SAPAICore`, `Ollama`), `HttpMcpServerConfig`, `SseMcpServerConfig`, `RemoteAgentConfig`, `MemoryConfig`, `NetworkConfig`, `AgentContextConfig`, `EmbeddingConfig`.

**Fork-specific features:**
- `BaseModel.TLSInsecureSkipVerify`, `TLSCACertPath`, `TLSDisableSystemCAs`
- `BaseModel.DefaultHeaders`, `APIKeyPassthrough`
- `TokenExchangeConfig` for GDCH
- Custom JSON marshaling with type field injection
- `sql.Scanner` / `driver.Valuer` interfaces for database storage
- `ParseModel()` factory with backwards-compatible `type` vs `provider` field handling
- `ModelToEmbeddingConfig()` conversion helper

**Dual maintenance:** changes here must match `python/packages/kagent-adk/src/kagent/adk/types.py`.

---

## Translator (`go/core/internal/controller/translator/agent/`)

### `adk_api_translator.go` (~1,360 lines) — Sync Risk: HIGH

The core translator. `TranslateAgent()` converts Agent CRD specs into K8s resources (Deployment, Service, Secret, ServiceAccount).

**Major subsystems:**

| Function | What It Translates |
|----------|-------------------|
| `translateModel()` | ModelConfig CRD → `adk.Model` (all 9 providers), secret resolution, TLS mounting, token exchange |
| `buildSkillsInitContainer()` | Unified skills-init script for Git + OCI with SSH host key scanning, auth mounting |
| `translateMCPServerTarget()` | Multi-type support: MCPServer, RemoteMCPServer, Service |
| `translateStreamableHttpTool()` / `translateSseHttpTool()` | HTTP/SSE connection params with header resolution |
| `applyProxyURL()` | Gateway API-aware routing with `x-kagent-host` header |
| `runPlugins()` | Extensibility hooks for custom resource types |

**Fork-specific patterns:**
- `modelDeploymentData` struct — accumulates env/volumes/mounts during model translation
- `mergeDeploymentData()` — careful merging to avoid duplicate mounts
- `ValidationError` — custom error type for user-actionable CRD errors
- `DefaultImageConfig` — configurable default images with runtime-specific variants
- Runtime-specific readiness probes (Go: fast, Python: conservative)

---

### `deployments.go` (~280 lines) — Sync Risk: HIGH

Deployment resolution logic.

| Function | Purpose |
|----------|---------|
| `resolveInlineDeployment()` | Declarative agent: image selection (Python/Go), default resources (100m CPU, 384Mi) |
| `resolveByoDeployment()` | BYO agent: custom image/command/args |

**Fork-specific:**
- Runtime repository derivation (Go runtime uses "golang-adk" image)
- Automatic resource defaults if unspecified
- Service account precedence: agent config > global default > auto-created
- Default labels (managed by kagent, mergeable with custom)

---

### `template.go` (~128 lines) — Sync Risk: MEDIUM

Prompt template engine.

| Function | Purpose |
|----------|---------|
| `resolvePromptSources()` | Fetches ConfigMap data for template includes |
| `buildTemplateContext()` | Constructs template variables (AgentName, ToolNames, SkillNames, etc.) |
| `executeSystemMessageTemplate()` | Executes Go `text/template` with custom `include()` function |

**Architecture doc:** `docs/architecture/prompt-templates.md`

---

### Other translator files

| File | Purpose | Sync Risk |
|------|---------|-----------|
| `manifest_builder.go` | K8s manifest construction helpers | MEDIUM |
| `compiler.go` | Agent config compilation | MEDIUM |
| `conversion.go` | Type conversion utilities | LOW |
| `utils.go` | Shared utilities | LOW |
| `proxy_test.go` | Proxy URL tests | LOW |
| `mcp_validation_test.go` | MCP validation tests | LOW |
| `skills-init.sh.tmpl` | Embedded init container script template | MEDIUM |
| `testdata/` | Golden test inputs/outputs (40+ test cases) | N/A |

---

## HTTP Server (`go/core/internal/httpserver/`)

### `server.go` (~200+ lines) — Sync Risk: MEDIUM

Server setup with `ServerConfig`:
- KubeClient, A2AHandler, MCPHandler
- WatchedNamespaces, DbClient
- Authenticator, Authorizer
- ProxyURL, Reconciler

**API endpoints:**

| Category | Endpoints |
|----------|-----------|
| Core | `/health`, `/version`, `/api/me` |
| Agents | `/api/agents` |
| Models | `/api/modelconfigs` |
| Sessions | `/api/sessions`, `/api/runs`, `/api/tasks` |
| Tools | `/api/tools`, `/api/toolservers` |
| Protocols | `/api/a2a/{ns}/{name}`, `/mcp` |
| Features | `/api/feedback`, `/api/memories` |
| Integrations | `/api/langgraph`, `/api/crewai` |

**Fork-specific:**
- OTEL instrumentation with custom span naming
- Multi-handler architecture (A2A, MCP, REST)
- Pluggable auth/authz layer
- `MemoryCleanupRunnable` — periodic TTL-based memory pruning (24h interval, leader-election aware)

---

### `auth/` — Sync Risk: LOW (additive)

| File | Purpose |
|------|---------|
| `authn.go` | Authenticator interface |
| `proxy_authn.go` | `ProxyAuthenticator` — trust-the-proxy model, extracts JWT from Bearer tokens |
| `authz.go` | Authorizer interface |

**Architecture doc:** `docs/OIDC_PROXY_AUTH_ARCHITECTURE.md`

---

### `handlers/` — Sync Risk: MEDIUM

20+ handler files for REST API endpoints. Key custom handlers:
- `agents.go` — agent CRUD with namespace filtering
- `sessions.go` — session management with source filtering
- `memories.go` — vector memory CRUD and batch operations
- `tasks.go` — A2A task management
- `feedback.go` — user feedback collection

---

## Controllers (`go/core/internal/controller/`)

### `agent_controller.go` (~122 lines) — Sync Risk: HIGH

Main agent reconciliation. Delegates to `KagentReconciler.ReconcileKagentAgent()`.

**Owned resource types:** Deployments, ConfigMaps, Secrets, Services, ServiceAccounts, custom plugin types.

**Dependency watches:**
- ModelConfig changes → agent reconcile
- RemoteMCPServer changes → agent reconcile
- MCPService changes → agent reconcile
- ConfigMap changes → agent reconcile (if referenced)

**Fork-specific:**
- Predicate chaining: GenerationChanged, LabelChanged, AnnotationChanged
- Leader election awareness
- Plugin integration for custom watches

**Architecture doc:** `docs/architecture/controller-reconciliation.md`

---

### `modelconfig_controller.go` (~132 lines) — Sync Risk: MEDIUM

Watches ModelConfig generation changes and Secret changes. Triggers reconciliation on auth key rotation.

Fork-specific: TLS cert secret tracking, cross-reference finding for dependent agents.

---

### `remotemcpserver_controller.go` (~66 lines) — Sync Risk: MEDIUM

Periodic requeue (60s) for tool server status refresh and tool discovery.

---

### `reconciler/` — Sync Risk: HIGH

Shared `kagentReconciler` used by all controllers. Contains:
- Atomic resource reconciliation (create/update Deployments, Services, Secrets)
- Database upserts (agent metadata, tool servers, tools)
- Status condition management

---

## Database (`go/core/internal/database/`)

### Structure — Sync Risk: HIGH

| Component | Path | Purpose |
|-----------|------|---------|
| Connection | `connect.go` | PostgreSQL/SQLite connection setup |
| Client | `client_postgres.go` | Database client implementation |
| sqlc config | `sqlc.yaml` | Code generation configuration |
| Generated code | `gen/` | sqlc-generated Go from SQL queries |
| Queries | `queries/*.sql` | Hand-written SQL (source of truth) |
| Migrations | `../pkg/migrations/` | golang-migrate embedded migrations (core + vector tracks) |

**Query files:**

| File | Purpose |
|------|---------|
| `agents.sql` | Agent metadata CRUD |
| `sessions.sql` | Session management with source filtering |
| `memory.sql` | Memory CRUD, TTL pruning, vector search |
| `tasks.sql` | A2A task tracking |
| `events.sql` | Session event storage |
| `feedback.sql` | User feedback |
| `push_notifications.sql` | Async notification queue |
| `langgraph.sql`, `crewai.sql` | Multi-framework support |

**Fork-specific:**
- Dual database support (SQLite + PostgreSQL)
- Memory TTL system with periodic pruning
- Source-based session filtering (exclude subagent sessions from UI)
- sqlc for type-safe query generation

**See also:** `references/database-migrations.md` for migration authoring rules.

---

## Helm Charts (`helm/`)

### `kagent-crds/` — Sync Risk: HIGH

CRD chart generated from `go/api/`. Install first. Contains:
- `kagent.dev_agents.yaml`
- `kagent.dev_modelconfigs.yaml`
- `kagent.dev_remotemcpservers.yaml`
- `kagent.dev_toolservers.yaml`
- `kagent.dev_modelproviderconfigs.yaml`
- `kagent.dev_memories.yaml`

**Must regenerate after CRD type changes:** `make controller-manifests`

---

### `kagent/` — Sync Risk: MEDIUM

Main application chart. Key templates:
- `controller-deployment.yaml` — controller pod with all config
- `controller-configmap.yaml` — runtime configuration
- RBAC (roles, rolebindings, service accounts)
- `ui-deployment.yaml` + `ui-configmap.yaml`
- `postgresql-secret.yaml`
- `toolserver-kagent.yaml` — built-in tool server
- `oauth2-proxy-templates.yaml` — auth proxy

---

## Design Proposals

These documents describe fork-specific features and their implementation context:

| EP | Feature | Status | Architecture Doc |
|----|---------|--------|-----------------|
| EP-1256 | Semantic memory (pgvector, memory tools, vector migration track) | Implemented | `design/EP-1256-memory.md` |
| EP-476 | OIDC authentication (proxy model, Casbin RBAC) | Implemented | `docs/OIDC_PROXY_AUTH_ARCHITECTURE.md` |
| EP-685 | First-class KMCP support (v1alpha1 → v1alpha2 migration) | Implemented | `design/EP-685-kmcp.md` |

---

## Extension Points

The fork provides several plugin/extension interfaces:

| Interface | File | Purpose |
|-----------|------|---------|
| `TranslatorPlugin` | `translator/agent/` | Custom resource type handling in translator |
| `Authenticator` | `httpserver/auth/authn.go` | Custom authentication schemes |
| `Authorizer` | `httpserver/auth/authz.go` | Custom authorization policies |
| `A2AHandlerMux` | `httpserver/` | A2A protocol extensions |
| `MCPHandler` | `httpserver/` | MCP protocol customization |

---

## Upstream Sync Checklist (Go Controller)

1. **Before syncing kagent-dev/kagent upstream:**
   - Check CRD type changes in `go/api/v1alpha2/` — field additions/removals need careful merging
   - Check translator changes — our `adk_api_translator.go` is heavily customized (1,360 lines)
   - Check database schema changes — migration conflicts are hardest to merge
   - Check HTTP server routes — endpoint additions may conflict

2. **After syncing:**
   ```bash
   make controller-manifests          # Regenerate CRDs + copy to Helm
   UPDATE_GOLDEN=true make -C go test # Regenerate golden files
   make -C go lint                    # Check for lint issues
   make -C go e2e                     # Full E2E test
   ```

3. **Merge strategy by area:**
   - **CRDs:** Apply upstream changes first, then re-add our custom fields. Run `make controller-manifests`.
   - **Translator:** Most conflicts here. Review function-by-function. Check that new upstream features don't conflict with our model translation or skills init.
   - **Database:** Migrations must be sequentially numbered. If upstream adds migrations, renumber ours. Never edit existing migrations.
   - **Helm:** Compare `values.yaml` defaults carefully. Template changes need manual review.
   - **Controllers:** Usually safe to merge — our customizations are in the reconciler, not the controller setup.

4. **Dual-maintenance files:**
   - `go/api/adk/types.go` ↔ `python/packages/kagent-adk/src/kagent/adk/types.py` — keep in sync
   - CRD YAML in `helm/kagent-crds/` ↔ `go/api/config/crd/bases/` — regenerated, don't manually edit
