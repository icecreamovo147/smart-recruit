# Legacydomain Migration Finalization SPEC

## 1. Background

The previous `legacydomain-retirement` feature removed service-local `internal/legacydomain` directories from the target services and stopped active runtimes from importing those packages. Runtime follow-up analysis found two remaining migration-quality gaps:

- `smart-recruit-ai-agent-service` is functional only through a broad native gRPC/store adapter. Several public or internal gRPC methods are still inherited from `Unimplemented...Server` or return `success` with empty data.
- `smart-recruit-recruitment-service` no longer depends on `legacydomain`, but its active runtime is still backed by one monolithic `internal/infrastructure/persistence/native_adapters.go` bundle that mixes job, taxonomy, candidate, application, collaboration, admin, usage, outbox, and owner-contract responsibilities.

This feature finalizes the migration state for Recruitment and AI Agent without changing frontend routes, gateway public HTTP contracts, protobuf public API shape, or database schema.

## 2. Goals

- Make AI Agent runtime use explicit application services and infrastructure adapters for AI-owned capabilities instead of native empty or partially implemented gRPC adapters.
- Ensure AI Agent runtime does not silently return `success` for unconfigured stores, missing providers, unimplemented configuration behavior, or unsupported governance APIs.
- Implement or explicitly fail AI Agent gRPC methods that are reachable through Gateway or active internal callers.
- Refine Recruitment's post-legacy runtime by splitting the monolithic native bundle into explicit bounded adapters/services that match the runtime dependency interfaces.
- Preserve the existing no-`legacydomain` guardrails and strengthen checks against reintroducing broad native catch-all adapters as long-term runtime owners.
- Keep `.knowledge/**` aligned with the finalized architecture and require `knowledge_impact` reporting for every implementation TASK.

## 3. Non-Goals

- Do not reintroduce `internal/legacydomain` in any service.
- Do not rename `legacydomain` to another package name while preserving the same broad shared-domain pattern.
- Do not change frontend-visible behavior except replacing empty/unimplemented responses with working behavior or explicit errors.
- Do not change Gateway route paths, permission names, public HTTP request/response shapes, or user-facing navigation.
- Do not change database schema or migration files.
- Do not add new external dependencies.
- Do not move ownership of Recruitment, AI Agent, Identity, Interview, Offer, Notification, or Analytics tables.

## 4. User-Facing Behavior

- Existing HR admin pages for LLM providers/models, embedding providers/models, prompt templates, agent configs, MCP servers/policies/logs, Skill registry, and Agent Skills must continue to load through the current Gateway routes.
- LLM provider connection testing must execute real provider validation or return an explicit configuration/runtime error; it must not return gRPC `Unimplemented`.
- AI configuration lists must continue to return database-backed data when records exist.
- Actions that are not supported in the current runtime must return explicit errors instead of successful empty responses.
- Recruitment user workflows for jobs, candidates, resumes, applications, taxonomy/admin, collaboration, invite codes, usage reports, and lifecycle owner contracts must remain behavior-compatible.

## 5. Functional Requirements

1. AI Agent runtime must no longer register native gRPC services whose methods are mostly inherited from `Unimplemented...Server`.
2. AI Agent gRPC methods reachable from Gateway routes must be implemented through application services or explicit infrastructure/provider adapters.
3. AI Agent methods that cannot be implemented without a schema or public API change must return clear non-success responses and be documented as gaps in TASK evidence.
4. AI Agent store/provider dependency absence must fail fast during runtime construction or return explicit internal configuration errors; it must not produce success with empty data.
5. Recruitment runtime must keep using explicit `runtime.Deps`, but each dependency should be backed by a focused local application/infrastructure adapter instead of one broad native adapter owning all contexts.
6. Recruitment GORM records must remain private to `internal/infrastructure/persistence`.
7. Cross-context reads/writes must remain behind owner contracts, application ports, events, or explicit read-model adapters.
8. Boundary checks must continue to fail on any `internal/legacydomain` reintroduction.
9. Boundary or harness checks must detect newly introduced empty-success runtime stubs for AI Agent and Recruitment.
10. Each TASK must review and, when scope permits, update routed `.knowledge/**` documents.

## 6. Non-Functional Requirements

- Keep changes incremental and independently reviewable by TASK.
- Preserve compile-time safety and avoid `any`-style bypasses.
- Use `gofmt` for Go changes.
- Avoid broad mechanical formatting.
- Prefer existing repository patterns and standard library facilities.
- Keep runtime startup failures explicit and diagnosable.

## 7. Compatibility Requirements

- Gateway HTTP routes and permission middleware behavior must remain compatible.
- Existing protobuf RPC names and message fields must remain compatible. Adding internal-only contracts is out of scope unless separately confirmed.
- Existing MySQL schema and table ownership manifest must remain compatible.
- Existing local development scripts must continue to work.
- Existing service module tests must continue to pass for touched modules.

## 8. Observability and Debug Requirements

- Connection tests and provider/MCP/embedding failures must preserve meaningful error details without leaking secrets.
- Replaced empty stubs must produce logs or response messages that identify missing configuration, unsupported behavior, or provider failure.
- TASK reports must include `knowledge_impact` with reviewed documents, updated paths, stale/conflict findings, and coverage gaps.

## 9. Error Handling and Fallback Requirements

- Do not swallow provider, database, MCP, embedding, or repository errors.
- Do not return `Code: 0` for unsupported, unimplemented, or unconfigured behavior.
- Secrets must remain redacted in list/detail/test responses.
- Existing graceful degradation for optional runtime components may remain only when the response clearly communicates degraded behavior.

## 10. Security and Safety Requirements

- Do not expose API keys, access tokens, private headers, candidate sensitive data, or raw resumes in logs or responses.
- Preserve Gateway and service-level auth/RBAC expectations.
- Provider and MCP connectivity tests must follow existing private-network, command allowlist, and redaction policies where applicable.
- Knowledge updates must not include live secrets, raw local database data, or raw logs.

## 11. Acceptance Criteria

- `smart-recruit-ai-agent-service` has no active runtime path that returns `success` for unimplemented Gateway-reachable methods.
- `TestProviderConnection` no longer returns gRPC `Unimplemented`.
- AI Agent configuration/governance list and mutation methods are either implemented or explicitly return non-success unsupported/configuration responses with tests.
- Recruitment runtime no longer depends on one catch-all `nativeAdapter` implementing unrelated bounded APIs.
- No `internal/legacydomain` directories exist in `smart-recruit-recruitment-service` or `smart-recruit-ai-agent-service`.
- Non-test Go code has no `legacydomain` import.
- Boundary, table ownership, knowledge validation, and relevant Go tests pass.
- `.knowledge` active documents no longer describe AI Agent or Recruitment as being in an incomplete native-adapter interim state after the relevant TASKs complete.

## 12. Out of Scope

- Frontend redesign.
- New database migrations.
- Public API redesign.
- Multi-database split.
- New provider SDK dependencies.
- Reworking Interview, Offer, Identity, Notification, Analytics, or Worker services beyond guardrail checks and knowledge references.

## 13. Assumptions Requiring Confirmation

- Existing database tables contain enough columns to implement the currently exposed AI configuration and governance APIs without schema changes.
- Existing shared AI/MCP/provider helpers are sufficient for connection tests without adding dependencies.
- Recruitment's current behavior in `native_adapters.go` is the compatibility baseline until focused adapters replace it.

## 14. Open Questions

- Should unsupported AI Agent management APIs return a platform-standard non-zero application code or a gRPC error for admin pages?
- Should AI Agent provider connection tests perform a real model call, a lightweight credential validation, or both?
- Which Recruitment contexts should be extracted first if implementation risk requires further subdivision?
