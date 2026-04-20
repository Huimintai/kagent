---
name: kagent-ci-kind
description: >
  Build and deploy kagent to a local Kind cluster. Covers fast local builds (BUILD_MODE=local),
  Helm installation with runtime ConfigMap feature flags, port-forwarding, and debugging.
  Use this skill when deploying kagent locally for development or testing.
---

# Kagent CI — Local Kind Cluster

## Quick Reference

```bash
# Full rebuild + deploy (fast, ~2 min with BUILD_MODE=local)
make create-kind-cluster           # Only if cluster doesn't exist
make build BUILD_MODE=local        # Compile locally, thin Docker images
make helm-install-provider         # Helm deploy (no image rebuild)

# Build only (no Helm apply, won't recreate default agents)
make build BUILD_MODE=local
kubectl rollout restart deployment/kagent-controller deployment/kagent-ui -n kagent --context kind-kagent

# Rebuild single component + restart
make build-controller BUILD_MODE=local
kubectl rollout restart deployment/kagent-controller -n kagent --context kind-kagent

# SAPAICore setup (after helm-install, one-time)
# See "SAPAICore Configuration" section below

# Port-forward
kubectl port-forward -n kagent --context kind-kagent svc/kagent-ui 8082:8080 &
kubectl port-forward -n kagent --context kind-kagent svc/kagent-controller 8083:8083 &
```

## BUILD_MODE=local (Recommended)

Compiles Go/UI on the host machine, then copies artifacts into thin Docker images. **20-25x faster than Docker-in-Docker builds for Go components.**

| Component | Docker mode | Local mode | Speedup |
|-----------|-------------|------------|---------|
| Controller (Go) | 5-8 min | **13s** | ~25x |
| Go ADK (Go) | 5-8 min | **19s** | ~20x |
| Skills-init (Go) | 3-5 min | **2:30** (first run) | ~2x |
| UI (Next.js) | 3-4 min | **50s** (cached) | ~4x |
| kagent-adk (Python) | 5-8 min | N/A (always Docker) | — |
| App (Python) | 1 min | N/A (always Docker) | — |

```bash
# Build everything in local mode
make build BUILD_MODE=local

# Build single component
make build-controller BUILD_MODE=local
make build-ui BUILD_MODE=local
make build-golang-adk BUILD_MODE=local
make build-skills-init BUILD_MODE=local

# Python components always use Docker (native deps)
make build-kagent-adk   # No BUILD_MODE needed
make build-app          # No BUILD_MODE needed
```

**Files involved:**
- `go/Dockerfile.local` — thin runtime image for Go binaries
- `ui/Dockerfile.local` — thin runtime image for Next.js standalone
- `docker/skills-init/Dockerfile.local` — thin runtime image for krane
- `.local-build/` — staging directory for compiled artifacts (gitignored)

## Cluster Setup

```bash
# Create cluster with local registry + MetalLB
make create-kind-cluster

# Verify
kubectl get nodes --context kind-kagent
curl -s http://localhost:5001/v2/_catalog  # Local registry
```

This creates:
- Kind cluster `kagent` with Kubernetes v1.35.0
- Docker registry at `localhost:5001` (connected to kind network)
- MetalLB for LoadBalancer IPs (172.18.255.0/24)

## Helm Install

```bash
# Full install (builds all images first)
make helm-install

# Helm only (skip image build, use existing images)
make helm-install-provider

# Custom overrides via KAGENT_HELM_EXTRA_ARGS
make helm-install-provider \
  KAGENT_HELM_EXTRA_ARGS="--set ui.featureFlags.allowedNamespace=dbci-agent"
```

**Note:** SAPAICore is NOT a built-in Helm provider. The Helm chart installs with the default OpenAI provider.
SAPAICore ModelConfigs are created separately after deployment — see the "SAPAICore Configuration" section below.

### Runtime Feature Flags (via ConfigMap)

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
  --set ui.featureFlags.disableModelCreation=true \
  --kube-context kind-kagent
# Pod auto-restarts (checksum annotation triggers rollout)
```

### SAPAICore Configuration

SAPAICore is configured via kubectl after helm install (not through Helm values).

**Step 1: Create the credential secret**

```bash
kubectl create secret generic sap-aicore-creds -n kagent \
  --from-literal=client_id='YOUR_CLIENT_ID' \
  --from-literal=client_secret='YOUR_CLIENT_SECRET' \
  --from-literal=auth_url='https://YOUR_TENANT.authentication.YOUR_REGION.hana.ondemand.com' \
  --context kind-kagent \
  --dry-run=client -o yaml | kubectl apply --context kind-kagent -f -
```

**Step 2: Create ModelConfigs**

```bash
cat <<'EOF' | kubectl apply --context kind-kagent -f -
apiVersion: kagent.dev/v1alpha2
kind: ModelConfig
metadata:
  name: sap-aicore-claude-46-opus
  namespace: kagent
spec:
  apiKeySecret: sap-aicore-creds
  apiKeySecretKey: client_id
  model: anthropic--claude-4.6-opus
  provider: SAPAICore
  sapAICore:
    authUrl: https://YOUR_TENANT.authentication.YOUR_REGION.hana.ondemand.com
    baseUrl: https://api.ai.YOUR_REGION.aws.ml.hana.ondemand.com
    resourceGroup: default
---
apiVersion: kagent.dev/v1alpha2
kind: ModelConfig
metadata:
  name: sap-aicore-claude-46-sonnet
  namespace: kagent
spec:
  apiKeySecret: sap-aicore-creds
  apiKeySecretKey: client_id
  model: anthropic--claude-4.6-sonnet
  provider: SAPAICore
  sapAICore:
    authUrl: https://YOUR_TENANT.authentication.YOUR_REGION.hana.ondemand.com
    baseUrl: https://api.ai.YOUR_REGION.aws.ml.hana.ondemand.com
    resourceGroup: default
---
apiVersion: kagent.dev/v1alpha2
kind: ModelConfig
metadata:
  name: sap-aicore-gpt5
  namespace: kagent
spec:
  apiKeySecret: sap-aicore-creds
  apiKeySecretKey: client_id
  model: gpt-5
  provider: SAPAICore
  sapAICore:
    authUrl: https://YOUR_TENANT.authentication.YOUR_REGION.hana.ondemand.com
    baseUrl: https://api.ai.YOUR_REGION.aws.ml.hana.ondemand.com
    resourceGroup: default
---
apiVersion: kagent.dev/v1alpha2
kind: ModelConfig
metadata:
  name: sap-aicore-text-embedding-3-small
  namespace: kagent
spec:
  apiKeySecret: sap-aicore-creds
  apiKeySecretKey: client_id
  model: text-embedding-3-small
  provider: SAPAICore
  sapAICore:
    authUrl: https://YOUR_TENANT.authentication.YOUR_REGION.hana.ondemand.com
    baseUrl: https://api.ai.YOUR_REGION.aws.ml.hana.ondemand.com
    resourceGroup: default
EOF
```

**Step 3: Verify**

```bash
kubectl get modelconfig -n kagent --context kind-kagent
# Should show sap-aicore-* entries with status Accepted
```

**SAPAICore endpoint details:**
| Field | Value |
|-------|-------|
| Auth URL | `https://YOUR_TENANT.authentication.YOUR_REGION.hana.ondemand.com` |
| Base URL | `https://api.ai.YOUR_REGION.aws.ml.hana.ondemand.com` |
| Resource Group | `default` |
| Secret name | `sap-aicore-creds` (keys: `client_id`, `client_secret`, `auth_url`) |

## Build Only (Skip Helm Apply)

When the cluster is already running and you only want to update code **without reapplying Helm resources** (which would recreate default agents, reset ConfigMaps, etc.), use `make build` + `rollout restart` instead of `helm-install-provider`:

```bash
# Build all images and push to local registry — NO Helm apply
make build BUILD_MODE=local

# Restart deployments to pick up new images (imagePullPolicy=Always)
kubectl rollout restart deployment/kagent-controller deployment/kagent-ui -n kagent --context kind-kagent

# For Python runtime (if changed)
kubectl rollout restart deployment/kagent-app -n kagent --context kind-kagent
```

**Single component update:**
```bash
# Controller only
make build-controller BUILD_MODE=local
kubectl rollout restart deployment/kagent-controller -n kagent --context kind-kagent

# UI only
make build-ui BUILD_MODE=local
kubectl rollout restart deployment/kagent-ui -n kagent --context kind-kagent

# Python app only
make build-app
kubectl rollout restart deployment/kagent-app -n kagent --context kind-kagent
```

**Why use this instead of `helm-install-provider`?**
- `helm upgrade` reapplies all Helm templates, which recreates default agents (helm-agent, k8s-agent, etc.) even if you deleted them
- `build` + `rollout restart` only updates the running container images, leaving all K8s resources (Agents, ModelConfigs, etc.) untouched

## Port Forwarding

**Important:** `kubectl port-forward` must run in the user's shell session, not as a background agent task. Background tasks close stdin which causes port-forward to exit immediately. Always ask the user to run these commands themselves with the `!` prefix:

```bash
# Kill any existing port-forwards first
lsof -ti:8082 | xargs kill 2>/dev/null; lsof -ti:8083 | xargs kill 2>/dev/null

# Run in user's shell (use ! prefix in Claude Code)
! kubectl port-forward -n kagent --context kind-kagent svc/kagent-ui 8082:8080 &
! kubectl port-forward -n kagent --context kind-kagent svc/kagent-controller 8083:8083 &

# Or use MetalLB IPs directly (no port-forward needed):
# UI: http://172.18.255.0:8080
# API: http://172.18.255.1:8083
```

## Verification

```bash
# Feature flags
curl -s http://localhost:8082/api/config | python3 -m json.tool

# Agents
curl -s http://localhost:8083/api/agents | python3 -c "
import sys,json
for a in json.loads(sys.stdin.read())['data']:
  ns = a['agent']['metadata']['namespace']
  name = a['agent']['metadata']['name']
  print(f'{ns}/{name}: ready={a.get(\"deploymentReady\",False)}')
"

# All pods
kubectl get pods -n kagent --context kind-kagent
kubectl get pods -n dbci-agent --context kind-kagent
```

## Common Issues

### 1. `.dockerignore` blocks local-build artifacts
The Go and UI `.dockerignore` files exclude `bin/` and `.next/`. `BUILD_MODE=local` works around this by staging artifacts to `.local-build/` (outside the component directories).

### 2. Next.js standalone path nesting
`npm run build` on macOS produces `.next/standalone/<project-path>/ui/` (mirrors CWD). The Makefile `_compile-ui` target auto-flattens this to `.next/standalone/`.

### 3. External images stuck pulling (proxy/network)
Kind nodes can't reach Docker Hub or ghcr.io behind a proxy. Configure containerd proxy:
```bash
docker exec kagent-control-plane bash -c '
mkdir -p /etc/systemd/system/containerd.service.d
cat > /etc/systemd/system/containerd.service.d/proxy.conf <<EOF
[Service]
Environment="HTTP_PROXY=http://host.docker.internal:YOUR_PROXY_PORT"
Environment="HTTPS_PROXY=http://host.docker.internal:YOUR_PROXY_PORT"
Environment="NO_PROXY=localhost,127.0.0.1,10.96.0.0/12,10.244.0.0/16,localhost:5001,192.168.0.0/16"
EOF
systemctl daemon-reload && systemctl restart containerd
'
```

### 4. Controller fails startup (DB not ready)
PostgreSQL takes ~30s to start. Controller auto-restarts and succeeds once DB is up.

### 5. Buildx builder can't reach registries (gcr.io, docker.io)
The `kagent-builder` buildx container doesn't inherit host proxy settings. Recreate it with proxy:
```bash
docker buildx rm kagent-builder-v0.23.0 2>/dev/null
docker buildx create --name kagent-builder-v0.23.0 \
  --platform linux/amd64,linux/arm64 \
  --driver docker-container --use \
  --driver-opt network=host \
  --driver-opt env.http_proxy=http://host.docker.internal:YOUR_PROXY_PORT \
  --driver-opt env.https_proxy=http://host.docker.internal:YOUR_PROXY_PORT \
  --driver-opt env.HTTP_PROXY=http://host.docker.internal:YOUR_PROXY_PORT \
  --driver-opt env.HTTPS_PROXY=http://host.docker.internal:YOUR_PROXY_PORT
```
Also pre-pull base images via Docker Desktop (which uses its own proxy settings):
```bash
export http_proxy=http://127.0.0.1:YOUR_PROXY_PORT https_proxy=http://127.0.0.1:YOUR_PROXY_PORT
docker pull gcr.io/distroless/static:nonroot  # Go runtime
docker pull alpine:3.23                        # Skills-init and Python runtime
```

> **Note on base images:** Skills-init and Python components now use `alpine:3.23` instead of Chainguard images, which significantly reduces build times (faster pulls, smaller layers). Go components still use `gcr.io/distroless/static:nonroot` for minimal attack surface. UI uses Chainguard Wolfi.

## Cluster Info

| Resource | Value |
|----------|-------|
| Cluster name | `kagent` |
| Context | `kind-kagent` |
| Registry | `localhost:5001` |
| UI port-forward | `localhost:8082` |
| API port-forward | `localhost:8083` |
| MetalLB range | `172.18.255.0/24` |
