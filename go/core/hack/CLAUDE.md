# hack/

Developer utilities and test helpers (not shipped in production images).

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `makeagentconfig/` | CLI tool that generates a sample `AgentConfig` JSON and A2A AgentCard for local testing |
| `mockllm/` | Runs a mock LLM server (using the mockllm library) plus a mock STS server for E2E test development |
