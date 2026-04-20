# feat(ui): update agent create page, model pages, and misc improvements

**Commit:** `f82699a7`  
**Date:** 2026-04-20  
**Type:** feat (UI)

## What

This is the UI integration commit that wires together all backend features from
previous commits into a cohesive user experience.

### 1. Agent Create/Edit Page Overhaul
Complete rewrite of the agent creation form:
- **Inline skills editor**: Dedicated section with name/description/content fields
  for creating skills that don't require container images
- **CLI container images**: Separate section (distinct from skills) with preset
  quick-add buttons for common tool containers
- **Private/public toggle**: Owner can set agent visibility
- **Category selection**: Combobox with predefined suggestions
- **Read-only view mode**: Non-owners see the agent config but cannot edit
- **Protected agent check**: Protected agents (per feature flag) show a warning
  and block modifications

### 2. Per-User Agent Visibility
The `getAgents()` server action now filters agents based on the current user:
- Private agents are visible only to their owner
- Public agents are visible to everyone
- Ownership is determined from `user_id` in the response or `kagent.dev/user-id`
  annotation

### 3. SAPAICore UI Support
Model creation/edit forms now support SAP AI Core as a provider option with:
- Proper icon in provider combobox
- Config payload fields (baseUrl, resourceGroup, authUrl)
- Model creation disabled state with explanatory banner (when feature flag is set)

### 4. A2A Token Propagation
The A2A route handler converts GitHub OAuth cookies to `X-MCP-Token-Github-*`
headers for the backend, completing the cookie-to-header bridge.

### 5. Feature Flag Integration
- Model pages respect `disableModelCreation` — disables new/edit/delete buttons
- Server pages respect `disableMcpServerCreation`
- All pages respect `allowedNamespace` for namespace filtering

## Why

The previous agent creation form was a monolithic page that didn't distinguish
between inline skills and container-based skills. With the new inline skills CRD
support (commit 08), the UI needs separate editing surfaces for each. The private/
public toggle and read-only mode are required by the per-user access control
system — without them, the visibility feature would have no UI surface.

SAPAICore UI support is the final piece of the provider integration that spans
Go ADK → Python ADK → CRDs → HTTP handlers → UI.

## Scope of Changes

| Area | Files |
|------|-------|
| **Agent Create Page** | `agents/new/page.tsx` — major rewrite with inline skills, CLI containers, access control |
| **Server Actions** | `actions/agents.ts` — per-user filtering, category/tool-type label computation |
| **Auth Utils** | New `actions/utils.ts` — `getCurrentUserId()` helper |
| **Model Pages** | `models/new/page.tsx`, `models/page.tsx` — SAPAICore support, feature flag gating |
| **Server/Tool Pages** | `servers/page.tsx`, `tools/page.tsx` — namespace filtering, flag gating |
| **A2A Route** | `a2a/.../route.ts` — cookie-to-header token bridge |
| **Sidebars** | `AgentDetailsSidebar.tsx`, `SessionsSidebar.tsx` — inline skill display, positioning fixes |
| **Types** | `types/index.ts` — new InlineSkill, SAPAICoreConfigPayload types |
| **Provider UI** | `ModelProviderCombobox.tsx`, `ProviderCombobox.tsx`, `lib/providers.ts` — SAPAICore option |
