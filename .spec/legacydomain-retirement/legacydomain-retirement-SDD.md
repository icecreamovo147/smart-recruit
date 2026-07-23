# Legacydomain Retirement SDD

## 1. Existing Architecture Summary

The current service architecture uses local `domain/application/infrastructure/interfaces/runtime` packages where prior DDD extraction has completed. Four services still include service-private `internal/legacydomain` packages:

- Offer and Interview use legacy repositories/models as infrastructure adapter backing.
- Recruitment still constructs active legacy job, taxonomy, candidate, application, collaboration, admin, usage, authz, outbox, and OSS-facing services from `cmd/recruitment-service`.
- AI Agent still constructs active legacy AI chat, candidate chat, agent-run, prompt/config, MCP, skill, embedding, provider, worker, and recruiting-intelligence services from `cmd/ai-agent-service`.

The knowledge base contains active source references to these paths.

## 2. Problem Analysis

`legacydomain` now hides real remaining migration work. Deleting the directories mechanically would either break runtime behavior or move the same debt under another name. The retirement must first replace active dependencies with local infrastructure and owner contracts, then delete dead legacy packages, then update guardrails and knowledge.

## 3. Proposed Design

Retire `legacydomain` in dependency order:

1. Establish baseline inventory, knowledge impact routes, and future guardrails.
2. Add confirmed owner contracts for application lifecycle/snapshot and identity authorization where existing public RPCs are insufficient.
3. Replace Offer legacy adapters with local persistence, outbox, and owner clients; delete Offer `legacydomain`.
4. Replace Interview legacy adapters with local persistence, outbox, staff/application/authorization clients; delete Interview `legacydomain`.
5. Replace Recruitment legacy runtime graph with local DDD application and infrastructure adapters; delete Recruitment `legacydomain`.
6. Replace AI Agent legacy runtime graph with local application, provider, MCP, MQ, persistence, and gRPC adapters; delete AI Agent `legacydomain`.
7. Enforce final guardrails and update `.knowledge` references.

## 4. Data Structure Changes

No database schema changes are planned. GORM record structs needed for existing tables move into service-local `internal/infrastructure/persistence` packages as private or package-local persistence records. Domain models remain free of GORM/protobuf tags.

## 5. API and Interface Changes

Public frontend and gateway behavior remains unchanged.

Internal service contracts may be added only in the confirmed owner-contract TASK:

- Recruitment application snapshot and lifecycle transition contract for Offer/Interview.
- Identity principal/authorization/scope contract for service authorizers.

Service-local ports to standardize:

- `ApplicationSnapshotReader`
- `ApplicationLifecycle`
- `Authorizer`
- `OutboxPublisher`
- `OwnerSnapshotClient`

## 6. Algorithm or Workflow Changes

Runtime construction changes from legacy service graphs to local dependency graphs. Business rules already expressed in domain/application layers must remain there. Infrastructure adapters handle SQL, OSS, MQ, provider SDK, MCP, and owner-service clients.

Cross-owner writes are replaced by owner contracts or events. Cross-owner reads use explicit clients/read models rather than copied repositories.

## 7. Configuration Design

No new configuration keys are planned. Existing MySQL, Redis, RabbitMQ, OSS, AI provider, Nacos, health, metrics, trace, and serviceconfig wiring must remain compatible.

## 8. Compatibility Strategy

Each service cutover preserves protobuf response shapes, error codes/messages, outbox routing, status keys, permission keys, and local startup behavior. Where new internal RPCs are introduced, they are additive and must not change existing public RPC semantics.

## 9. Error Handling and Fallback Design

Owner-client failures return existing compatible service errors. AI provider and embedding fallback behavior remains explicit. MCP policy failures continue to deny unsafe operations. MQ consumers remain idempotent and keep graceful shutdown behavior.

## 10. Observability and Debug Output Design

Use existing logging and health conventions. Do not add noisy logs or log secrets/PII. TASK reports and evidence are the audit trail for migration progress, validation, and knowledge impact.

## 11. Testing Strategy

Run service-local `go test ./...` for each touched service. Run backend boundary, MySQL table ownership, knowledge validation, knowledge reference checks, task scope, and agent checks for each implementation TASK.

Add focused tests when replacing adapters:

- Offer/Interview persistence and gRPC compatibility.
- Recruitment service-surface adapters and application lifecycle.
- AI Agent provider, MCP, embedding, stream, worker, and gRPC compatibility.

## 12. Migration Risks

- Cross-owner lifecycle updates can regress application status or transition audit.
- Authz/scope replacement can accidentally weaken permissions.
- AI Agent streaming, long-task worker, and provider fallback paths are high risk.
- Knowledge files can drift if path references are not updated in the same TASK.

## 13. Implementation Boundaries

Do not edit business code during feature initialization. During implementation, obey one TASK scope at a time. `.knowledge/**` is allowed in every TASK so routed active knowledge can be updated. Public API, schema, auth/security, dependency, lockfile, and global config changes are hard stops unless the TASK explicitly requires confirmation.

## 14. Alternatives Considered

- Rename `legacydomain` to `compatibility`: rejected because it hides debt.
- Delete all legacy directories first: rejected because active runtime depends on them.
- Keep knowledge updates for a final task only: rejected because active knowledge would route agents to stale paths during migration.

## 15. Assumptions Requiring Confirmation

- New internal protobuf/service contracts are acceptable only in the owner-contract TASK after explicit confirmation.
- No schema changes are needed to retire `legacydomain`.

## 16. Open Questions

- Exact internal Recruitment and Identity contract shapes must be finalized in TASK-002.
