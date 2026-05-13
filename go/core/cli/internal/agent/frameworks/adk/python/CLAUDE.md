# cli/internal/agent/frameworks/adk/python/

Python ADK agent project generator. Scaffolds a complete Python agent project with pyproject.toml, Dockerfile, docker-compose, and agent/MCP server code.

`generator.go` implements the `Generator` interface using embedded templates.

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `templates/` | Go template files embedded via `//go:embed`; contains `agent/` (agent Python code) and `mcp_server/` (MCP server stubs), plus project-level templates (Dockerfile, pyproject.toml, docker-compose, README) |
