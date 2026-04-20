# feat: add inline skills and CLI tool containers with many-to-many support

**Commit:** `4e280072`  
**Date:** 2026-04-20  
**Type:** feat (CRDs, Go controller, Helm)

## What

Core infrastructure for inline skills and improvements to the skill architecture:

### 1. Inline Skills (CRD + Controller)
New `InlineSkill` type in the Agent CRD with `name`, `description`, and `content`
fields. The controller:
- Generates a ConfigMap from inline skill contents
- Mounts each skill as `/skills/<name>/SKILL.md` via SubPath
- **Inline-only agents skip the init container entirely** — no OCI image pull
  needed, dramatically faster pod startup
- Mixed agents (inline + container) get both the ConfigMap mount and the
  skills-init container
- Name collision detection prevents inline skills from shadowing container skills

### 2. OCI Registry Authentication
New `ociAuthSecretRef` field on `SkillForAgent` allows specifying a
`kubernetes.io/dockerconfigjson` secret for pulling skill images from private
OCI registries. Falls back to a global default via `KAGENT_DEFAULT_OCI_AUTH_SECRET`.
The skills-init script sets `DOCKER_CONFIG` from the mounted secret.

### 3. SessionTokenLabel for MCP Servers
New `sessionTokenLabel` field on `McpServerTool` in the CRD. When set, the Python
runtime's `set_mcp_token` tool stores user tokens under this label, and the
`MCPOAuthToolWrapper` injects them on MCP calls. This completes the CRD-side
support for the per-user MCP auth feature.

### 4. Agent Access Metadata Generalization
The `setAgentAccessMetadata` handler function is refactored to accept any
`client.Object` (not just `*v1alpha2.Agent`), enabling reuse for SandboxAgent
and future resource types. Called during both create and update operations.

## Why

**Inline skills** are a major DX improvement. Previously, every skill required
building and pushing a container image, even for simple prompt-only skills that
are just a `SKILL.md` file. This created an unnecessarily high barrier to entry
for skill authorship and a ~30s startup penalty per skill (image pull). Inline
skills eliminate both: the content is stored directly in the Agent CR and mounted
via ConfigMap — no image, no pull, no init container.

**OCI auth** unblocks private registry deployments where skill images are stored
behind authentication (e.g., company-internal container registries).

**SessionTokenLabel** closes the loop on MCP multi-tenancy — without a CRD field,
there was no declarative way to tell the runtime which session state key to use
for a given MCP server's token.

## Scope of Changes

| Area | Files |
|------|-------|
| **CRDs** | `kagent.dev_agents.yaml`, `kagent.dev_sandboxagents.yaml` — inlineSkills array, sessionTokenLabel, ociAuthSecretRef |
| **API Types** | `agent_types.go` — InlineSkill struct, OCIAuthSecretRef, SessionTokenLabel |
| **Translator** | `adk_api_translator.go` — SAPAICore model translation, OCI auth resolution |
| **Manifest Builder** | `manifest_builder.go` — ConfigMap generation, SubPath mounts, inline/container/mixed logic, name collision detection |
| **Skills-init Template** | `skills-init.sh.tmpl` — DOCKER_CONFIG setup from OCI auth mount |
| **Template Engine** | `template.go` — inline skill names in PromptTemplateContext |
| **HTTP Handlers** | `agents.go` — generalized `setAgentAccessMetadata` |
| **Tests** | Comprehensive tests for OCI auth, inline-only, mixed skills, name collisions |
| **Helm CRDs** | Mirrored all CRD changes to Helm templates |
| **Test Fixtures** | New input/output fixtures for inline and mixed skill agents |
