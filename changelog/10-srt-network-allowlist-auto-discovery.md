# feat: auto-discover same-namespace services for SRT network allowlist

**Date:** 2026-04-22  
**Type:** feat (Go controller, Go ADK, Python skills)

## What

At agent pod creation time, the controller now automatically discovers all
Kubernetes Services in the agent's namespace and adds their DNS names to the
`network.allowedDomains` list in `srt-settings.json`. One DNS form is added
per service:

- `<svc>.<namespace>` (short form, e.g. `cam-profiles-graph.kagent`)

User-specified domains from `spec.sandbox.network.allowedDomains` are merged in
after the auto-discovered entries.

Two additional top-level SRT flags are now set unconditionally:

- `"enableWeakerNestedSandbox": true` — enables weaker nested sandbox mode for
  container environments (avoids double-bwrap failures)
- `"allowAllUnixSockets": true` — disables Unix socket blocking inside the sandbox

The `buildSRTSettingsJSON` function is promoted from a free function to an
`adkApiTranslator` method so it can call `kube.List` at reconcile time.

## Why

The SRT sandbox uses `bwrap --unshare-net` to isolate the network namespace and
routes all traffic through its own proxy. Without explicit `allowedDomains`
entries, agents with skills or code execution could not reach other services in
the same namespace. Previously, operators had to enumerate every service manually
via `spec.sandbox.network`.

Auto-discovery removes this friction: agents gain access to all same-namespace
services by default, without any manual configuration, and the allowlist stays
current as services are added or removed (re-evaluated at every reconcile).

`enableWeakerNestedSandbox` and `allowAllUnixSockets` address runtime failures
observed in Docker/Kind environments where the default strict sandbox mode
prevents correct operation.

## Scope of Changes

| Area | Files |
|------|-------|
| **Manifest Builder** | `manifest_builder.go` — `buildSRTSettingsJSON` → method, `client.InNamespace` service listing, new top-level JSON flags |
| **Manifest Builder Test** | `manifest_builder_test.go` — updated to use fake kube client, added assertions for new flags |
| **Skills Unit Test** | `skills_unit_test.go` — fixed pre-existing `prepareSkillsInitData` call-site arity mismatch |
| **Golden Files** | `testdata/outputs/agent_with_code.json`, `agent_with_git_skills.json`, `agent_with_inline_skills.json`, `agent_with_mixed_skills.json`, `agent_with_skills.json` — updated srt-settings.json and config-hash |
