# kagent-sota-adapter - CLI-to-A2A Adapter

Generic adapter that wraps CLI-based AI agents (Claude Code, Codex) as BYO kagent agents via the A2A protocol.

## Structure

| Directory | Role |
|-----------|------|
| `src/kagent/sota_adapter/` | Core adapter logic (app, converters, parsers) |
| `tests/` | Unit tests |

## Key Modules (src/kagent/sota_adapter/)

| File/Dir | Role |
|----------|------|
| `parsers/` | CLI output parsers per runtime |
| `app.py` | FastAPI A2A application builder |
| `_executor.py` | CLI process executor |
| `_converters.py` | Event converters to A2A format |

## Dependencies

- kagent-core (shared config, tracing)
