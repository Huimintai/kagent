# feat(ui): add feature flag gating to model pages and new TypeScript types

**Commit:** `6df86c66`  
**Date:** 2026-04-23  
**Type:** feat (UI)

## What

### 1. Model Pages Feature Flag Gating
Model creation and listing pages now respect the `disableModelCreation` feature flag:
- Model list page: "New Model" button disabled, edit/delete buttons disabled, info
  banner displayed with configurable message
- Model create page: redirects to info banner when flag is active (edit mode still
  permitted)
- Import of `useAppConfig` store for flag state

### 2. SAPAICore Type Cleanup
- Rename `SAPAICoreConfigPayload` to `SAPAICoreConfig` in `ModelConfigSpec`
- Add new `SAPAICoreConfig` interface with `baseUrl`, `resourceGroup`, `authUrl`

### 3. New TypeScript Types
Additional type definitions to support backend features:
- `InlineSkill` — name/description/content for inline prompt-based skills
- `ResourceMetadata` — gains `annotations` and `labels` fields
- `AgentResponse` — gains `user_id` and `private_mode` fields
- `DeclarativeAgentSpec` — gains `inlineSkills` array

## Why

The model creation feature flag was configured in the backend ConfigMap and fetched
by the config store, but the model pages were not yet gated. This commit wires the
flag to the actual UI buttons and form pages.

The TypeScript type additions mirror backend CRD/API changes from the Go and Python
commits, providing type safety for the UI components that consume these fields.

## Scope of Changes

| Area | Files |
|------|-------|
| **Model Create Page** | `models/new/page.tsx` — feature flag gating, SAPAICore type cast fix |
| **Model List Page** | `models/page.tsx` — disabled buttons, info banner |
| **Types** | `types/index.ts` — SAPAICoreConfig, InlineSkill, agent access fields |
