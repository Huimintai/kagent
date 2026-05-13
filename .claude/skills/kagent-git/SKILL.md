---
name: kagent-git
description: >
  Reorganize and squash messy commits on the current branch into clean, atomic commits.
  First split by feature, then within each feature split by scope layer (ui, controller,
  agent runtime, sap adapter). Use this skill when the user asks to tidy up commits,
  reorganize git history, or prepare a branch for PR review.
---

# Kagent Git — Atomic Commit Reorganization

Reorganize all commits on the current branch (since diverging from main) into clean,
atomic commits. The key principle: **Feature-first, then Layer-split**.

## Splitting Strategy

### Dimension 1: Feature (primary axis)

First, identify distinct **features** or **logical units of work** across all commits.
A feature is a user-facing capability or a cohesive infrastructure change.

Examples of features:
- "Scheduled Runs" (cron-based agent execution)
- "Agent Ownership" (multi-user with ownership tracking)
- "Memory System" (semantic memory with vector search)
- "MCP OAuth" (per-user OAuth token management for MCP servers)
- "Inline Skills" (OCI/Git skill packaging)

### Dimension 2: Scope Layer (secondary axis, within each feature)

Within each feature, split commits by **scope layer**:

| Layer | Scope Tag | What It Covers |
|-------|-----------|----------------|
| **API / CRD** | `api` | CRD types (`go/api/v1alpha2/`), ADK types (`go/api/adk/`), deepcopy, generated CRD YAML, Helm CRD templates |
| **Controller** | `controller` | Translator, reconciler, HTTP handlers, database queries/migrations, CLI (`go/core/`) |
| **Agent Runtime** | `runtime` | Python ADK (`python/packages/`), executor, converters, tools, session/memory services |
| **UI** | `ui` | Next.js frontend (`ui/`), components, pages, API clients |
| **SAP Adapter** | `sap` | SAP AI Core model provider, orchestration, GDCH token exchange — anything SAP-specific |
| **Helm / Infra** | `helm` | Helm chart templates, values, RBAC (feature-related, not pure packaging) |

### Special Group: Build & CI (always ONE single commit, at the end)

All build/CI/tooling/skill changes go into **one commit** at the bottom of the stack.

**Included:**
- `Makefile`, `**/Makefile`
- `**/Dockerfile*`
- `.github/**`, `.claude/skills/**`
- `docker/**`, `.goreleaser*`, `.dockerignore`
- `helm/**/Chart.yaml`, `helm/**/values.yaml` (packaging only)
- Changelog, architecture docs that don't belong to a specific feature

**Commit message:** `chore: update build, CI, and tooling`

## Commit Message Format

```
<type>(<feature>/<layer>): <description>
```

Or if the feature IS the layer scope:
```
<type>(<layer>): <description>
```

**Examples:**
```
feat(scheduled-runs/api): add ScheduledRun CRD types and validation
feat(scheduled-runs/controller): add scheduler reconciler and HTTP endpoints
feat(scheduled-runs/ui): add scheduled runs management page
feat(ownership/api): add owner field to Agent CRD
feat(ownership/controller): add ownership filtering to agent handlers
feat(ownership/ui): add user profile and agent ownership display
feat(mcp-oauth/runtime): add MCP OAuth token tools and session storage
feat(sap): add SAP AI Core model provider with orchestration service
chore: update build, CI, and tooling
```

## Execution Procedure

### Step 1: Analyze

```bash
git merge-base main HEAD
git diff --name-only $(git merge-base main HEAD)..HEAD
git log --oneline $(git merge-base main HEAD)..HEAD
```

Study the full diff. Identify distinct features by reading commit messages AND file changes.

### Step 2: Plan — Feature × Layer Matrix

Present a matrix to the user:

| # | Feature | Layer | Key Files | Proposed Message |
|---|---------|-------|-----------|-----------------|
| 1 | scheduled-runs | api | agent_types.go, ... | `feat(scheduled-runs/api): ...` |
| 2 | scheduled-runs | controller | handlers/, queries/ | `feat(scheduled-runs/controller): ...` |
| 3 | scheduled-runs | ui | ScheduledRuns.tsx | `feat(scheduled-runs/ui): ...` |
| 4 | ownership | api | ... | `feat(ownership/api): ...` |
| ... | | | | |
| N | — | build/ci | Makefile, Dockerfile | `chore: ...` |

**Rules for the matrix:**
- A feature that only touches one layer gets one commit (no need to split further)
- A feature touching 2+ layers gets one commit per layer
- Files that serve multiple features: ask the user or assign to the primary feature
- Generated files go with the commit that caused generation
- If a layer has <5 lines of trivial change for a feature, merge into adjacent layer commit

**Wait for user approval before proceeding.**

### Step 3: Execute

```bash
ORIGINAL_HEAD=$(git rev-parse HEAD)
BASE=$(git merge-base main HEAD)

# Soft reset to merge-base — all changes become staged
git reset --soft $BASE

# Unstage everything
git reset HEAD .

# Commit each group (bottom of stack first → build/ci last)
# Feature A / Layer 1
git add <files...>
git commit -s -m "<type>(<feature>/<layer>): <description>"

# Feature A / Layer 2
git add <files...>
git commit -s -m "<type>(<feature>/<layer>): <description>"

# ... repeat for all groups ...

# Last: Build & CI
git add <remaining build files...>
git commit -s -m "chore: update build, CI, and tooling"
```

### Step 4: Verify

```bash
# Must be empty — no code lost
git diff $ORIGINAL_HEAD HEAD

# Clean tree
git status

# New log
git log --oneline $BASE..HEAD
```

If diff is non-empty, investigate and fix before proceeding.

## Ordering of Commits

Commits should be ordered for reviewability:

1. API / CRD changes first (types that other layers depend on)
2. Controller changes (implements the API)
3. Runtime changes (Python ADK consuming the config)
4. UI changes (consumes the HTTP API)
5. SAP adapter (specialized provider)
6. Helm / Infra (deployment of the above)
7. Build & CI last (tooling, always at top of log)

Within the same layer, order features by size (largest first).

## Important Notes

- **NEVER force-push without user confirmation.**
- **Preserve all changes.** `git diff $ORIGINAL_HEAD HEAD` must be empty.
- **Sign commits** with `-s` flag.
- **When in doubt, ask.** If a file could belong to multiple features, ask the user.
- **Untracked build artifacts** should NOT be committed. Mention them to the user.
- **Generated files** go with the commit that caused generation.
- **Helm CRD templates** that mirror CRD YAML go with the feature/api commit.
