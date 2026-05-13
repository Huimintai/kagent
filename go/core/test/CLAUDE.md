# go/core/test — Integration and E2E Tests

## Structure

| Directory | Role |
|-----------|------|
| `e2e/` | End-to-end tests against a running cluster |

## E2E Tests (e2e/)

| File/Dir | Tests |
|----------|-------|
| `agents/` | Agent CRUD and lifecycle |
| `auth_api_test.go` | Authentication API |
| `cli_runtime_test.go` | CLI runtime (Claude Code, Codex) agents |
| `invoke_api_test.go` | Agent invocation API |
| `invoke_mcp_test.go` | MCP tool invocation |
| `refprotection_test.go` | Reference protection (prevent deleting in-use resources) |
| `manifests/` | Test K8s manifests |
| `mocks/` | Mock servers for testing |
| `testdata/` | Test fixtures |

## Running

```bash
cd go
go test -tags=e2e ./core/test/e2e/...
```
