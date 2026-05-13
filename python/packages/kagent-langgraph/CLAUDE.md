# kagent-langgraph - LangGraph Integration

LangGraph framework integration with A2A server support. Enables LangGraph-based agents to run as kagent services.

## Structure

| Directory | Purpose |
|-----------|---------|
| `src/` | Source code (namespace: `kagent.langgraph`) |
| `tests/` | Unit tests |

## Key Modules

- `_executor.py` - LangGraph agent executor
- `_a2a.py` - A2A protocol server adapter
- `_checkpointer.py` - State checkpointing
- `_converters.py` - Message format conversion
- `_error_mappings.py` - Error translation
- `_metadata_utils.py` - Metadata handling

## Dependencies

- `langgraph` >=0.6.5, `langchain-core` >=0.3.0
- `kagent-core` (shared utilities)
- `a2a-sdk`, `fastapi`, `uvicorn`
- `langsmith[otel]` (observability)
