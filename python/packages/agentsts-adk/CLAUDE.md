# agentsts-adk - ADK Security Token Service Integration

Framework-specific (Google ADK) integration points with agentsts-core for token exchange.

## Structure

| Directory | Purpose |
|-----------|---------|
| `src/` | Source code (namespace: `agentsts.adk`) |
| `tests/` | Unit tests |

## Dependencies

- `agentsts-core` (token exchange client)
- `google-adk` >=1.28.1 (ADK framework)
- `google-genai`, `google-auth`
- `httpx`, `pydantic`, `PyJWT`
