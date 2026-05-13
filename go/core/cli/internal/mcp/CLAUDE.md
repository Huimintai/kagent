# cli/internal/mcp/

MCP server project scaffolding for `kagent mcp init`. Generates complete MCP server projects in Go, Python, TypeScript, or Java.

`config.go` defines `ProjectConfig` used by all framework generators.

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `builder/` | High-level build orchestrator (`builder.go`) that drives project generation |
| `frameworks/` | Language-specific generators (golang, java, python, typescript) plus shared base logic |
| `manifests/` | K8s manifest types (`ToolConfig`, `SecretsConfig`) and manifest manager for generated projects |
