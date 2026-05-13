# scripts/ - Build & Deployment Automation

## Sub-packages

| Directory | Role |
|-----------|------|
| `build/` | Per-component Docker build scripts (shared env.sh + component builders) |
| `kind/` | Local Kind cluster setup (config, MetalLB, networking) |

## Key Files

| File | Role |
|------|------|
| `get-kagent` | Installer script for kagent CLI |
