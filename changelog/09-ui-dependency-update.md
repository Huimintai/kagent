# chore(ui): update dependencies

**Commit:** `3054c93c`  
**Date:** 2026-04-20  
**Type:** chore (UI dependencies)

## What

Routine dependency update for the UI package. Notable version bumps:

| Package | From | To |
|---------|------|----|
| Next.js | 16.2.2 | 16.2.3 |
| React | 19.2.4 | 19.2.5 |
| Storybook | 10.3.4 | 10.3.5 |
| @types/node | 25.5.2 | 25.6.0 |
| Vitest/Playwright | 4.1.2 | 4.1.4 |
| MSW (MockServiceWorker) | 2.13.2 | 2.13.4 |

## Why

Keeps dependencies current with security patches and bug fixes. Patch-level updates
like these are low-risk but important for:
- Closing known CVEs in transitive dependencies
- Picking up stability fixes in the build toolchain (Next.js, Vitest)
- Avoiding dependency drift that makes future upgrades harder

## Scope of Changes

| Area | Files |
|------|-------|
| **Lock file** | `ui/package-lock.json` — full regeneration |
| **MSW** | `ui/public/mockServiceWorker.js` — version bump |
