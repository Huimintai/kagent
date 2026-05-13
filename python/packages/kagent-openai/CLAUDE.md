# kagent-openai - OpenAI Agents SDK Integration

OpenAI Agents SDK integration with A2A server support. Runs OpenAI-agents-based agents as kagent services.

## Structure

| Directory | Purpose |
|-----------|---------|
| `src/` | Source code (namespace: `kagent.openai`) |

## Key Modules (in `kagent.openai`)

- `_agent_executor.py` - OpenAI agent executor
- `_a2a.py` - A2A protocol server adapter
- `_event_converter.py` - Event format conversion
- `_session_service.py` - Session management
- `tools/` - Tool adapters (`_tools.py`)

## Dependencies

- `openai` >=1.72.0, `openai-agents` >=0.4.0
- `kagent-core`, `kagent-skills`
- `a2a-sdk`, `fastapi`, `uvicorn`
- `opentelemetry-instrumentation-openai-agents`
