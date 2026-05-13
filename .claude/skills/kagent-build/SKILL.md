---
name: kagent-build
description: >
  Incremental build. Analyzes git diff to determine which components changed,
  maps file paths to build targets, and runs per-component scripts in parallel.
  Use this skill when you want to rebuild only what changed.
user-invocable: true
argument-hint: "optional: component names to force-build (controller golang-adk ui claude-code codex aicore-proxy)"
---

# Kagent Build — Incremental Parallel Build

Analyze git diff -> determine affected components -> run per-component build scripts **in parallel**.

---

## Components

| Component | Script | Image |
|-----------|--------|-------|
| `controller` | `scripts/build/controller.sh` | `kagent-controller` |
| `golang-adk` | `scripts/build/golang-adk.sh` | `golang-adk` |
| `golang-adk-full` | `scripts/build/golang-adk-full.sh` | `golang-adk-full` |
| `claude-code` | `scripts/build/claude-code.sh` | `golang-adk-claude-code` |
| `codex` | `scripts/build/codex.sh` | `golang-adk-codex` |
| `ui` | `scripts/build/ui.sh` | `kagent-ui` |
| `aicore-proxy` | `scripts/build/aicore-proxy.sh` | `aicore-proxy` |
| `skills-init` | `scripts/build/skills-init.sh` | `kagent-skills-init` |

---

## Path-to-Target Mapping

| Path Pattern | Build Target(s) |
|-------------|----------------|
| `go/core/**`, `go/api/**` | `controller` |
| `go/adk/**` (not cmd/cli, cmd/codex) | `golang-adk` |
| `go/adk/cmd/cli/**` | `claude-code` |
| `go/adk/cmd/codex/**` | `codex` |
| `go/adk/pkg/**` | `golang-adk claude-code codex` (shared code) |
| `ui/**` | `ui` |
| `helm/**`, `.claude/**`, `docs/**`, `*.md` | (no build) |

---

## Execution

### Step 1: Determine targets

If user passed component names as argument, use those directly (skip git analysis).

Otherwise, analyze changed files:

```bash
git diff --name-only $(git merge-base main HEAD)..HEAD
```

Map each path to targets using the table above. Collect unique set.

### Step 2: Report plan

```
Changed files: 12
Affected targets: controller, claude-code
```

### Step 3: Build in parallel

Run each target's build script as a **parallel background process** using the Agent tool with multiple Bash calls, or via shell backgrounding:

```bash
# Set shared env
export TAG=$(git rev-parse --short HEAD)
export PUSH=false

# Launch all targets in parallel
pids=()
for target in controller claude-code; do
    ./scripts/build/${target}.sh > /tmp/kagent-build-${target}.log 2>&1 &
    pids+=($!)
done

# Wait and check
failed=()
for i in "${!pids[@]}"; do
    if ! wait "${pids[$i]}"; then
        failed+=("${targets[$i]}")
    fi
done
```

Alternatively, if using the Agent tool, spawn **one Bash call per target** in parallel:
```
Bash: TAG=abc123 PUSH=false ./scripts/build/controller.sh
Bash: TAG=abc123 PUSH=false ./scripts/build/claude-code.sh
```

### Step 4: Report results

```bash
docker images --filter "reference=hanaservice-dev.common.repositories.cloud.sap/kagent/*:${TAG}" \
    --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}"
```

---

## Special Cases

- **Only helm/docs/skills changes** -> "No build needed."
- **go/api changes** -> always triggers `controller`. Does NOT trigger `golang-adk` unless `go/adk/**` also changed.
- **go/adk/pkg changes** -> triggers ALL Go ADK targets: `golang-adk`, `claude-code`, `codex`.

---

## Environment Variables

| Var | Default | Description |
|-----|---------|-------------|
| `TAG` | git short SHA | Image tag |
| `PUSH` | `true` | Push to registry after build |
| `PROXY` | (none) | HTTP proxy for Docker builds |
| `REGISTRY` | `hanaservice-dev.common.repositories.cloud.sap/kagent` | Image registry |
