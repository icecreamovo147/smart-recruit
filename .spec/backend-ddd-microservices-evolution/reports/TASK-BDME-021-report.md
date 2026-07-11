# TASK-BDME-021 Report

## TASK

- TASK ID: TASK-BDME-021
- Title: Inbox Idempotency Foundation
- Status: completed
- Self-review verdict: 通过

## Modified Files

- `.spec/backend-ddd-microservices-evolution/pipeline-state.json`: recorded TASK-BDME-021 checks, evidence, human-gate override, completion state, and pre-seeded TASK-BDME-022.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-021-report.md`: this TASK report.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-021-evidence.json`: machine-readable TASK evidence.
- `logic-grpc-service/migrations/000052_add_event_inbox.sql`: added `event_inbox` for consumer idempotency, duplicate diagnostics, status, attempt count, and retention timestamps.
- `logic-grpc-service/migrations/000052_add_event_inbox.down.sql`: added rollback for `event_inbox`.
- `db.sql`: aligned the current schema reference with `event_inbox`.
- `logic-grpc-service/model/model.go`: added `EventInbox` model and Inbox status constants.
- `logic-grpc-service/repository/inbox_repo.go`: added claim, duplicate skip, failed reclaim, dead-letter skip, status updates, stats, and retention helpers.
- `logic-grpc-service/repository/inbox_repo_test.go`: covered duplicate skip, failed reclaim, dead-letter skip, stats, and retention cleanup.
- `logic-grpc-service/repository/repo_test_helper.go`: included `EventInbox` in repository test migrations.
- `logic-grpc-service/service/inbox_consumer.go`: added shared MQ consumer idempotency wrapper and payload identity extraction.
- `logic-grpc-service/service/inbox_consumer_test.go`: covered duplicate skip through the wrapper and body-hash fallback identity.
- `logic-grpc-service/service/{notification,email,resume_parse,embedding_event,agent_run}_consumer.go`: wired `Start` handlers through the optional shared Inbox helper.
- `logic-grpc-service/service/services.go`: injects a shared `InboxRepo` into MQ consumers.
- Knowledge files: updated outbox/inbox domain, persistence, migration drift, migration runbook, and recruitment lifecycle debug guidance.

## Scope Result

- Scope check: passed.
- Out-of-scope changes: none.
- Forbidden files modified: none.
- Human confirmation: required by task-scope and satisfied by the user's global gate override in this thread.
- Public API, frontend, package, dependency, auth, and deployment traffic changes: none.
- Database schema changes: yes; explicitly allowed by TASK-BDME-021 scope and covered by human-gate override.

## SPEC / SDD / Acceptance Comparison

- SPEC §5 FR-006: satisfied by adding Inbox controls for cross-domain event consumers.
- SPEC §5 FR-011: satisfied by recording per-consumer event claims, attempts, duplicate skips, failures, dead-letter state, and retention timestamps.
- SPEC §5 FR-012: satisfied by pairing standardized Outbox events with a consumer-side Inbox foundation.
- SPEC §9 Error Handling and Fallback Requirements: satisfied by failed/dead statuses, duplicate detection, retry-safe failed reclaim, and diagnostics.
- SDD §6.2 Domain Event Flow: satisfied by adding the `Inbox/idempotent consumer` step for MQ consumers.
- SDD §9 Error Handling and Fallback Design: satisfied by explicit idempotency keys or body-hash fallback, failed processing records, dead-letter skip semantics, and retention helpers.
- Acceptance:
  - TASK goal implemented as scoped: passed.
  - Existing behavior remains compatible: passed; existing `handle` methods keep their business behavior and `Start` adds idempotency around delivery.
  - Report/evidence include scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact: passed.

## Checks

- `go test ./repository -run 'TestInbox'`: passed.
- `go test ./service -run 'TestConsumeWithInbox|TestInboxIdentity'`: passed.
- `cd logic-grpc-service && go test ./...`: passed.
- `git diff --name-only && git ls-files --others --exclude-standard`: passed.
- `TASK_BASE_TREE=1d2cb5c094b2d83b905fb1f6ff11a635bdcd616f bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-021`: passed.
- `bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh`: passed.
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 1d2cb5c094b2d83b905fb1f6ff11a635bdcd616f`: passed.
- `node .knowledge/scripts/validate-knowledge.mjs --root .`: passed.

## Knowledge Impact

- Impact result: `update_required`.
- Updated active knowledge documents:
  - `.knowledge/domains/notification-outbox.md`: UPDATED.
  - `.knowledge/architecture/persistence-and-migrations.md`: UPDATED.
  - `.knowledge/pitfalls/migration-model-drift.md`: UPDATED.
  - `.knowledge/runbooks/debug-recruitment-lifecycle.md`: UPDATED.
  - `.knowledge/runbooks/protobuf-and-migration-change.md`: UPDATED.
- Reviewed but unchanged due broad routes: `agent-runtime`, `agent-skill`, `ai-configuration-governance`, `auth-rbac-security`, `debug-ai-configuration`, `debug-resume-intelligence`, `embedding-fallback`, `knowledge-coverage-audit`, `local-development`, `mcp-policy-audit`, `mcp-tool-governance`, `recruitment`, `recruitment-lifecycle`, `resume-intelligence`, `resume-sensitive-data`, `semantic-retrieval`, `service-boundaries`, `status-notification-drift`, `system-overview`.
- Coverage gap: false.

## Risks

- Inbox wrapping occurs at MQ `Start` entrypoints; direct unit tests that call `handle` still bypass Inbox by design.
- Failed events are reclaimable by duplicate redelivery, while dead-letter records are skipped until a future explicit repair/replay workflow.
- Messages without `event_id` use a body hash as fallback identity; producers should prefer standard envelope event IDs.

## Next TASK

Next TASK can start: yes.
