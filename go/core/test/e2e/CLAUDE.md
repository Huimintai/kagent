# test/e2e/

End-to-end integration tests. Tests run against a real kagent deployment (typically Kind cluster) with a mock LLM backend.

## Test Files

| File | Coverage |
|------|----------|
| `auth_api_test.go` | Authentication and authorization API flows |
| `cli_runtime_test.go` | CLI runtime (Claude Code, Codex) agent creation and invocation |
| `invoke_api_test.go` | Agent invocation via HTTP API |
| `invoke_mcp_test.go` | MCP tool invocation through agents |
| `refprotection_test.go` | Cross-namespace reference protection validation |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `agents/` | Pre-built test agent binaries (e.g., kebab agent) used in E2E scenarios |
| `manifests/` | K8s manifest files applied during test setup |
| `mocks/` | Mock LLM server and STS server implementations for deterministic testing |
| `testdata/` | Test fixtures including skill scripts and agent configurations |
