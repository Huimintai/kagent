# pkg/agent/

Core agent construction logic for the Go ADK runtime.

`agent.go` implements `CreateGoogleADKAgent()` which builds a Google ADK `agent.Agent` from an `AgentConfig`. Resolves model providers (OpenAI, Anthropic, Gemini, Ollama), connects MCP tool servers, wires remote sub-agents, and configures memory tools.

`approval.go` implements human-in-the-loop tool approval callbacks. `createllm_test.go` tests model creation logic.

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `testdata/` | Test fixture files for agent creation unit tests |
