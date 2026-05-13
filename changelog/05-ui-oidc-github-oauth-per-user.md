# feat(ui): add OIDC login, GitHub OAuth, and per-user agent access

**Commit:** `867d3cd3`  
**Date:** 2026-04-23  
**Type:** feat (UI)

## What

Implements the UI-side authentication layer with three interconnected features:

### 1. OIDC Login
The UI resolves the logged-in user from OIDC proxy headers (`x-forwarded-user`,
`x-auth-request-email`) set by an upstream proxy like oauth2-proxy. On startup, the
user store fetches identity via `/api/auth/user` and stores it for session-scoped
agent filtering and ownership.

### 2. GitHub OAuth (Multi-Instance)
Full OAuth2 authorization code flow supporting **multiple GitHub Enterprise instances**
simultaneously:
- Each instance gets its own connect/disconnect button, OAuth cookies, and token
  lifecycle management
- CSRF protection via `state` parameter with HMAC verification
- Token revocation on disconnect (calls GitHub's revocation endpoint)
- Pre-login dialog for enterprise instances requiring SSO
- Tokens are stored as per-instance HTTP-only cookies

### 3. Auth Header Propagation
The `auth.ts` module now forwards a comprehensive set of identity headers to the
backend API calls:
- `X-Auth-Request-*` and `X-Forwarded-*` headers (from OIDC proxy)
- Synthesized `X-User-Id` from the most reliable source (proxy headers → local store)

### 4. Identicon Component
Generates deterministic 5×5 mirrored pixel-art avatars from username hashes,
providing visual user identification without requiring profile pictures.

## Why

Multi-user deployments behind an OIDC proxy (oauth2-proxy, Keycloak, etc.) need the
UI to be identity-aware so that:
1. Users see only their own private agents
2. Agent creation records the owner's identity
3. Edit/delete permissions are enforced per-user

GitHub OAuth is specifically needed for MCP server tools that require user-specific
GitHub tokens. In an enterprise setting, users access different repos with different
permissions — a shared service account token would be both a security risk and
functionally inadequate.

## Scope of Changes

| Area | Files |
|------|-------|
| **API Routes** | New routes under `app/actions/api/auth/github/` (callback, disconnect, route, status) and `api/auth/user/` |
| **Components** | New `GitHubConnectButton.tsx` (252 lines), `Identicon.tsx` |
| **Auth Library** | `lib/auth.ts` — expanded header forwarding with X-User-Id synthesis |
| **GitHub Library** | New `lib/github.ts` — multi-instance config from env vars, API helpers |
| **OIDC Library** | New `lib/oidcUser.ts` — client-side user info fetch |
| **User Store** | `lib/userStore.ts` — async OIDC resolution on init, `clearLoginSession()` |
