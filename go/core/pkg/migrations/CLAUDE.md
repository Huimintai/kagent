# pkg/migrations/

Database schema migrations using golang-migrate. Embeds SQL files and provides a runner that applies them at startup.

`migrations.go` embeds the `core/` and `vector/` directories via `//go:embed`.
`runner.go` implements `RunUp()` which applies core migrations first, then optionally vector migrations (requires pgvector extension).

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `core/` | Core schema migrations (agents, sessions, tools, etc.) - numbered SQL up/down pairs |
| `vector/` | Vector/memory schema migrations (pgvector tables, HNSW indexes) - applied only when vector support is enabled |
