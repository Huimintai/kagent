---
name: kagent-ci
description: >
  Build and deploy kagent components (controller, UI, app) to remote Kubernetes clusters.
  Covers image building with cross-platform support, UI build-time restrictions, Helm deployment,
  ConfigMap-driven agent image updates, rollback procedures, and common pitfalls.
  Use this skill when deploying kagent to non-local clusters (e.g. HaaS), building images for
  specific registries, or troubleshooting deployment failures.
---

# Kagent CI / Deploy Guide

## Quick Reference

### Build Individual Components

All build commands default to the local architecture. **For remote amd64 clusters (e.g. HaaS), always override `DOCKER_BUILD_ARGS`:**

```bash
# Controller
make build-controller \
  CONTROLLER_IMG=<registry>/kagent-controller:<tag> \
  DOCKER_BUILD_ARGS="--push --platform linux/amd64"

# UI (with restrictions)
make build-ui \
  UI_IMG=<registry>/kagent-ui:<tag> \
  DOCKER_BUILD_ARGS="--push --platform linux/amd64 --network host \
    --build-arg NEXT_PUBLIC_ALLOWED_NAMESPACE=<ns> \
    --build-arg NEXT_PUBLIC_DISABLE_MODEL_CREATION=true"

# App (requires kagent-adk base first)
make build-kagent-adk \
  KAGENT_ADK_IMG=<registry>/kagent-adk:<tag> \
  DOCKER_BUILD_ARGS="--push --platform linux/amd64"

make build-app \
  APP_IMG=<registry>/kagent-app:<tag> \
  DOCKER_REGISTRY=docker.io DOCKER_REPO=<registry-user> \
  DOCKER_BUILD_ARGS="--push --platform linux/amd64"
```

> **Why `DOCKER_REGISTRY` and `DOCKER_REPO` for build-app?**
> `python/Dockerfile.app` uses `FROM $DOCKER_REGISTRY/$DOCKER_REPO/kagent-adk:$KAGENT_ADK_VERSION` — these must resolve to wherever you pushed the kagent-adk image.

### Deploy to a Remote Cluster

```bash
export KUBECONFIG=~/kagent-dev/kubeconfigs/haas

# 1. Apply CRD (server-side apply required — CRD is too large for client-side annotation)
kubectl apply -f go/api/config/crd/bases/kagent.dev_agents.yaml --server-side --force-conflicts

# 2. Update controller image
kubectl set image deployment/kagent-controller -n kagent controller=<registry>/kagent-controller:<tag>

# 3. Restart UI (same tag = rollout restart to pull new digest)
kubectl rollout restart deployment/kagent-ui -n kagent

# 4. Update agent app image via ConfigMap (controller reads on startup)
kubectl patch configmap kagent-controller -n kagent --type merge -p '{
  "data": {
    "IMAGE_REGISTRY": "docker.io",
    "IMAGE_REPOSITORY": "<registry>/kagent-app",
    "IMAGE_TAG": "<tag>",
    "IMAGE_PULL_POLICY": "Always"
  }
}'

# 5. Restart controller to pick up new ConfigMap + reconcile agent deployments
kubectl rollout restart deployment/kagent-controller -n kagent
```

### Rollback

```bash
export KUBECONFIG=~/kagent-dev/kubeconfigs/haas

# Controller
kubectl set image deployment/kagent-controller -n kagent controller=<old-image>

# UI
kubectl rollout undo deployment/kagent-ui -n kagent

# Agent app image (via ConfigMap + controller restart)
kubectl patch configmap kagent-controller -n kagent --type merge -p '{
  "data": {
    "IMAGE_REGISTRY": "docker.io",
    "IMAGE_REPOSITORY": "<old-repo>",
    "IMAGE_TAG": "<old-tag>",
    "IMAGE_PULL_POLICY": "IfNotPresent"
  }
}'
kubectl rollout restart deployment/kagent-controller -n kagent
```

---

## Architecture: How Images Flow

```
┌──────────────┐     kubectl set image      ┌─────────────────────┐
│  Controller  │ ◄────────────────────────── │  Direct deployment  │
│  Image       │                             │  update             │
└──────────────┘                             └─────────────────────┘

┌──────────────┐     kubectl rollout restart ┌─────────────────────┐
│  UI Image    │ ◄────────────────────────── │  Direct deployment  │
│              │                             │  update             │
└──────────────┘                             └─────────────────────┘

┌──────────────┐     ConfigMap               ┌─────────────────────┐
│  Agent App   │ ◄── kagent-controller ◄──── │  IMAGE_REGISTRY     │
│  Image       │     reconciles agent        │  IMAGE_REPOSITORY   │
│  (per agent) │     deployments on          │  IMAGE_TAG          │
└──────────────┘     CR change               └─────────────────────┘
```

**Key insight:** Agent pod images are NOT set directly on deployments. The controller reads `IMAGE_*` from ConfigMap `kagent-controller` and sets them during Agent CR reconciliation. You must:
1. Patch the ConfigMap
2. Restart the controller
3. Wait for reconciliation (or annotate Agent CRs to trigger it)

---

## Makefile Variables

| Variable | Default | Override for remote deploy |
|----------|---------|---------------------------|
| `DOCKER_REGISTRY` | `localhost:5001` | `docker.io` |
| `DOCKER_REPO` | `kagent-dev/kagent` | Docker Hub username (e.g. `guswong`) |
| `DOCKER_BUILD_ARGS` | `--push --platform linux/$(LOCALARCH)` | `--push --platform linux/amd64` |
| `KUBE_CONTEXT` | `kind-$(KIND_CLUSTER_NAME)` | Remote cluster context name |
| `CONTROLLER_IMG` | `$(DOCKER_REGISTRY)/$(DOCKER_REPO)/controller:$(VERSION)` | Full image override |
| `UI_IMG` | `$(DOCKER_REGISTRY)/$(DOCKER_REPO)/ui:$(VERSION)` | Full image override |
| `APP_IMG` | `$(DOCKER_REGISTRY)/$(DOCKER_REPO)/app:$(VERSION)` | Full image override |
| `KAGENT_ADK_IMG` | `$(DOCKER_REGISTRY)/$(DOCKER_REPO)/kagent-adk:$(VERSION)` | Full image override |

**Tip:** Override the full `*_IMG` variable to avoid composing from parts.

---

## UI Build-Time Restrictions

These are `--build-arg` values baked into the Next.js static build. **Cannot be changed at runtime.**

| Build Arg | Default | Effect |
|-----------|---------|--------|
| `NEXT_PUBLIC_ALLOWED_NAMESPACE` | empty (all namespaces) | Lock UI to a single namespace (e.g. `dbci-agent`) |
| `NEXT_PUBLIC_DISABLE_MODEL_CREATION` | `""` (enabled) | `true` to hide New Model button + show disabled banner |
| `NEXT_PUBLIC_DISABLE_MCP_SERVER_CREATION` | `""` (disabled) | Set to `false` to **enable** MCP server creation. Disabled by default. |
| `NEXT_PUBLIC_DISABLE_BYO_AGENT_CREATION` | `""` (disabled) | Set to `false` to **enable** BYO agent type. Disabled by default. |
| `NEXT_PUBLIC_PROTECTED_AGENTS` | empty | Comma-separated agent names that cannot be edited/deleted |

> **Note on defaults:** `DISABLE_MCP_SERVER_CREATION` and `DISABLE_BYO_AGENT_CREATION` are **disabled by default** (not set = disabled). Pass `=false` to enable. This is the opposite of `DISABLE_MODEL_CREATION` which is enabled by default (not set = enabled).

Pass via `DOCKER_BUILD_ARGS`:
```bash
DOCKER_BUILD_ARGS="--push --platform linux/amd64 \
  --build-arg NEXT_PUBLIC_ALLOWED_NAMESPACE=dbci-agent \
  --build-arg NEXT_PUBLIC_DISABLE_MODEL_CREATION=true"
```

To enable MCP server or BYO agent creation:
```bash
DOCKER_BUILD_ARGS="... \
  --build-arg NEXT_PUBLIC_DISABLE_MCP_SERVER_CREATION=false \
  --build-arg NEXT_PUBLIC_DISABLE_BYO_AGENT_CREATION=false"
```

---

## ConfigMap: kagent-controller

The controller reads these env vars from ConfigMap `kagent-controller` (via `envFrom`):

| Key | Description | Example |
|-----|-------------|---------|
| `IMAGE_REGISTRY` | Registry for agent app pods | `docker.io` |
| `IMAGE_REPOSITORY` | Repo path for agent app image | `guswong/kagent-app` |
| `IMAGE_TAG` | Tag for agent app image | `latest` |
| `IMAGE_PULL_POLICY` | Pull policy for agent pods | `Always` or `IfNotPresent` |
| `SKILLS_INIT_IMAGE_REGISTRY` | Registry for skills-init container | `cr.kagent.dev` |
| `SKILLS_INIT_IMAGE_REPOSITORY` | Repo for skills-init | `kagent-dev/kagent/skills-init` |
| `SKILLS_INIT_IMAGE_TAG` | Tag for skills-init | `0.8.3` |

**After patching ConfigMap, always restart the controller** — env vars from `envFrom` are injected at pod start, not live-reloaded.

---

## Common Pitfalls

### 1. Platform mismatch (`ImagePullBackOff: no match for platform`)

**Cause:** Mac (arm64) built image with default `--platform linux/arm64`, remote cluster runs amd64.

**Fix:** Always pass `DOCKER_BUILD_ARGS="--push --platform linux/amd64"` when targeting remote clusters.

### 2. CRD too large for `kubectl apply`

**Cause:** `agents.kagent.dev` CRD exceeds 262144-byte annotation limit for client-side apply.

**Fix:** Use server-side apply:
```bash
kubectl apply -f go/api/config/crd/bases/kagent.dev_agents.yaml --server-side --force-conflicts
```

### 3. Agent pods don't update after ConfigMap change

**Cause:** Controller reads ConfigMap via `envFrom` (env vars set at pod start). ConfigMap change alone doesn't restart controller.

**Fix:** After patching ConfigMap, restart controller:
```bash
kubectl rollout restart deployment/kagent-controller -n kagent
```

Then either wait for natural reconciliation or annotate Agent CRs to trigger:
```bash
kubectl annotate agents.kagent.dev <name> -n <ns> --overwrite reconcile-trigger="$(date +%s)"
```

### 4. `build-app` can't find kagent-adk base image

**Cause:** `Dockerfile.app` constructs `FROM $DOCKER_REGISTRY/$DOCKER_REPO/kagent-adk:$VERSION`. If `DOCKER_REGISTRY=docker.io/guswong`, it becomes `docker.io/guswong/kagent-dev/kagent/kagent-adk` — wrong path.

**Fix:** Set `DOCKER_REGISTRY=docker.io` and `DOCKER_REPO=<user>` (not `<user>/kagent-dev/kagent`):
```bash
make build-app APP_IMG=guswong/kagent-app:latest DOCKER_REGISTRY=docker.io DOCKER_REPO=guswong
```

### 5. Controller reconciles agent deployments back to old image

**Cause:** `kubectl set image` on agent deployments is overwritten on next reconciliation because the controller uses ConfigMap values.

**Fix:** Always update agent images via ConfigMap, not `kubectl set image`.

---

## Known Clusters

| Cluster | Kubeconfig | Context |
|---------|-----------|---------|
| HaaS | `~/kagent-dev/kubeconfigs/haas` | `default-cluster-admin-blue` |
| Local Kind | `~/.kube/config` | `kind-kagent` |

---

## Proxy for Docker Hub

If Docker Hub is unreachable (connection reset), set proxy before buildx:

```bash
export http_proxy=http://127.0.0.1:<port>
export https_proxy=http://127.0.0.1:<port>
export HTTP_PROXY=http://127.0.0.1:<port>
export HTTPS_PROXY=http://127.0.0.1:<port>
```

Note: `docker buildx` with `docker-container` driver does NOT inherit host env proxy. For push-only issues (build succeeds, push fails), host-level proxy is sufficient.
