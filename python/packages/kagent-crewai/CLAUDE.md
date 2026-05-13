# kagent-crewai - CrewAI Integration

CrewAI framework integration for kagent with A2A server support, session-aware memory, and FastAPI hosting.

## Structure

| Directory | Role |
|-----------|------|
| `src/kagent/crewai/` | Core CrewAI runtime and A2A integration |

## Key Modules (src/kagent/crewai/)

| File | Role |
|------|------|
| `_executor.py` | CrewAI task/crew executor |
| `_app.py` | FastAPI A2A application builder |
| `_event_converter.py` | Crew events to A2A event stream |
| `_memory.py` | Session-aware memory backend |

## Dependencies

- kagent-core (shared config, A2A, tracing)
- crewai (framework)
