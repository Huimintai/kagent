# cli/internal/mcp/frameworks/python/

Python MCP server project generator. Scaffolds a Python package with pyproject.toml and src/tests layout.

`generator.go` implements `GenerateProject` using embedded templates.

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `templates/` | Embedded templates: `src/tools/` (tool implementations), `src/core/` (server setup), `tests/` (test stubs), pyproject.toml, Dockerfile, README |
