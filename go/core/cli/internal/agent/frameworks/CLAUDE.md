# cli/internal/agent/frameworks/

Framework implementations for agent project scaffolding.

`frameworks.go` defines the `Generator` interface and the factory function `NewGenerator(framework, language)` that dispatches to concrete implementations.

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `adk/` | ADK framework scaffolding (currently Python only) |
| `common/` | Shared base generator (`BaseGenerator`) and manifest manager (`AgentManifest` / `kagent.yaml`) used by all framework generators |
