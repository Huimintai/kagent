# reconciler/

Generic reconciliation logic for all kagent CRDs. Implements the `KagentReconciler` interface that the service controller delegates to.

## Responsibilities

- Reconcile Agent and SandboxAgent CRDs (compile spec, build K8s manifests, upsert to DB)
- Reconcile ModelConfig (validate secrets, compute hash for drift detection)
- Reconcile ModelProviderConfig (discover models, resolve secrets)
- Reconcile RemoteMCPServer and ToolServer (connect via MCP transport, list tools, persist to DB)
- Reconcile MCP-annotated Services (convert Service to RemoteMCPServer equivalent)
- Validate cross-namespace references and runtime feature compatibility
- Manage owned K8s objects lifecycle (create/update/prune)
- Update status conditions on all reconciled resources

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `utils/` | Helpers for finding owned objects, comparing K8s objects (including protobuf-based CRDs), and indexing by owner UID |
