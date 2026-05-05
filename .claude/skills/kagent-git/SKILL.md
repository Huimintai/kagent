---
name: kagent-git
description: >
  Reorganize and squash messy commits on the current branch into clean, logical commits.
  Use this skill when the user asks to tidy up commits, reorganize git history,
  or prepare a branch for PR review by grouping changes by category.
---

# Kagent Git — Commit Reorganization

Reorganize all commits on the current branch (since diverging from main) into clean,
well-structured commits grouped by category. This makes PR reviews easier and keeps
history meaningful.

## Commit Grouping Rules

### Group 1: Build & CI (always ONE single commit)

All build, CI, containerization, and skill-related changes go into **one commit**.

**Included file patterns:**
- `Makefile`, `**/Makefile`
- `**/Dockerfile*` (Dockerfile, Dockerfile.local, etc.)
- `.github/**` (GitHub Actions workflows, CI configs)
- `.claude/skills/**` (Claude Code skill definitions)
- `docker/**` (Docker build contexts, scripts)
- `.goreleaser*`, `.dockerignore`
- `helm/**/Chart.yaml`, `helm/**/values.yaml` (Helm packaging metadata, NOT template changes that reflect CRD/feature work)

**Commit message:** `chore: update build, CI, and skill configurations`
(Adjust the message to accurately reflect what changed.)

### Group 2+: Feature / Module Commits (split by logical unit)

All remaining changes are split into separate commits by **module + functionality**.
Each commit should be a coherent, self-contained unit of work.

**Splitting heuristics (in priority order):**

1. **By feature/functionality** — If multiple files serve one feature, they belong together.
   - Example: A new CRD field spans `agent_types.go`, `zz_generated.deepcopy.go`,
     `agents.yaml` (CRD base), Helm CRD template, translator, and tests — all one commit.

2. **By module boundary** — When changes are independent across modules, split them.
   - `go/api/` — API types, CRDs, database models
   - `go/core/` — Controllers, HTTP server, CLI
   - `go/adk/` — Go Agent Development Kit
   - `python/` — Python ADK and agent runtime
   - `ui/` — Next.js frontend
   - `helm/` — Helm chart templates (feature-related, not pure packaging)

3. **By logical concern** — Within a module, further split if changes are unrelated.
   - Example: In `ui/`, auth-related components vs. agent-list UI vs. config/feature-flags
     should be separate commits.

**Commit message format:** Follow conventional commits:
```
<type>(<scope>): <description>
```
Examples:
- `feat(api): add inline skill and CLI tool container types`
- `feat(ui): add GitHub OAuth and user profile components`
- `fix(controller): handle nil pointer in reconciler utils`
- `feat(python): add SAP AI Core model provider`

## Execution Procedure

Follow these steps **exactly**:

### Step 1: Analyze

```bash
# Identify the base branch and diverge point
git merge-base main HEAD

# List all changed files
git diff --name-only $(git merge-base main HEAD)..HEAD

# View the full diff for context
git diff $(git merge-base main HEAD)..HEAD
```

Study the diff carefully. Understand what each changed file does before grouping.

### Step 2: Plan the Commits

Create a grouping plan and present it to the user as a table:

| # | Type | Scope | Files | Proposed message |
|---|------|-------|-------|-----------------|
| 1 | chore | build/ci | Makefile, Dockerfile.local, ... | `chore: ...` |
| 2 | feat | api | agent_types.go, ... | `feat(api): ...` |
| 3 | feat | ui | AgentList.tsx, ... | `feat(ui): ...` |
| ... | | | | |

**Wait for user approval before proceeding.**

### Step 3: Execute the Rewrite

Use interactive rebase with careful reset and re-staging:

```bash
# Save current HEAD
ORIGINAL_HEAD=$(git rev-parse HEAD)
BASE=$(git merge-base main HEAD)

# Soft reset to the merge-base — all changes become staged
git reset --soft $BASE

# Unstage everything
git reset HEAD .

# Now selectively stage and commit each group:

# Group 1: Build & CI
git add Makefile docker/ .claude/skills/ go/Dockerfile.local ui/Dockerfile.local ...
git commit -m "chore: update build, CI, and skill configurations"

# Group 2: Feature A
git add go/api/v1alpha2/agent_types.go go/api/v1alpha2/zz_generated.deepcopy.go ...
git commit -m "feat(api): add inline skill and CLI tool container types"

# ... repeat for each group
```

### Step 4: Verify

```bash
# Confirm no changes are lost
git diff $ORIGINAL_HEAD HEAD  # Should be empty

# Confirm clean working tree
git status

# Show the new commit log
git log --oneline $BASE..HEAD
```

If `git diff $ORIGINAL_HEAD HEAD` shows any output, something was missed — investigate and fix.

## Important Notes

- **NEVER force-push without user confirmation.** After rewriting, remind the user
  that `git push --force-with-lease` is needed if the branch was already pushed.
- **Preserve all changes.** The diff between the old HEAD and new HEAD must be empty.
  No code should be added or removed during reorganization.
- **When in doubt, ask.** If a file could belong to multiple groups, ask the user.
- **Untracked files** in `docker/skills-init/bin/` and similar build artifacts should
  NOT be committed. Mention them to the user if present.
- **Generated files** (like `zz_generated.deepcopy.go`, CRD YAML bases) go with
  the commit that caused the generation, not in the build/CI commit.
- **Helm CRD templates** that mirror generated CRD bases go with the feature commit,
  not the build commit. Only `Chart.yaml` / `values.yaml` packaging changes go in build/CI.
