# TASKS - hr-agent-resumable-stream

Execute tasks in order. Each TASK must pass its acceptance file and Harness checks before the next TASK starts. Any TASK marked `requiresHumanConfirmation: true` must receive explicit user confirmation before implementation begins.

## Task Overview

| TASK | Title | Status | Scope | Acceptance |
|------|-------|--------|-------|------------|
| TASK-HARS-001 | Persistence Schema And Model Alignment | pending | schema, model, db baseline | acceptance/TASK-HARS-001.md |
| TASK-HARS-002 | Run State And Event Repository | pending | logic repository and state helpers | acceptance/TASK-HARS-002.md |
| TASK-HARS-003 | Proto Contract For Durable Runs | pending | mirrored proto and generated code | acceptance/TASK-HARS-003.md |
| TASK-HARS-004 | Durable Run Worker And Logic Service | pending | logic service runtime | acceptance/TASK-HARS-004.md |
| TASK-HARS-005 | Gateway REST And SSE Endpoints | pending | web-gin handlers/routes/rpc | acceptance/TASK-HARS-005.md |
| TASK-HARS-006 | Frontend Run API Reducer And Composable | pending | HR frontend API/types/runtime | acceptance/TASK-HARS-006.md |
| TASK-HARS-007 | Migrate HR Agent View Entry Points | pending | HR chat view migration | acceptance/TASK-HARS-007.md |
| TASK-HARS-008 | Refresh Recovery And End-To-End Validation | pending | cross-stack recovery validation | acceptance/TASK-HARS-008.md |

## TASK-HARS-001 - Persistence Schema And Model Alignment

Status: pending

Requires human confirmation: true

Purpose:

Create the durable persistence foundation for resumable HR Agent runs.

Scope:

- Add forward and rollback migrations for run events, active run pointers, idempotency, snapshots, and run-history association.
- Align `db.sql` with the new schema.
- Align Go models with the schema.
- Add focused persistence test helpers if needed.

Acceptance:

- `.spec/hr-agent-resumable-stream/acceptance/TASK-HARS-001.md`

## TASK-HARS-002 - Run State And Event Repository

Status: pending

Requires human confirmation: false

Purpose:

Implement backend state transition and event replay primitives after the schema exists.

Scope:

- Add run event repository operations.
- Add state transition helper tests.
- Extend existing run repository behavior only where required for durable active runs.

Acceptance:

- `.spec/hr-agent-resumable-stream/acceptance/TASK-HARS-002.md`

## TASK-HARS-003 - Proto Contract For Durable Runs

Status: pending

Requires human confirmation: true

Purpose:

Define the logic-grpc contract for create, get, active lookup, event subscription, cancel, and confirm operations.

Scope:

- Update mirrored recruitment proto files.
- Regenerate and align generated Go code in both services.
- Add compile-level checks where the repository already supports them.

Acceptance:

- `.spec/hr-agent-resumable-stream/acceptance/TASK-HARS-003.md`

## TASK-HARS-004 - Durable Run Worker And Logic Service

Status: pending

Requires human confirmation: true

Purpose:

Make Agent execution backend-owned and durable across browser disconnects.

Scope:

- Add create/get/active/cancel/confirm service behavior.
- Add worker dispatch and run execution independent of subscription contexts.
- Append run events and update snapshots during execution.
- Preserve existing chat stream compatibility.

Acceptance:

- `.spec/hr-agent-resumable-stream/acceptance/TASK-HARS-004.md`

## TASK-HARS-005 - Gateway REST And SSE Endpoints

Status: pending

Requires human confirmation: true

Purpose:

Expose the durable run lifecycle to the HR frontend through authenticated gateway endpoints.

Scope:

- Add HR run command/query handlers.
- Add SSE event replay and live subscription endpoint.
- Add routes and handler tests.
- Ensure HTTP disconnect closes only the subscription, not the backend run.

Acceptance:

- `.spec/hr-agent-resumable-stream/acceptance/TASK-HARS-005.md`

## TASK-HARS-006 - Frontend Run API Reducer And Composable

Status: pending

Requires human confirmation: true

Purpose:

Create the unified HR Agent runtime that all HR Agent streaming entry points will use.

Scope:

- Add frontend API helpers and types for durable run operations.
- Add pure TypeScript reducer and tests.
- Add Vue composable for create, subscribe, hydrate, cancel, confirm, and cleanup.

Acceptance:

- `.spec/hr-agent-resumable-stream/acceptance/TASK-HARS-006.md`

## TASK-HARS-007 - Migrate HR Agent View Entry Points

Status: pending

Requires human confirmation: false

Purpose:

Remove duplicated stream orchestration from the HR chat page by routing existing flows through the unified runtime.

Scope:

- Migrate normal submit, retry, route-created candidate analysis, candidate option actions, and skill confirmation in `AIChatView.vue`.
- Keep current UI behavior and presentation stable.
- Add or update focused frontend tests.

Acceptance:

- `.spec/hr-agent-resumable-stream/acceptance/TASK-HARS-007.md`

## TASK-HARS-008 - Refresh Recovery And End-To-End Validation

Status: pending

Requires human confirmation: true

Purpose:

Wire and validate the complete refresh/reconnect behavior across frontend and backend.

Scope:

- Restore active run state on route mount/page refresh.
- Reconnect subscriptions with last sequence.
- Validate cancel, confirmation, completion, and legacy compatibility.
- Produce final TASK report with knowledge impact notes and remaining rollout risks.

Acceptance:

- `.spec/hr-agent-resumable-stream/acceptance/TASK-HARS-008.md`

## Required Knowledge Review

Every non-trivial TASK must review `AGENTS.md`, this feature contract, `.knowledge/README.md`, and the active knowledge docs relevant to its scope. Likely relevant active documents include:

- `.knowledge/architecture/system-overview.md`
- `.knowledge/architecture/service-boundaries.md`
- `.knowledge/architecture/agent-runtime.md`
- `.knowledge/architecture/api-contracts-and-gateway.md`
- `.knowledge/architecture/persistence-and-migrations.md`
- `.knowledge/architecture/frontend-apps.md`
- `.knowledge/domains/notification-outbox.md`
- `.knowledge/runbooks/protobuf-and-migration-change.md`
- `.knowledge/runbooks/frontend-validation.md`
- `.knowledge/pitfalls/protobuf-synchronization.md`
- `.knowledge/pitfalls/migration-model-drift.md`

If a TASK discovers knowledge drift and its scope does not allow editing `.knowledge`, report the drift in the TASK report instead of modifying out-of-scope files.
