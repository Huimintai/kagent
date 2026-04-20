# feat(ui): add feature flags, agent filtering, and UI branding

**Commit:** `37e0b23e`  
**Date:** 2026-04-20  
**Type:** feat (UI, Helm)

## What

### 1. Feature Flags via ConfigMap
UI behavior is now controlled by environment variables injected from a Kubernetes
ConfigMap (`kagent-ui-config`), making customization a runtime concern rather than
a build-time one. Available flags:

| Flag | Effect |
|------|--------|
| `KAGENT_ALLOWED_NAMESPACE` | Lock UI to a single namespace |
| `KAGENT_DISABLE_MODEL_CREATION` | Hide the "New Model" button |
| `KAGENT_DISABLE_MCP_SERVER_CREATION` | Disable MCP server creation |
| `KAGENT_DISABLE_BYO_AGENT_CREATION` | Disable BYO agent type |
| `KAGENT_PROTECTED_AGENTS` | Comma-separated list of protected agent names |

The UI fetches these from `/api/config` on startup via a Zustand store.

### 2. Agent Filtering & Organization
The agent list is completely overhauled:
- **Category grouping**: Agents are grouped by `kagent.dev/category` label into
  collapsible sections
- **Search**: Free-text search across agent names
- **Privacy filter**: Toggle between "All" and "My Agents"
- **Badge filters**: Filter by tool type and category badges
- **Visual indicators**: Private/public badge, protection shield icon, owner info

### 3. UI Rebranding
Primary color changed from purple (hue 262) to blue (hue 213.6), aligning with
SAP's visual identity. Header updated to show logged-in user identity, GitHub
connect button, and sign-out option.

### 4. Helm Integration
All feature flags and GitHub OAuth config are exposed as Helm values under
`ui.featureFlags` and `ui.github`, with a config checksum annotation on the
deployment to trigger automatic rollouts on config changes.

## Why

**Feature flags** are essential for enterprise deployments where different teams
need different UI capabilities. For example, a platform team might provision models
centrally and disable model creation for application teams. Runtime ConfigMap-based
flags avoid the need to rebuild Docker images for each deployment variant.

**Agent filtering** becomes critical once you have 20+ agents across multiple
categories and users. Without grouping and filtering, the flat list becomes
unmanageable.

**Rebranding** aligns the UI with the deployment environment's visual identity,
which matters for internal enterprise tools where brand consistency is expected.

## Scope of Changes

| Area | Files |
|------|-------|
| **Helm** | New `ui-configmap.yaml`, updated `ui-deployment.yaml` and `values.yaml` with featureFlags/github sections |
| **UI API** | New `/api/config` route returning feature flags as JSON |
| **Agent Components** | Rewritten `AgentList.tsx` (category groups), new `AgentFilterToolbar.tsx`, `CategoryCombobox.tsx`; updated `AgentCard.tsx` |
| **State Management** | New `lib/configStore.ts` (Zustand), new `lib/constants.ts` |
| **Header/Footer** | Updated with user identity, GitHub connect, sign-out |
| **CSS** | `globals.css` — primary color rebranding |
| **Icons** | New `SAPAICore.tsx` provider icon |
| **Onboarding** | Updated branding in wizard and welcome step |
