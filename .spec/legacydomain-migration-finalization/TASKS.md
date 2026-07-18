# TASKS - legacydomain-migration-finalization

## Task Overview

| TASK | Title | Status | Scope | Acceptance |
|------|-------|--------|-------|------------|
| TASK-001 | Baseline finalization guardrails | pending | Guard scripts, inventory docs, knowledge review | acceptance/TASK-001.md |
| TASK-002 | Refine Recruitment runtime adapters | pending | Recruitment runtime wiring and focused adapters | acceptance/TASK-002.md |
| TASK-003 | Implement AI Agent configuration services | pending | LLM, embedding, prompt, agent config application paths | acceptance/TASK-003.md |
| TASK-004 | Implement AI Agent MCP and Skill governance services | pending | MCP, Skill registry, Agent Skill application paths | acceptance/TASK-004.md |
| TASK-005 | Complete AI Agent runtime and recruiting intelligence behavior | pending | Chat/agent run fallbacks, provider wiring, recruiting intelligence | acceptance/TASK-005.md |
| TASK-006 | Final verification and knowledge convergence | pending | Boundary hardening, final docs, reports | acceptance/TASK-006.md |

## TASK-001 - Baseline finalization guardrails

### Goal

Record the post-commit baseline for Recruitment and AI Agent finalization, and extend guardrails so later TASKs cannot hide unimplemented runtime behavior behind success responses.

### Scope

Inventory current AI Agent empty/unimplemented methods, Recruitment native adapter responsibilities, and relevant knowledge routes. Add or extend scripts only for detection and reporting.

### Allowed Files

- `.spec/legacydomain-migration-finalization/**`
- `scripts/check-backend-boundaries.mjs`
- `scripts/check-mysql-table-ownership.mjs`
- `.knowledge/**`

### Forbidden Files

- `smart-recruit-*/**/*.go`
- `smart-recruit-proto/**`
- frontend apps
- migrations and `db.sql`
- package manifests and lockfiles

### Dependencies

None.

### Acceptance Criteria

- Baseline doc lists all active AI Agent empty/unimplemented runtime methods discovered from current code.
- Baseline doc lists Recruitment native adapter responsibility groups and extraction targets.
- Boundary checks detect `internal/legacydomain` reintroduction and newly introduced empty-success runtime stubs for targeted services.
- `knowledge_impact` is reported.

### Required Tests

- `node scripts/check-backend-boundaries.mjs`
- `node scripts/check-mysql-table-ownership.mjs`
- `node .knowledge/scripts/validate-knowledge.mjs --root .`
- `node .knowledge/scripts/check-references.mjs --root .`
- `bash .spec/legacydomain-migration-finalization/scripts/check-task-scope.sh TASK-001`
- `bash .spec/legacydomain-migration-finalization/scripts/agent-check.sh`

### Risks

Guardrails may need allowlists for legitimate empty list responses.

### Notes

Do not implement runtime behavior in this TASK.

## TASK-002 - Refine Recruitment runtime adapters

### Goal

Split Recruitment's catch-all native adapter into focused adapters/services while preserving runtime behavior and `runtime.Deps`.

### Scope

Refactor `smart-recruit-recruitment-service/internal/infrastructure/persistence/native_adapters.go` into focused files/types for job, taxonomy, candidate/resume, application lifecycle, application-owner contract, collaboration, admin/invite, usage, and outbox responsibilities.

### Allowed Files

- `smart-recruit-recruitment-service/cmd/recruitment-service/main.go`
- `smart-recruit-recruitment-service/internal/runtime/**`
- `smart-recruit-recruitment-service/internal/application/**`
- `smart-recruit-recruitment-service/internal/domain/**`
- `smart-recruit-recruitment-service/internal/infrastructure/**`
- `smart-recruit-recruitment-service/internal/interfaces/**`
- `smart-recruit-recruitment-service/internal/docs/**`
- `.knowledge/**`
- `.spec/legacydomain-migration-finalization/**`

### Forbidden Files

- `smart-recruit-proto/**`
- Gateway and frontend public routes
- migrations and `db.sql`
- package manifests and lockfiles

### Dependencies

TASK-001.

### Acceptance Criteria

- Active Recruitment runtime no longer uses one catch-all `nativeAdapter` type for unrelated APIs.
- Focused adapters preserve existing SQL behavior and lifecycle side effects.
- `go test ./...` passes in `smart-recruit-recruitment-service`.
- No `legacydomain` import or directory is reintroduced.
- `knowledge_impact` is reported and relevant Recruitment knowledge is updated if stale.

### Required Tests

- `cd smart-recruit-recruitment-service && go test ./...`
- common Harness and knowledge checks.

### Risks

Large-file extraction can change behavior if query helpers or transaction boundaries move incorrectly.

### Notes

Prefer extraction with parity tests over behavior redesign.

## TASK-003 - Implement AI Agent configuration services

### Goal

Replace native AI configuration gRPC stubs with application services for LLM providers/models, embedding providers/models, prompt templates, and agent configs.

### Scope

Implement database-backed list/create/update/delete/test/default/render/version/capability behavior where supported by current schema and helpers. Return explicit non-success errors for unsupported paths discovered during implementation.

### Allowed Files

- `smart-recruit-ai-agent-service/cmd/ai-agent-service/main.go`
- `smart-recruit-ai-agent-service/internal/runtime/**`
- `smart-recruit-ai-agent-service/internal/application/**`
- `smart-recruit-ai-agent-service/internal/domain/**`
- `smart-recruit-ai-agent-service/internal/infrastructure/**`
- `smart-recruit-ai-agent-service/internal/interfaces/**`
- `.knowledge/**`
- `.spec/legacydomain-migration-finalization/**`

### Forbidden Files

- `smart-recruit-proto/**`
- Gateway route definitions
- frontend apps
- migrations and `db.sql`
- package manifests and lockfiles

### Dependencies

TASK-001.

### Acceptance Criteria

- `LlmConfigService.TestProviderConnection` no longer returns gRPC `Unimplemented`.
- LLM, embedding, prompt, and agent config Gateway-reachable methods do not silently return successful empty stubs.
- Secrets remain redacted.
- `go test ./...` passes in `smart-recruit-ai-agent-service`.
- `knowledge_impact` is reported and relevant AI configuration knowledge is updated if stale.

### Required Tests

- `cd smart-recruit-ai-agent-service && go test ./...`
- common Harness and knowledge checks.

### Risks

Provider test semantics may require careful distinction between credential validation and real model calls.

### Notes

Do not change Gateway or frontend API shapes.

## TASK-004 - Implement AI Agent MCP and Skill governance services

### Goal

Replace native MCP, Skill registry, and Agent Skill stubs with real application paths or explicit non-success unsupported responses.

### Scope

Implement MCP servers/policies/logs/test/tools/call behavior, Skill registry CRUD/version/tool behavior, and Agent Skill detail/version/status/preview/debug behavior where current schema supports it.

### Allowed Files

- `smart-recruit-ai-agent-service/internal/application/**`
- `smart-recruit-ai-agent-service/internal/domain/**`
- `smart-recruit-ai-agent-service/internal/infrastructure/**`
- `smart-recruit-ai-agent-service/internal/interfaces/**`
- `.knowledge/**`
- `.spec/legacydomain-migration-finalization/**`

### Forbidden Files

- `smart-recruit-proto/**`
- Gateway route definitions
- frontend apps
- migrations and `db.sql`
- package manifests and lockfiles

### Dependencies

TASK-001 and TASK-003.

### Acceptance Criteria

- `ListMCPToolPolicies`, `ListMCPToolLogs`, and `ListSkills` no longer return unconditional empty success.
- MCP and Skill admin routes receive implemented or explicit non-success responses.
- Policy/log redaction and sensitive-data safety are preserved.
- `go test ./...` passes in `smart-recruit-ai-agent-service`.
- `knowledge_impact` is reported.

### Required Tests

- `cd smart-recruit-ai-agent-service && go test ./...`
- common Harness and knowledge checks.

### Risks

MCP test/tool execution can touch network or local command boundaries; keep tests deterministic.

### Notes

Avoid adding new provider or MCP dependencies.

## TASK-005 - Complete AI Agent runtime and recruiting intelligence behavior

### Goal

Remove remaining AI Agent native runtime fallback behavior that returns success for missing store/provider/recruiting intelligence work.

### Scope

Harden AI chat/session/agent-run store/provider requirements, implement or explicitly fail recruiting intelligence methods, and ensure runtime construction fails fast for required missing dependencies.

### Allowed Files

- `smart-recruit-ai-agent-service/cmd/ai-agent-service/main.go`
- `smart-recruit-ai-agent-service/internal/runtime/**`
- `smart-recruit-ai-agent-service/internal/application/**`
- `smart-recruit-ai-agent-service/internal/domain/**`
- `smart-recruit-ai-agent-service/internal/infrastructure/**`
- `smart-recruit-ai-agent-service/internal/interfaces/**`
- `.knowledge/**`
- `.spec/legacydomain-migration-finalization/**`

### Forbidden Files

- `smart-recruit-proto/**`
- Gateway route definitions
- frontend apps
- migrations and `db.sql`
- package manifests and lockfiles

### Dependencies

TASK-003 and TASK-004.

### Acceptance Criteria

- AI Agent runtime has no active `store == nil` success fallback for database-backed operations.
- `CompareCandidatesForJob` does not return unconditional success with an empty payload.
- Missing provider/configuration failures are explicit and test-covered.
- `go test ./...` passes in `smart-recruit-ai-agent-service`.
- `knowledge_impact` is reported.

### Required Tests

- `cd smart-recruit-ai-agent-service && go test ./...`
- common Harness and knowledge checks.

### Risks

Some recruiting intelligence behavior may expose missing cross-service read models; report Hard Stop if a public API or schema change is required.

### Notes

Do not change public protobuf contracts.

## TASK-006 - Final verification and knowledge convergence

### Goal

Verify Recruitment and AI Agent have reached the final post-legacydomain migration state and update active knowledge accordingly.

### Scope

Run final checks, remove obsolete finalization docs if applicable, update knowledge references, and ensure guardrails describe the final architecture.

### Allowed Files

- `.spec/legacydomain-migration-finalization/**`
- `.knowledge/**`
- `scripts/check-backend-boundaries.mjs`
- `scripts/check-mysql-table-ownership.mjs`

### Forbidden Files

- service runtime/business code
- `smart-recruit-proto/**`
- frontend apps
- migrations and `db.sql`
- package manifests and lockfiles

### Dependencies

TASK-001 through TASK-005.

### Acceptance Criteria

- No targeted service contains `internal/legacydomain`.
- Non-test code has no `legacydomain` import.
- Guardrails pass.
- Knowledge validation and reference checks pass.
- Final report states whether the feature can move to pipeline completion.

### Required Tests

- `node scripts/check-backend-boundaries.mjs`
- `node scripts/check-mysql-table-ownership.mjs`
- `node .knowledge/scripts/validate-knowledge.mjs --root .`
- `node .knowledge/scripts/check-references.mjs --root .`
- common Harness checks.

### Risks

Final checks may reveal stale knowledge outside this feature scope; report rather than silently widening scope.

### Notes

No business code changes in this TASK.
