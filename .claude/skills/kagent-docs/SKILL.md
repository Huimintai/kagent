---
name: kagent-docs
description: >
  Documentation freshness checker. Analyzes recent code changes, maps them to
  the documentation hierarchy, and detects stale docs that need updating.
  Proposes and applies updates after user approval.
user-invocable: true
argument-hint: ""
---

# Kagent Docs — Documentation Freshness Checker

Detect which documentation needs updating based on recent code changes.

---

## Documentation Hierarchy

| Level | Files | Scope |
|-------|-------|-------|
| L0 | `CLAUDE.md` | Developer conventions, project structure, persistent memory |
| L1 | `docs/architecture/README.md` | System overview, core components, key decisions |
| L2 | `docs/architecture/*.md` | Detailed architecture docs |
| L3 | `docs/OIDC_PROXY_AUTH_ARCHITECTURE.md` | Auth architecture |
| L4 | Per-component READMEs (see below) | Component-specific docs |
| -- | `CHANGELOG.md` | Feature log (always check) |

### Component READMEs (L4)

| Path | Scope |
|------|-------|
| `go/README.md` | Go workspace structure, build instructions |
| `go/adk/pkg/README.md` | Go ADK package docs |
| `go/core/test/e2e/README.md` | E2E test guide |
| `python/README.md` | Python workspace, UV setup |
| `python/packages/kagent-adk/README.md` | Python ADK package |
| `python/packages/kagent-core/README.md` | Core utilities |
| `python/packages/kagent-sota-adapter/README.md` | SOTA adapter |
| `ui/README.md` | UI development guide |
| `helm/README.md` | Helm chart usage |
| `contrib/README.md` | Community contributions |

---

## Workflow

### Step 1: Analyze Code Changes

```bash
git diff --name-only $(git merge-base main HEAD)..HEAD
```

### Step 2: Map Changes to Docs

| Code Path | Affected Documentation |
|-----------|----------------------|
| `go/api/v1alpha2/` | `docs/architecture/crds-and-types.md`, `docs/architecture/README.md` |
| `go/core/internal/httpserver/` | `docs/architecture/data-flow.md`, API endpoint docs |
| `go/core/internal/controller/` | `docs/architecture/controller-reconciliation.md`, `docs/architecture/README.md` |
| `go/core/internal/controller/translator/` | `docs/architecture/controller-reconciliation.md` |
| `go/adk/` | `go/adk/pkg/README.md`, `docs/architecture/README.md` |
| `helm/` | `helm/README.md`, `CLAUDE.md` (quick reference) |
| `ui/` | `ui/README.md` |
| `python/packages/kagent-adk/` | `python/packages/kagent-adk/README.md`, `docs/architecture/README.md` |
| `python/packages/kagent-sota-adapter/` | `python/packages/kagent-sota-adapter/README.md` |
| Auth-related (`oidc`, `auth`, `credential`) | `docs/OIDC_PROXY_AUTH_ARCHITECTURE.md` |
| `go/core/test/e2e/` | `go/core/test/e2e/README.md` |
| A2A or subagent changes | `docs/architecture/a2a-subagents.md` |
| Human-in-the-loop / approval | `docs/architecture/human-in-the-loop.md` |
| Prompt template changes | `docs/architecture/prompt-templates.md` |
| New CRDs or controllers | `docs/architecture/crds-and-types.md`, potentially new architecture doc |
| `scripts/build/`, `Makefile` | `CLAUDE.md` (quick reference section) |

### Step 3: Check Each Affected Doc

For each doc identified:
1. Read the documentation file
2. Read the changed source code
3. Determine staleness:
   - **Missing:** New feature not mentioned at all
   - **Wrong:** Examples or instructions reference old behavior
   - **Incomplete:** Feature mentioned but details outdated

### Step 4: Check CHANGELOG.md

Read `CHANGELOG.md` unreleased section. Compare against actual code changes.
- New features without CHANGELOG entry -> suggest addition
- CHANGELOG entries referencing removed code -> suggest removal

### Step 5: Report & Fix

Present findings as a table:

```
| Doc | Status | Issue |
|-----|--------|-------|
| docs/architecture/crds-and-types.md | STALE | Missing ScheduledRun CRD |
| docs/architecture/README.md | OK | — |
| CHANGELOG.md | STALE | Missing SandboxAgent feature |
```

For each STALE doc:
- Show proposed changes (diff-style)
- Apply after user approval

---

## Skip Conditions

- If only `.claude/skills/**`, `docs/**`, or `*.md` files changed -> "No code changes to check against docs."
- If no mapping matches -> "Changes don't affect user-facing documentation."

---

## Quality Checks

When updating docs, ensure:
- Code examples compile / are syntactically valid
- CLI commands use current flag names
- CRD field names match `go/api/v1alpha2/` types
- Links to other docs are valid relative paths
- No references to deleted features (v1alpha1, MCPServer KMCP, CrewAI as primary)
- Architecture docs align with current controller count (10 controllers, 8 CRDs)

---

## Current CRDs (v1alpha2)

For reference when checking docs:
- Agent, SandboxAgent, AgentHarness
- ModelConfig, ModelProviderConfig, PlatformCredential
- RemoteMCPServer, ScheduledRun

## Current Controllers

- agent, sandboxagent, agentharness
- modelconfig, modelproviderconfig
- remote_mcp_server, mcp_server_tool
- scheduledrun (+ scheduler), service
