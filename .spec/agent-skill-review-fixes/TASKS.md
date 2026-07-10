# TASKS - agent-skill-review-fixes

## Task Overview

| TASK | Title | Status | Scope | Acceptance |
|------|-------|--------|-------|------------|
| TASK-001 | Fix retry Skill confirmation handling | completed | HR chat frontend | acceptance/TASK-001.md |
| TASK-002 | Make single Skill embedding regeneration deterministic | completed | Embedding backfill + Agent Skill UI | acceptance/TASK-002.md |
| TASK-003 | Make semantic debug metadata request-local | completed | Logic service semantic debug + embedding service | acceptance/TASK-003.md |

## TASK-001 - Fix retry Skill confirmation handling

### Goal

Make `retry()` render and preserve the Agent Skill confirmation card when a retried stream emits `agent_skill_selection_required`.

### Scope

Frontend HR chat stream event handling.

### Allowed Files

- `hr-frontend/src/views/hr/AIChatView.vue`
- `hr-frontend/src/views/hr/AIChatView.test.ts`
- `hr-frontend/src/views/hr/__tests__/AIChatView.test.ts`

### Forbidden Files

- Backend service files
- Protobuf files
- Package manifests and lockfiles
- Router/auth/permission files

### Dependencies

None.

### Acceptance Criteria

- Retry handles `agent_skill_selection_required`.
- Retry calls `setSkillSelectionMessage()` or equivalent shared handler.
- Retry skips post-stream message refresh while Skill confirmation is pending.
- Existing submit and confirmed submit flows keep current behavior.

### Required Tests

- `pnpm --filter hr-frontend typecheck`
- Add or update focused frontend test if existing test setup supports it. If not, document manual verification in the report.

### Risks

- Over-refactoring stream handling could destabilize normal submit behavior.

### Notes

Keep helper extraction local to `AIChatView.vue` unless a broader shared abstraction is explicitly approved.

## TASK-002 - Make single Skill embedding regeneration deterministic

### Goal

Prevent all-zero embedding regeneration results from being treated as success.

### Scope

Embedding backfill result semantics and Agent Skill management UI response handling.

### Allowed Files

- `logic-grpc-service/service/embedding_backfill_service.go`
- `logic-grpc-service/service/embedding_service_test.go`
- `logic-grpc-service/service/embedding_backfill_service_test.go`
- `hr-frontend/src/views/hr/admin/AgentSkillManageView.vue`
- `hr-frontend/src/api/agentSkill.ts`

### Forbidden Files

- Protobuf files unless explicit confirmation is given
- Database migrations
- Auth/permission files
- Package manifests and lockfiles

### Dependencies

TASK-001 may be completed independently.

### Acceptance Criteria

- `object_id > 0` with no eligible Agent Skill returns a non-success result.
- Frontend success toast requires `success_count > 0`.
- Frontend skipped/all-zero result displays warning or error.
- Batch backfill behavior remains compatible.

### Required Tests

- `go test ./...` from `logic-grpc-service/` or a targeted package test if full suite is too slow.
- `pnpm --filter hr-frontend typecheck`.

### Risks

- Changing result semantics too broadly could affect batch backfill UX.

### Notes

Prefer using existing response fields over protobuf changes.

## TASK-003 - Make semantic debug metadata request-local

### Goal

Ensure semantic debug embedding metadata describes Skill retrieval and cannot be overwritten by Memory retrieval.

### Scope

Embedding search metadata flow inside logic service.

### Allowed Files

- `logic-grpc-service/service/embedding_service.go`
- `logic-grpc-service/service/embedding_service_test.go`
- `logic-grpc-service/service/agent_skill_service.go`
- `logic-grpc-service/service/agent_skill_service_test.go`

### Forbidden Files

- Protobuf files unless explicit confirmation is given
- Web gateway files
- Frontend files
- Database migrations
- Package manifests and lockfiles

### Dependencies

TASK-002 may be completed independently. TASK-003 should not depend on TASK-001.

### Acceptance Criteria

- Skill search metadata is captured before memory search.
- `DebugSemanticRetrievalResponse` uses Skill search metadata.
- Memory debug search cannot overwrite the metadata returned for Skill debug fields.
- Existing semantic debug fallback behavior remains intact.

### Required Tests

- `go test ./...` from `logic-grpc-service/` or targeted service tests if full suite is too slow.

### Risks

- Changing `Search()` signatures directly may create broad churn. Prefer additive helpers or local metadata capture.

### Notes

Stop and request confirmation if protobuf schema changes appear necessary.
