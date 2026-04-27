---
name: kagent-ci-kind
description: >
  Build and deploy kagent to a local Kind cluster. Covers fast local builds (BUILD_MODE=local),
  runtime ConfigMap feature flags, port-forwarding, and debugging.
  Use this skill when deploying kagent locally for development or testing.
---

# Kagent CI — Local Kind Cluster

## Critical Rules

1. **ALWAYS use `localhost:5001` as the registry.** The Makefile defaults (`DOCKER_REGISTRY=localhost:5001`, `DOCKER_REPO=kagent-dev/kagent`) are correct for Kind. Do NOT set `DOCKER_REGISTRY` or `DOCKER_REPO` in `.env` — the prod skill passes them explicitly. If `.env` contains these overrides, remove them.
2. **NEVER use `make helm-install` or `make helm-install-provider`.** Helm upgrade reapplies all templates, recreates default agents, resets ConfigMaps, and overwrites manual customizations.
3. **When using a proxy**, Python Docker builds (kagent-adk, app) need explicit `--build-arg` proxy settings because they use plain `docker build` (not buildx) in `BUILD_MODE=local`. See the Proxy section.
4. **Always use `kind load docker-image`** to load images into the Kind node. containerd v2.2.0 has a registry:2 incompatibility that causes 502 on image pull. `kind load` bypasses the registry entirely.
5. **Deploy by setting the image digest**, not by `rollout restart`. After `kind load`, get the image ID with `docker inspect` and use `kubectl set image` with the `@sha256:...` digest. Each build produces a new digest, so Kubernetes automatically rolls out new pods.

## Quick Reference

```bash
VERSION=$(git describe --tags --always)
IMG=localhost:5001/kagent-dev/kagent

# Full rebuild + deploy
make build BUILD_MODE=local
kind load docker-image $IMG/controller:v0.0.0-$VERSION --name kagent
kind load docker-image $IMG/ui:v0.0.0-$VERSION --name kagent
kubectl set image deployment/kagent-controller controller=$IMG/controller@$(docker inspect --format='{{.Id}}' $IMG/controller:v0.0.0-$VERSION) -n kagent --context kind-kagent
kubectl set image deployment/kagent-ui ui=$IMG/ui@$(docker inspect --format='{{.Id}}' $IMG/ui:v0.0.0-$VERSION) -n kagent --context kind-kagent

# Controller only
make build-controller BUILD_MODE=local
kind load docker-image $IMG/controller:v0.0.0-$VERSION --name kagent
kubectl set image deployment/kagent-controller controller=$IMG/controller@$(docker inspect --format='{{.Id}}' $IMG/controller:v0.0.0-$VERSION) -n kagent --context kind-kagent

# UI only
make build-ui BUILD_MODE=local
kind load docker-image $IMG/ui:v0.0.0-$VERSION --name kagent
kubectl set image deployment/kagent-ui ui=$IMG/ui@$(docker inspect --format='{{.Id}}' $IMG/ui:v0.0.0-$VERSION) -n kagent --context kind-kagent

# No-cache rebuild
make build BUILD_MODE=local DOCKER_BUILD_ARGS="--no-cache --push --platform linux/arm64"
kind load docker-image $IMG/controller:v0.0.0-$VERSION --name kagent
kind load docker-image $IMG/ui:v0.0.0-$VERSION --name kagent
kubectl set image deployment/kagent-controller controller=$IMG/controller@$(docker inspect --format='{{.Id}}' $IMG/controller:v0.0.0-$VERSION) -n kagent --context kind-kagent
kubectl set image deployment/kagent-ui ui=$IMG/ui@$(docker inspect --format='{{.Id}}' $IMG/ui:v0.0.0-$VERSION) -n kagent --context kind-kagent

# Python components (always Docker, no BUILD_MODE needed)
make build-kagent-adk
make build-app

# Port-forward
kubectl port-forward -n kagent --context kind-kagent svc/kagent-ui 8082:8080 &
kubectl port-forward -n kagent --context kind-kagent svc/kagent-controller 8083:8083 &
```

### How it works

1. `make build` compiles and builds Docker images locally with a version tag (e.g., `v0.0.0-65f0b756`)
2. `kind load` copies the image from the local Docker daemon into the Kind node's containerd store
3. `docker inspect --format='{{.Id}}'` returns the image digest (`sha256:abc123...`)
4. `kubectl set image` updates the deployment spec with `image@sha256:...` — each build produces a different digest, so Kubernetes always sees a spec change and creates new pods automatically

Deployments must use `imagePullPolicy: Never` so kubelet uses the pre-loaded image without contacting the registry. Set this once:

```bash
kubectl patch deployment kagent-controller -n kagent --context kind-kagent --type json \
  -p '[{"op":"replace","path":"/spec/template/spec/containers/0/imagePullPolicy","value":"Never"}]'
kubectl patch deployment kagent-ui -n kagent --context kind-kagent --type json \
  -p '[{"op":"replace","path":"/spec/template/spec/containers/0/imagePullPolicy","value":"Never"}]'
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
make build BUILD_MODE=local                  # All components
make build-controller BUILD_MODE=local       # Controller only
make build-ui BUILD_MODE=local               # UI only
make build-golang-adk BUILD_MODE=local       # Go ADK only
make build-skills-init BUILD_MODE=local      # Skills-init only
```

**Files involved:**
- `go/Dockerfile.local` — thin runtime image for Go binaries
- `ui/Dockerfile.local` — thin runtime image for Next.js standalone
- `docker/skills-init/Dockerfile.local` — thin runtime image for krane
- `.local-build/` — staging directory for compiled artifacts (gitignored)

## Cluster Setup

```bash
make create-kind-cluster
kubectl get nodes --context kind-kagent
curl -s http://localhost:5001/v2/_catalog
```

This creates:
- Kind cluster `kagent` with Kubernetes v1.35.0
- Docker registry at `localhost:5001` (connected to kind network)
- MetalLB for LoadBalancer IPs (172.18.255.0/24)

## Why `kind load` (not registry pull)

containerd v2.2.0 appends `?ns=localhost:5001` to all mirror requests. Docker's `registry:2` (v2.8.3) returns 502 on HEAD requests with this parameter, causing `ImagePullBackOff`. `kind load docker-image` copies images directly from the local Docker daemon into the Kind node's containerd store, bypassing the registry.

**Manual rebuild without OCI attestation** (if building outside the Makefile):
```bash
docker build --platform linux/arm64 --provenance=false \
  -t localhost:5001/kagent-dev/kagent/controller:$VERSION \
  -f go/Dockerfile.local ./.local-build
kind load docker-image localhost:5001/kagent-dev/kagent/controller:$VERSION --name kagent
```

## Runtime Feature Flags (via ConfigMap)

UI feature flags live in the `kagent-ui-config` ConfigMap — change with pod restart, no rebuild needed.

| ConfigMap Key | Default | Effect |
|--------------|---------|--------|
| `KAGENT_ALLOWED_NAMESPACE` | `"dbci-agent"` | Lock UI to one namespace |
| `KAGENT_DISABLE_MODEL_CREATION` | `"true"` | Hide New Model button |
| `KAGENT_DISABLE_MCP_SERVER_CREATION` | `"true"` | Disable MCP server creation |
| `KAGENT_DISABLE_BYO_AGENT_CREATION` | `"true"` | Disable BYO agent type |
| `KAGENT_PROTECTED_AGENTS` | `""` | Comma-separated protected agent names |

```bash
kubectl patch configmap kagent-ui-config -n kagent --context kind-kagent --type merge -p '{
  "data": {
    "KAGENT_ALLOWED_NAMESPACE": "dbci-agent",
    "KAGENT_DISABLE_MODEL_CREATION": "true"
  }
}'
# Trigger new pods to pick up ConfigMap change:
IMG=localhost:5001/kagent-dev/kagent
kubectl set image deployment/kagent-ui ui=$IMG/ui@$(docker inspect --format='{{.Id}}' $IMG/ui:v0.0.0-$(git describe --tags --always)) -n kagent --context kind-kagent
```

## Port Forwarding

**Important:** `kubectl port-forward` must run in the user's shell session, not as a background agent task. Always ask the user to run these with the `!` prefix:

```bash
lsof -ti:8082 | xargs kill 2>/dev/null; lsof -ti:8083 | xargs kill 2>/dev/null
! kubectl port-forward -n kagent --context kind-kagent svc/kagent-ui 8082:8080 &
! kubectl port-forward -n kagent --context kind-kagent svc/kagent-controller 8083:8083 &

# Or use MetalLB IPs directly (no port-forward needed):
# UI: http://172.18.255.0:8080
# API: http://172.18.255.1:8083
```

## Verification

```bash
curl -s http://localhost:8082/api/config | python3 -m json.tool
kubectl get pods -n kagent --context kind-kagent
kubectl get pods -n dbci-agent --context kind-kagent
```

## Common Issues

### 1. `.dockerignore` blocks local-build artifacts
`BUILD_MODE=local` stages artifacts to `.local-build/` (outside component directories) to avoid `.dockerignore` exclusions.

### 2. Next.js standalone path nesting
`npm run build` on macOS produces `.next/standalone/<project-path>/ui/`. The Makefile `_compile-ui` target auto-flattens this.

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
Recreate buildx builder with proxy:
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
Pre-pull base images via Docker Desktop:
```bash
export http_proxy=http://127.0.0.1:YOUR_PROXY_PORT https_proxy=http://127.0.0.1:YOUR_PROXY_PORT
docker pull gcr.io/distroless/static:nonroot
docker pull alpine:3.23
```

### 6. Images pushed to wrong registry (docker.io instead of localhost:5001)
**Cause:** `.env` contains `DOCKER_REGISTRY=docker.io`. **Fix:** Remove `DOCKER_REGISTRY` and `DOCKER_REPO` from `.env`.

### 7. Python Docker builds fail with network errors behind proxy
Python components use plain `docker build` (not buildx). Pass proxy as build args:
```bash
export http_proxy=http://127.0.0.1:YOUR_PROXY_PORT https_proxy=http://127.0.0.1:YOUR_PROXY_PORT
make build-controller build-ui build-golang-adk build-skills-init BUILD_MODE=local

# Python with explicit proxy args if needed:
docker build --platform linux/arm64 \
  --build-arg http_proxy=http://host.docker.internal:YOUR_PROXY_PORT \
  --build-arg https_proxy=http://host.docker.internal:YOUR_PROXY_PORT \
  -t localhost:5001/kagent-dev/kagent/kagent-adk:$(git describe --tags --always) \
  -f python/Dockerfile ./python
```

### 8. `.local-build/app` binary collision (controller vs golang-adk)
Both compile to `.local-build/app`. After `make build` (full), `.local-build/app` contains the golang-adk binary. Fix: use `make build-controller BUILD_MODE=local` which recompiles before building the image, or manually:
```bash
cd go && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ../.local-build/app core/cmd/controller/main.go && cd ..
docker build --platform linux/arm64 --provenance=false \
  -t localhost:5001/kagent-dev/kagent/controller:$VERSION \
  -f go/Dockerfile.local ./.local-build
kind load docker-image localhost:5001/kagent-dev/kagent/controller:$VERSION --name kagent
```

## Cluster Info

| Resource | Value |
|----------|-------|
| Cluster name | `kagent` |
| Context | `kind-kagent` |
| Registry | `localhost:5001` |
| UI port-forward | `localhost:8082` |
| API port-forward | `localhost:8083` |
| MetalLB range | `172.18.255.0/24` |
