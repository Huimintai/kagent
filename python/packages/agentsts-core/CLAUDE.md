# agentsts-core - RFC 8693 Token Exchange Client

Security Token Service client implementing OAuth 2.0 Token Exchange (RFC 8693). Provides credential exchange for agent authentication.

## Structure

| Directory | Purpose |
|-----------|---------|
| `src/` | Source code (namespace: `agentsts.core`) |
| `tests/` | Unit tests |

## Dependencies

- `httpx` (HTTP client)
- `pydantic` (data validation)
- `cryptography` (JWT handling)
- `PyJWT` (token parsing)
