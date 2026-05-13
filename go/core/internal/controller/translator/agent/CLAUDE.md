# translator/agent/

Translates Agent/SandboxAgent CRD specs into ADK AgentConfig JSON and Kubernetes manifests (Deployments, Services, Secrets, ConfigMaps).

## Key Interface

`AdkApiTranslator` with two phases:
1. `CompileAgent()` - Resolves models, tools, system messages, memory, context config into `AgentManifestInputs`
2. `BuildManifest()` - Emits K8s objects (Deployment, Service, Secret, ServiceAccount, etc.) from compiled inputs

## Files

| File | Role |
|------|------|
| `adk_api_translator.go` | Main translator implementation; constants, image config, MCP tool translation, model resolution, embedding config |
| `compiler.go` | `CompileAgent()` orchestrator; validates agent DAG, translates inline agents, resolves system messages, handles CLI runtime proxy sidecar injection |
| `conversion.go` | Converts K8s Service objects to RemoteMCPServer specs using annotations |
| `deployments.go` | Resolves deployment specs for inline and BYO agents; merges SharedDeploymentSpec, determines image/cmd/args per runtime |
| `manifest_builder.go` | `BuildManifest()` implementation; emits Deployment, Service, Secret, ConfigMap, ServiceAccount objects |
| `template.go` | System message Go-template processing with PromptSource data from ConfigMaps |
| `utils.go` | A2A AgentCard generation from agent spec |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `testdata/` | Golden test inputs/outputs for translator unit tests |
