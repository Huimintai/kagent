---
name: kagent-ci
description: >
  Deploy to orc (canary) or haas (production) cluster using deploy.sh.
  Handles build, push, and deploy in one command. Use this skill for any deployment task.
user-invocable: true
argument-hint: "orc | haas"
---

# Kagent CI — Deploy via deploy.sh

One-command deployment: build -> push -> deploy.

---

## Workflow

### 1. Determine Parameters

**Cluster:** If not provided as argument, ask:
```
AskUserQuestion: "Which cluster?"
  - orc (canary — fast iteration)
  - haas (production — stable releases)
```

**Components:** If not specified, default to `all`. Valid: `controller`, `golang-adk`, `ui`, `claude-code`, `codex`, `aicore-proxy`, `all`.

**Tag:** Default to git short SHA: `git rev-parse --short HEAD`

### 2. Run deploy.sh

```bash
.local/deploy.sh <cluster> <components> --tag <tag>
```

**Important:** The script lives at `.local/deploy.sh` (not repo root). Run from the repo root directory.

### 3. Report Result

deploy.sh handles everything:
- Phase 1: Preflight (tool check, cluster connectivity, crane auth)
- Phase 2: Build (parallel per-component scripts in scripts/build/)
- Phase 3: Push (docker save + crane push to SAP registry)
- Phase 4: Deploy (kubectl set image or helm upgrade --install)
- Phase 5: Verify (rollout status, pod check)

---

## Options

| Flag | Effect |
|------|--------|
| `--skip-build` | Reuse existing local images (push + deploy only) |
| `--skip-push` | Images already in registry (deploy only) |
| `--full-install` | Helm upgrade --install instead of kubectl set image |
| `--dry-run` | Print what would happen without executing |

---

## Common Scenarios

```bash
# Full deploy all to orc
.local/deploy.sh orc all --tag $(git rev-parse --short HEAD)

# Deploy only controller + ui to haas
.local/deploy.sh haas controller ui --tag v1.0

# Redeploy without rebuilding
.local/deploy.sh orc all --tag abc123 --skip-build

# First-time install or Helm chart changes
.local/deploy.sh orc all --tag abc123 --full-install

# Preview what would happen
.local/deploy.sh orc all --tag abc123 --dry-run
```

---

## Known Issue: Script Path References

The script uses `SCRIPT_DIR` (resolves to `.local/`) for paths like `${SCRIPT_DIR}/scripts/build/`, `${SCRIPT_DIR}/helm/`, and `${SCRIPT_DIR}/go/api/config/crd/bases/`. These directories are at the repo root, not under `.local/`.

**Workaround:** The build scripts at `scripts/build/*.sh` are invoked with `bash` and source their own `env.sh`, so they work from any directory. For `--full-install` mode, the helm/CRD paths will fail unless symlinks are created:

```bash
# Create symlinks if using --full-install
ln -sf ../scripts .local/scripts
ln -sf ../helm .local/helm
ln -sf ../go .local/go
```

For incremental deploys (default mode without `--full-install`), the script works without symlinks since it only uses build scripts and kubectl commands.

---

## CRD-Only Updates

If only CRDs changed (no image rebuild needed):

```bash
kubectl apply -f go/api/config/crd/bases/ --server-side --force-conflicts \
  --kubeconfig=.local/secrets/<cluster> --context=default-cluster-admin-blue
```
