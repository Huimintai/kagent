# chore: update build configs, CI skills, Dockerfiles, and changelogs

**Commit:** `4b046dcc`  
**Date:** 2026-04-23  
**Type:** chore (build infrastructure)

## What

Introduces a `BUILD_MODE=local` development workflow that compiles Go binaries and Next.js
output on the host machine, then copies the pre-built artifacts into thin Docker images.
This replaces the previous approach of compiling everything inside Docker (Docker-in-Docker),
which was extremely slow due to lack of host caching and cross-compilation overhead.

Additionally, this commit adds a suite of Claude AI skills for CI workflows:
- **kagent-ci-kind**: Comprehensive guide for local Kind cluster builds and deployment
- **kagent-ci-prod**: Guide for remote production cluster deployment with cross-compilation
- **kagent-git**: Skill for reorganizing git commit history
- **kagent-dev** updates: Fork-specific customization info and reference documents

## Why

The Docker-in-Docker build for Go components took 5-8 minutes per build. For a tight
inner dev loop (edit → build → test), this was unacceptable. `BUILD_MODE=local` leverages
the host's Go toolchain and build cache, reducing controller builds from ~5 min to **13
seconds** — a 25x improvement. UI builds drop from 3-4 min to ~50s.

Cross-compilation (`TARGETARCH=amd64`) is critical because most developers run Apple
Silicon (arm64) Macs but deploy to amd64 clusters. Go handles this natively; the thin
`Dockerfile.local` just copies the pre-compiled binary.

The CI skills codify tribal knowledge about build/deploy workflows into reusable
documentation that Claude can reference, reducing the "how do I deploy this again?"
friction for new and existing contributors.

## Scope of Changes

| Area | Files |
|------|-------|
| **Makefile** | Major additions: `BUILD_MODE=local` logic, per-component build targets (`build-controller`, `build-ui`, etc.), `TARGETARCH` cross-compilation, `DOCKER_REGISTRY`/`DOCKER_REPO` overrides |
| **Docker** | New `go/Dockerfile.local`, `ui/Dockerfile.local`, `docker/skills-init/Dockerfile.local` — thin runtime images that `COPY` pre-built artifacts |
| **UI** | New `ui/conf/nginx.conf` for serving Next.js standalone output in the thin image |
| **Claude Skills** | New `.claude/skills/kagent-ci-kind/`, `.claude/skills/kagent-ci-prod/`, `.claude/skills/kagent-git/`; updated `.claude/skills/kagent-dev/` |
| **Git** | `.gitignore` updated for `.local-build/` staging directory |
