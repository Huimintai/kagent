# go/core/internal/controller/translator — CRD-to-Runtime Translation

Translates CRD specs (Agent, SandboxAgent) into runtime artifacts: Kubernetes manifests (Deployments, ConfigMaps) and ADK configuration files.

## Sub-packages

| Package | Role |
|---------|------|
| `agent/` | Agent translator — builds K8s manifests and ADK config from Agent/SandboxAgent CRDs |
| `labels/` | Label constants and helpers for K8s resource labeling |

## Top-level Files

| File | Role |
|------|------|
| `mutate.go` | Mutation logic for runtime resources |
| `mutate_sandbox_test.go` | Sandbox mutation tests |

## agent/ Package

| File | Role |
|------|------|
| `adk_api_translator.go` | Converts CRD Agent spec to ADK API config struct |
| `manifest_builder.go` | Builds K8s Deployment/ConfigMap manifests |
| `compiler.go` | Compiles full agent runtime artifacts |
| `conversion.go` | Type conversion helpers |
| `deployments.go` | Deployment spec construction |
| `template.go` | Init container script templating |
| `utils.go` | Shared utilities |
| `testdata/` | Golden test fixtures |

## Runtime Awareness

The translator handles all runtime types (python, go, claudeCode, codex) and generates appropriate container specs, environment variables, and config mounts for each.
