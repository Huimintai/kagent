---
name: kagent-ci-prod
description: >
  Build and deploy kagent to a remote production (amd64) cluster. Covers cross-compilation
  with BUILD_MODE=local, Docker Hub push, Helm deployment, ConfigMap-driven agent image
  updates, rollback, proxy/network troubleshooting, and buildx issues.
  Use this skill when deploying kagent to a remote production cluster.
---

# Kagent CI — Remote Production Cluster

## Quick Reference

```bash
# Set proxy + kubeconfig
export http_proxy=http://127.0.0.1:YOUR_PROXY_PORT
export https_proxy=http://127.0.0.1:YOUR_PROXY_PORT
export HTTP_PROXY=http://127.0.0.1:YOUR_PROXY_PORT
export HTTPS_PROXY=http://127.0.0.1:YOUR_PROXY_PORT
export KUBECONFIG=~/kagent-dev/kubeconfigs/haas

# Cross-compile locally + push to Docker Hub (~2 min total)
make build BUILD_MODE=local TARGETARCH=amd64 \
  DOCKER_REGISTRY=docker.io DOCKER_REPO=YOUR_DOCKERHUB_USER

# Helm install
make helm-install-provider KAGENT_DEFAULT_MODEL_PROVIDER=SAPAICore \
  DOCKER_REGISTRY=docker.io DOCKER_REPO=YOUR_DOCKERHUB_USER \
  KUBE_CONTEXT=default-cluster-admin-blue
```

## BUILD_MODE=local with Cross-Compilation

For arm64 Mac → amd64 remote cluster, use `BUILD_MODE=local TARGETARCH=amd64`. Go cross-compiles natively; JS is arch-independent.

```bash
# Build everything (cross-compile Go to amd64, push to Docker Hub)
make build BUILD_MODE=local TARGETARCH=amd64 \
  DOCKER_REGISTRY=docker.io DOCKER_REPO=YOUR_DOCKERHUB_USER

# Build single component
make build-controller BUILD_MODE=local TARGETARCH=amd64 \
  CONTROLLER_IMG=docker.io/YOUR_DOCKERHUB_USER/controller:$(git describe --tags --always --dirty)

make build-ui BUILD_MODE=local TARGETARCH=amd64 \
  UI_IMG=docker.io/YOUR_DOCKERHUB_USER/ui:$(git describe --tags --always --dirty)

# Python components — always Docker, need --platform linux/amd64
make build-kagent-adk \
  KAGENT_ADK_IMG=docker.io/YOUR_DOCKERHUB_USER/kagent-adk:latest \
  DOCKER_BUILD_ARGS="--push --platform linux/amd64 --network host"

make build-app \
  APP_IMG=docker.io/YOUR_DOCKERHUB_USER/app:latest \
  DOCKER_REGISTRY=docker.io DOCKER_REPO=YOUR_DOCKERHUB_USER \
  DOCKER_BUILD_ARGS="--push --platform linux/amd64 --network host"
```

> **Why `DOCKER_REGISTRY` and `DOCKER_REPO` for build-app?**
> `python/Dockerfile.app` uses `FROM $DOCKER_REGISTRY/$DOCKER_REPO/kagent-adk:$VERSION` — these must resolve to wherever you pushed kagent-adk.

## Deploy to Remote Cluster

### Option A: Helm Install (full)

```bash
make helm-install-provider KAGENT_DEFAULT_MODEL_PROVIDER=SAPAICore \
  DOCKER_REGISTRY=docker.io DOCKER_REPO=YOUR_DOCKERHUB_USER \
  KUBE_CONTEXT=default-cluster-admin-blue
```

### Option B: Incremental Update

```bash
export KUBECONFIG=~/kagent-dev/kubeconfigs/haas

# 1. Apply CRD (server-side apply required — too large for client-side)
kubectl apply -f go/api/config/crd/bases/kagent.dev_agents.yaml \
  --server-side --force-conflicts

# 2. Update controller image
kubectl set image deployment/kagent-controller -n kagent \
  controller=docker.io/YOUR_DOCKERHUB_USER/controller:<tag>

# 3. Restart UI (same tag → rollout restart to pull new digest)
kubectl rollout restart deployment/kagent-ui -n kagent

# 4. Update agent app image via ConfigMap
kubectl patch configmap kagent-controller -n kagent --type merge -p '{
  "data": {
    "IMAGE_REGISTRY": "docker.io",
    "IMAGE_REPOSITORY": "YOUR_DOCKERHUB_USER/app",
    "IMAGE_TAG": "<tag>",
    "IMAGE_PULL_POLICY": "Always"
  }
}'

# 5. Restart controller to pick up new ConfigMap + reconcile agent deployments
kubectl rollout restart deployment/kagent-controller -n kagent
```

## Image Flow Architecture

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

## ConfigMap: kagent-controller

The controller reads these env vars from ConfigMap `kagent-controller` (via `envFrom`):

| Key | Description | Example |
|-----|-------------|---------|
| `IMAGE_REGISTRY` | Registry for agent app pods | `docker.io` |
| `IMAGE_REPOSITORY` | Repo path for agent app image | `YOUR_DOCKERHUB_USER/app` |
| `IMAGE_TAG` | Tag for agent app image | `latest` |
| `IMAGE_PULL_POLICY` | Pull policy for agent pods | `Always` or `IfNotPresent` |
| `SKILLS_INIT_IMAGE_REGISTRY` | Registry for skills-init container | `cr.kagent.dev` |
| `SKILLS_INIT_IMAGE_REPOSITORY` | Repo for skills-init | `kagent-dev/kagent/skills-init` |
| `SKILLS_INIT_IMAGE_TAG` | Tag for skills-init | `0.8.3` |

**After patching ConfigMap, always restart the controller** — env vars from `envFrom` are injected at pod start, not live-reloaded.

## Runtime Feature Flags (via ConfigMap)

UI feature flags are configured at runtime via the `kagent-ui-config` ConfigMap, **not** baked into the Docker image. Change with pod restart, no rebuild needed.

| Helm Value | ConfigMap Key | Default | Effect |
|-----------|--------------|---------|--------|
| `ui.featureFlags.allowedNamespace` | `KAGENT_ALLOWED_NAMESPACE` | `""` (all) | Lock UI to one namespace |
| `ui.featureFlags.disableModelCreation` | `KAGENT_DISABLE_MODEL_CREATION` | `"false"` | Hide New Model button |
| `ui.featureFlags.disableMcpServerCreation` | `KAGENT_DISABLE_MCP_SERVER_CREATION` | `"true"` | Disable MCP server creation |
| `ui.featureFlags.disableByoAgentCreation` | `KAGENT_DISABLE_BYO_AGENT_CREATION` | `"true"` | Disable BYO agent type |
| `ui.featureFlags.protectedAgents` | `KAGENT_PROTECTED_AGENTS` | `""` | Comma-separated protected agent names |

```bash
# Change feature flags without rebuild
helm upgrade kagent helm/kagent -n kagent --reuse-values \
  --set ui.featureFlags.allowedNamespace=dbci-agent \
  --set ui.featureFlags.disableModelCreation=true
# Pod auto-restarts (checksum annotation triggers rollout)
```

## Rollback

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

## Proxy & Network Troubleshooting

### Host-level proxy (for docker push, kubectl, helm)

```bash
export http_proxy=http://127.0.0.1:YOUR_PROXY_PORT
export https_proxy=http://127.0.0.1:YOUR_PROXY_PORT
export HTTP_PROXY=http://127.0.0.1:YOUR_PROXY_PORT
export HTTPS_PROXY=http://127.0.0.1:YOUR_PROXY_PORT
```

### Buildx `docker-container` driver and proxy

The buildkit driver runs in its own container and does **NOT** inherit host proxy env vars. This affects:
- **Metadata resolve** (`FROM` image pulls) — runs inside the buildkit daemon
- **Build-stage network requests** (`go mod download`, `uv sync`, `npm install`)
- **Push to registry**

**What works:**

1. **Always pass `--network host`** in `DOCKER_BUILD_ARGS` so build-stage commands reach the host proxy:
   ```bash
   DOCKER_BUILD_ARGS="--push --platform linux/amd64 --network host"
   ```

2. **For metadata resolve failures (EOF from cgr.dev, gcr.io):** Often intermittent. Retry — buildkit caches pulled layers across attempts, so retries get faster.

3. **For push failures (EOF from docker.io auth):** Ensure `docker login` is current, then retry.

### Rebuilding the buildx builder

If the builder gets into a bad state:

```bash
docker buildx rm kagent-builder-v0.23.0
docker buildx create \
  --name kagent-builder-v0.23.0 \
  --platform linux/amd64,linux/arm64 \
  --driver docker-container \
  --driver-opt network=host \
  --use
```

### Base Image Notes

Skills-init and Python components now use `alpine:3.23` instead of Chainguard images, which significantly reduces build times (faster pulls, smaller layers, better cache hits). Go components still use `gcr.io/distroless/static:nonroot`. UI uses Chainguard Wolfi (`cgr.dev/chainguard/wolfi-base`).

`cgr.dev/chainguard/wolfi-base` (used by UI) can return intermittent EOF during metadata resolve. Not a proxy issue — happens even with direct access.

**Mitigation:** Retry. Layer caching means retries are cheap.

## Common Pitfalls

### 1. Platform mismatch (`no match for platform`)

**Cause:** Mac (arm64) built image without `--platform linux/amd64`.

**Fix:** Always use `TARGETARCH=amd64` for BUILD_MODE=local, or `DOCKER_BUILD_ARGS="--push --platform linux/amd64"` for Docker builds.

### 2. CRD too large for `kubectl apply`

**Cause:** `agents.kagent.dev` CRD exceeds 262144-byte annotation limit.

**Fix:** Server-side apply:
```bash
kubectl apply -f go/api/config/crd/bases/kagent.dev_agents.yaml \
  --server-side --force-conflicts
```

### 3. Agent pods don't update after ConfigMap change

**Cause:** Controller reads ConfigMap via `envFrom` (env vars set at pod start only).

**Fix:** After patching ConfigMap, restart controller:
```bash
kubectl rollout restart deployment/kagent-controller -n kagent
```

### 4. `build-app` can't find kagent-adk base image

**Cause:** `Dockerfile.app` uses `FROM $DOCKER_REGISTRY/$DOCKER_REPO/kagent-adk:$VERSION`.

**Fix:** Set `DOCKER_REGISTRY=docker.io` and `DOCKER_REPO=YOUR_DOCKERHUB_USER` (flat):
```bash
make build-app APP_IMG=YOUR_DOCKERHUB_USER/app:latest \
  DOCKER_REGISTRY=docker.io DOCKER_REPO=YOUR_DOCKERHUB_USER
```

### 5. Controller reconciles agent deployments back to old image

**Cause:** `kubectl set image` on agent deployments is overwritten on next reconciliation.

**Fix:** Always update agent images via ConfigMap, not `kubectl set image`.

### 6. `check-api-key` blocks `helm-install` for SAPAICore

**Fix:** Use `KAGENT_DEFAULT_MODEL_PROVIDER=SAPAICore` to skip the API key check.

## Makefile Variables

| Variable | Default | Override for Prod |
|----------|---------|-------------------|
| `BUILD_MODE` | `docker` | `local` (cross-compile on host) |
| `TARGETARCH` | `$(LOCALARCH)` (arm64) | `amd64` |
| `DOCKER_REGISTRY` | `localhost:5001` | `docker.io` |
| `DOCKER_REPO` | `kagent-dev/kagent` | `YOUR_DOCKERHUB_USER` |
| `DOCKER_BUILD_ARGS` | `--push --platform linux/$(LOCALARCH)` | `--push --platform linux/amd64 --network host` |
| `KUBE_CONTEXT` | `kind-$(KIND_CLUSTER_NAME)` | `default-cluster-admin-blue` |
| `CONTROLLER_IMG` | composed from registry/repo | Full image override |
| `UI_IMG` | composed from registry/repo | Full image override |
| `APP_IMG` | composed from registry/repo | Full image override |

## Cluster Info

| Resource | Value |
|----------|-------|
| Kubeconfig | `~/kagent-dev/kubeconfigs/haas` |
| Context | `default-cluster-admin-blue` |
| Platform | `linux/amd64` |
| Registry | `docker.io` |
| Docker Hub user | `YOUR_DOCKERHUB_USER` |
| Proxy | `YOUR_PROXY_HOST:YOUR_PROXY_PORT` |
