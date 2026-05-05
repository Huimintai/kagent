# Changelog

All notable changes to the kagent fork are documented here.

## [Unreleased]

### ScheduledRun — Cron-Based Agent Triggering
- Add `ScheduledRun` CRD (`kagent.dev/v1alpha2`) for defining cron schedules that trigger agents with a predefined prompt
- Add in-memory cron scheduler (robfig/cron/v3) running as a leader-elected controller-runtime Runnable
- Add `ScheduledRunController` that validates cron expressions, verifies agent references, and syncs schedules
- Support concurrency policies: `Forbid` (default), `Allow`, `Replace`
- Support `suspend` flag to pause/resume scheduling without deleting the resource
- Track run history in CRD status (capped by `maxRunHistory`, default 10)
- Add HTTP API endpoints: `GET/POST /api/scheduledruns`, `GET/PUT/DELETE /api/scheduledruns/{ns}/{name}`, `POST /api/scheduledruns/{ns}/{name}/trigger` (manual trigger)
- Add full UI CRUD: list page (`/schedules`), create/edit page (`/schedules/new`), detail page with run history (`/schedules/{ns}/{name}`)
- Add RBAC rules for `scheduledruns`, `scheduledruns/status`, `scheduledruns/finalizers` in Helm getter/writer roles
- Add `kagent-feature` skill for master-sub agent orchestration of end-to-end feature development

### Build & CI
- Add `BUILD_MODE=local` with Dockerfile.local for fast local builds (controller, Go ADK, skills-init, UI)
- Split CI skills into kind (local) and prod (remote) deployment guides
- Update kagent-dev skill with fork workflow references and evaluation cases

### SAP AI Core Provider (Go)
- Add SAP AI Core model provider via Orchestration Service with OAuth2 token management
- Support SAP AI Core in ModelConfig CRD (baseUrl, resourceGroup, authUrl fields)
- Add model listing and provider config handlers for SAP AI Core

### SAP AI Core Provider (Python) & MCP OAuth
- Add SAP AI Core LLM provider for Python ADK with streaming and tool support
- Add SAP AI Core embedding backend for agent memory service
- Add per-user MCP OAuth 2.1 token flow (initiate + complete) and session-scoped token storage
- Add A2A auth header propagation for remote agent tool calls

### OIDC Login & Per-User Access Control
- Add OIDC-based user identification via OAuth2-proxy headers
- Add per-user agent ownership and private mode (annotations + database fields)
- Add A2A authenticator with MCP token header forwarding
- Update reconciler to sync user access metadata from CRD annotations to database

### UI: Authentication & GitHub OAuth
- Add GitHub OAuth connect/disconnect flow with multi-instance support
- Add OIDC user identity proxy and user store with localStorage persistence
- Add Identicon component for deterministic user avatars

### UI: Rebrand, Feature Flags & Agent Discovery
- Rebrand to DBCI kagent Playground with SAP layout (header, footer, onboarding wizard)
- Add runtime-configurable feature flags via ConfigMap (disable model/MCP creation, allowed namespaces, protected agents)
- Add agent filtering toolbar with search, privacy tabs, and classification badges
- Add category grouping and namespace filtering in agent list

### UI: Agent Editor, Model Pages & Misc
- Update agent creation page with inline skill and tool container support
- Add SAP AI Core icon and provider option in model creation
- Update MCP server, tools, and model listing pages for multi-namespace support
- Remove package-lock.json (switched to different package manager)

### Inline Skills & CLI Tool Containers
- Add InlineSkill CRD type for prompt-based skills defined directly in Agent spec
- Add CLI tool container support with OCI image and Git repository references
- Support many-to-many agent-to-skill mapping with name collision detection
- Add skills-init container configuration (resources, env vars) in Agent CRD
