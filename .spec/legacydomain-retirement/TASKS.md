# TASKS - legacydomain-retirement

## Task Overview

| TASK | Title | Status | Scope | Acceptance |
|------|-------|--------|-------|------------|
| TASK-001 | Legacy baseline, knowledge routes, and guardrail staging | pending | scripts, knowledge, spec docs only | acceptance/TASK-001.md |
| TASK-002 | Owner contracts for lifecycle and authorization | pending | proto/contracts, recruitment, identity, gateway clients, knowledge | acceptance/TASK-002.md |
| TASK-003 | Retire Offer legacydomain | pending | offer service, knowledge | acceptance/TASK-003.md |
| TASK-004 | Retire Interview legacydomain | pending | interview service, knowledge | acceptance/TASK-004.md |
| TASK-005 | Recruitment native runtime cutover | pending | recruitment service, knowledge | acceptance/TASK-005.md |
| TASK-006 | Retire Recruitment legacydomain | pending | recruitment service, scripts, knowledge | acceptance/TASK-006.md |
| TASK-007 | AI Agent native runtime cutover | pending | ai-agent service, knowledge | acceptance/TASK-007.md |
| TASK-008 | Retire AI Agent legacydomain | pending | ai-agent service, scripts, knowledge | acceptance/TASK-008.md |
| TASK-009 | Final legacydomain enforcement and documentation convergence | pending | scripts, docs, knowledge, reports | acceptance/TASK-009.md |

## TASK-001 - Legacy baseline, knowledge routes, and guardrail staging

### Goal

Create a current inventory of all `legacydomain` code, active imports, knowledge references, and validation gaps; stage guardrail changes that report, but do not yet fail, remaining legacy usage.

### Scope

No business code changes.

### Allowed Files

- `.spec/legacydomain-retirement/docs/**`
- `.spec/legacydomain-retirement/reports/**`
- `.spec/legacydomain-retirement/**`
- `.knowledge/**`
- `scripts/check-backend-boundaries.mjs`
- `scripts/check-mysql-table-ownership.mjs`

### Forbidden Files

- `smart-recruit-*-service/**`
- `smart-recruit-proto/**`
- `db.sql`
- `go.mod`
- `go.sum`
- `go.work`
- `go.work.sum`
- `package.json`
- `pnpm-lock.yaml`

### Dependencies

None.

### Acceptance Criteria

- Baseline inventory records current legacy files, non-test import sites, runtime entrypoints, and knowledge references.
- Guardrail scripts can report legacy usage without breaking current repository checks before implementation tasks retire it.
- Routed active `.knowledge` files are reviewed and updated or marked with explicit debt.

### Required Tests

- `node scripts/check-backend-boundaries.mjs`
- `node scripts/check-mysql-table-ownership.mjs`
- `node .knowledge/scripts/validate-knowledge.mjs --root .`
- `node .knowledge/scripts/check-references.mjs --root .`

### Risks

Guardrails must not make the current baseline impossible to work from before retirement tasks run.

### Notes

This task prepares evidence and checks only.

## TASK-002 - Owner contracts for lifecycle and authorization

### Goal

Add explicit owner contracts needed to replace copied cross-owner repositories for application lifecycle/snapshot and identity authorization/principal checks.

### Scope

Additive internal contracts only. No frontend or gateway public route changes.

### Allowed Files

- `smart-recruit-proto/**`
- `smart-recruit-recruitment-service/**`
- `smart-recruit-identity-service/**`
- `smart-recruit-gateway/rpc/**`
- `.knowledge/**`
- `.spec/legacydomain-retirement/reports/**`

### Forbidden Files

- `db.sql`
- `smart-recruit-*-service/internal/legacydomain/**`
- `package.json`
- `pnpm-lock.yaml`
- `go.work`
- `go.work.sum`

### Dependencies

TASK-001.

### Acceptance Criteria

- Internal owner contracts cover application snapshot/lifecycle needs for Offer and Interview.
- Internal owner contracts cover authorization/principal/scope needs for service authorizers.
- Existing public RPC behavior remains compatible.
- Protobuf/generated code changes, if required, are additive and documented.

### Required Tests

- `go test ./...` in `smart-recruit-proto`
- `go test ./...` in `smart-recruit-recruitment-service`
- `go test ./...` in `smart-recruit-identity-service`
- `go test ./...` in `smart-recruit-gateway`
- `node scripts/check-backend-boundaries.mjs`

### Risks

This task touches shared contracts and auth-sensitive behavior; it requires human confirmation before implementation.

### Notes

Do not use existing public HR-facing application status RPC if it cannot safely express internal conditional lifecycle transitions.

## TASK-003 - Retire Offer legacydomain

### Goal

Replace Offer legacy repositories/models with local infrastructure persistence, outbox, and owner clients, then delete `smart-recruit-offer-service/internal/legacydomain`.

### Scope

Offer service only plus knowledge updates.

### Allowed Files

- `smart-recruit-offer-service/**`
- `.knowledge/**`
- `.spec/legacydomain-retirement/reports/**`

### Forbidden Files

- `smart-recruit-proto/**`
- `smart-recruit-interview-service/**`
- `smart-recruit-recruitment-service/**`
- `smart-recruit-ai-agent-service/**`
- `db.sql`
- `package.json`
- `pnpm-lock.yaml`

### Dependencies

TASK-002.

### Acceptance Criteria

- Offer non-test code has no `legacydomain` imports.
- Offer `internal/legacydomain` directory is deleted.
- Offer owner tables use local persistence records/adapters.
- Outbox and application lifecycle/snapshot/authz dependencies use explicit adapters.
- Offer tests pass.

### Required Tests

- `go test ./...` in `smart-recruit-offer-service`
- `node scripts/check-backend-boundaries.mjs`
- `node scripts/check-mysql-table-ownership.mjs`

### Risks

Offer lifecycle changes can regress application status transitions.

### Notes

Do not change Offer protobuf behavior.

## TASK-004 - Retire Interview legacydomain

### Goal

Replace Interview legacy repositories/models with local infrastructure persistence, outbox, staff/application/authorization clients, then delete `smart-recruit-interview-service/internal/legacydomain`.

### Scope

Interview service only plus knowledge updates.

### Allowed Files

- `smart-recruit-interview-service/**`
- `.knowledge/**`
- `.spec/legacydomain-retirement/reports/**`

### Forbidden Files

- `smart-recruit-proto/**`
- `smart-recruit-offer-service/**`
- `smart-recruit-recruitment-service/**`
- `smart-recruit-ai-agent-service/**`
- `db.sql`
- `package.json`
- `pnpm-lock.yaml`

### Dependencies

TASK-003.

### Acceptance Criteria

- Interview non-test code has no `legacydomain` imports.
- Interview `internal/legacydomain` directory is deleted.
- Interview owner tables use local persistence records/adapters.
- Listing/detail, feedback, outbox, staff directory, lifecycle, and authz behavior remains compatible.
- Interview tests pass.

### Required Tests

- `go test ./...` in `smart-recruit-interview-service`
- `node scripts/check-backend-boundaries.mjs`
- `node scripts/check-mysql-table-ownership.mjs`

### Risks

Interview listing and candidate-sensitive field filtering are easy to regress.

### Notes

Do not change Interview protobuf behavior.

## TASK-005 - Recruitment native runtime cutover

### Goal

Replace Recruitment runtime construction of legacy service graphs with local application, infrastructure, and interface adapters.

### Scope

Recruitment service plus knowledge updates.

### Allowed Files

- `smart-recruit-recruitment-service/**`
- `.knowledge/**`
- `.spec/legacydomain-retirement/reports/**`

### Forbidden Files

- `smart-recruit-proto/**`
- `smart-recruit-offer-service/**`
- `smart-recruit-interview-service/**`
- `smart-recruit-ai-agent-service/**`
- `db.sql`
- `package.json`
- `pnpm-lock.yaml`

### Dependencies

TASK-004.

### Acceptance Criteria

- `cmd/recruitment-service` no longer constructs legacy service graph for active runtime.
- Job, candidate, application, collaboration, taxonomy/admin, usage surfaces are backed by local services/adapters.
- Existing public Recruitment protobuf behavior remains compatible.
- Recruitment tests pass.

### Required Tests

- `go test ./...` in `smart-recruit-recruitment-service`
- `node scripts/check-backend-boundaries.mjs`
- `node scripts/check-mysql-table-ownership.mjs`

### Risks

This is a large behavioral cutover and may need narrower follow-up split if one execution cycle becomes too large.

### Notes

Do not delete `legacydomain` until TASK-006.

## TASK-006 - Retire Recruitment legacydomain

### Goal

Remove remaining Recruitment `legacydomain` files, tests, and script exceptions after active runtime no longer depends on them.

### Scope

Recruitment service, guardrail scripts, knowledge.

### Allowed Files

- `smart-recruit-recruitment-service/**`
- `scripts/check-backend-boundaries.mjs`
- `scripts/check-mysql-table-ownership.mjs`
- `.knowledge/**`
- `.spec/legacydomain-retirement/reports/**`

### Forbidden Files

- `smart-recruit-proto/**`
- `smart-recruit-offer-service/**`
- `smart-recruit-interview-service/**`
- `smart-recruit-ai-agent-service/**`
- `db.sql`
- `package.json`
- `pnpm-lock.yaml`

### Dependencies

TASK-005.

### Acceptance Criteria

- Recruitment non-test code has no `legacydomain` imports.
- Recruitment `internal/legacydomain` directory is deleted.
- Guardrails no longer treat Recruitment legacy paths as allowed scan roots.
- Recruitment tests pass.

### Required Tests

- `go test ./...` in `smart-recruit-recruitment-service`
- `node scripts/check-backend-boundaries.mjs`
- `node scripts/check-mysql-table-ownership.mjs`

### Risks

Deleting copied repositories may reveal hidden test or tooling references.

### Notes

Keep table ownership checks aligned with new persistence paths.

## TASK-007 - AI Agent native runtime cutover

### Goal

Replace AI Agent legacy service graph with native application, provider, MCP, MQ, persistence, cache, client, and gRPC adapters.

### Scope

AI Agent service plus knowledge updates.

### Allowed Files

- `smart-recruit-ai-agent-service/**`
- `.knowledge/**`
- `.spec/legacydomain-retirement/reports/**`

### Forbidden Files

- `smart-recruit-proto/**`
- `smart-recruit-recruitment-service/**`
- `smart-recruit-offer-service/**`
- `smart-recruit-interview-service/**`
- `db.sql`
- `package.json`
- `pnpm-lock.yaml`

### Dependencies

TASK-006.

### Acceptance Criteria

- `cmd/ai-agent-service` no longer constructs legacy service graph for active runtime.
- `interfaces/grpc/legacy_servers.go` is replaced by native gRPC adapters.
- Chat, candidate chat, agent run, prompt/config, MCP, skill, embedding, recruiting intelligence, and workers remain compatible.
- AI Agent tests pass with fake/env-gated provider behavior.

### Required Tests

- `go test ./...` in `smart-recruit-ai-agent-service`
- `node scripts/check-backend-boundaries.mjs`
- `node scripts/check-mysql-table-ownership.mjs`

### Risks

Streaming, provider fallback, worker idempotency, MCP safety, and candidate-sensitive data handling are high risk.

### Notes

Do not delete `legacydomain` until TASK-008.

## TASK-008 - Retire AI Agent legacydomain

### Goal

Remove remaining AI Agent `legacydomain` files, tests, and script exceptions after native runtime cutover.

### Scope

AI Agent service, guardrail scripts, knowledge.

### Allowed Files

- `smart-recruit-ai-agent-service/**`
- `scripts/check-backend-boundaries.mjs`
- `scripts/check-mysql-table-ownership.mjs`
- `.knowledge/**`
- `.spec/legacydomain-retirement/reports/**`

### Forbidden Files

- `smart-recruit-proto/**`
- `smart-recruit-recruitment-service/**`
- `smart-recruit-offer-service/**`
- `smart-recruit-interview-service/**`
- `db.sql`
- `package.json`
- `pnpm-lock.yaml`

### Dependencies

TASK-007.

### Acceptance Criteria

- AI Agent non-test code has no `legacydomain` imports.
- AI Agent `internal/legacydomain` directory is deleted.
- Knowledge references to AI Agent legacy paths are updated.
- AI Agent tests pass.

### Required Tests

- `go test ./...` in `smart-recruit-ai-agent-service`
- `node scripts/check-backend-boundaries.mjs`
- `node scripts/check-mysql-table-ownership.mjs`

### Risks

Knowledge source references may need broad but targeted updates.

### Notes

Keep shared `smart-recruit-commons/ai` only if it remains a true generic provider helper.

## TASK-009 - Final legacydomain enforcement and documentation convergence

### Goal

Make legacydomain absence a hard validation rule, remove temporary scan exceptions, update final documentation/knowledge, and produce final retirement evidence.

### Scope

Scripts, knowledge, feature docs, reports.

### Allowed Files

- `scripts/check-backend-boundaries.mjs`
- `scripts/check-mysql-table-ownership.mjs`
- `.knowledge/**`
- `.spec/legacydomain-retirement/docs/**`
- `.spec/legacydomain-retirement/reports/**`

### Forbidden Files

- `smart-recruit-*-service/internal/legacydomain/**`
- `smart-recruit-proto/**`
- `db.sql`
- `package.json`
- `pnpm-lock.yaml`

### Dependencies

TASK-008.

### Acceptance Criteria

- Repository scan finds no target service `internal/legacydomain` directories.
- Repository scan finds no non-test `legacydomain` imports.
- Boundary checks fail on future legacydomain reintroduction.
- Knowledge validation and reference checks pass.
- Final report records all retired roots and remaining non-legacy debt, if any.

### Required Tests

- `find smart-recruit-*-service -path '*/internal/legacydomain' -type d -print`
- `rg -n 'legacydomain' smart-recruit-*-service scripts .knowledge -g '*.go' -g '*.md' -g '*.mjs'`
- `node scripts/check-backend-boundaries.mjs`
- `node scripts/check-mysql-table-ownership.mjs`
- `node .knowledge/scripts/validate-knowledge.mjs --root .`
- `node .knowledge/scripts/check-references.mjs --root .`

### Risks

Final enforcement can expose stale documentation outside routed active knowledge.

### Notes

Do not start this task until all four legacy roots are gone.
