# go/core/internal/database — Data Layer

PostgreSQL + pgvector data layer using sqlc for query generation.

## Structure

| Package/File | Role |
|-------------|------|
| `client_postgres.go` | PostgreSQL client implementation |
| `connect.go` | Database connection setup |
| `sqlc.yaml` | sqlc configuration |
| `queries/` | SQL query definitions (source of truth) |
| `gen/` | sqlc-generated Go code (DO NOT EDIT) |
| `fake/` | In-memory fake client for testing |

## SQL Query Files (queries/)

| File | Domain |
|------|--------|
| `agents.sql` | Agent CRUD, listing, status |
| `sessions.sql` | Session management |
| `tasks.sql` | Task persistence |
| `tools.sql` | Tool definitions |
| `events.sql` | Event tracking |
| `feedback.sql` | User feedback |
| `memory.sql` | Vector memory (pgvector) |
| `langgraph.sql` | LangGraph checkpoints |
| `push_notifications.sql` | Push notification records |
| `comments.sql` | Session comments |
| `stats.sql` | Usage statistics |

## Workflow

1. Edit `.sql` files in `queries/`
2. Run `sqlc generate` (or `go generate ./...`)
3. Generated Go code appears in `gen/`
4. `client_postgres.go` wraps generated code with business logic
