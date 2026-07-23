# TASK-BDME-020 Report

## TASK

- TASK ID: TASK-BDME-020
- Title: Outbox Standardization
- Status: completed
- Self-review verdict: 通过

## Modified Files

- `.spec/backend-ddd-microservices-evolution/pipeline-state.json`: recorded TASK-BDME-020 checks, evidence, human-gate override, completion state, and pre-seeded TASK-BDME-021.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-020-report.md`: this TASK report.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-020-evidence.json`: machine-readable TASK evidence.
- `logic-grpc-service/migrations/000051_standardize_event_outbox.sql`: added outbox envelope metadata, idempotency key, correlation/causation/trace fields, metadata JSON, publish/dead-letter timestamps, and observability indexes.
- `logic-grpc-service/migrations/000051_standardize_event_outbox.down.sql`: added rollback for the new outbox columns and indexes.
- `db.sql`: aligned the current schema reference for `event_outbox`.
- `logic-grpc-service/model/model.go`: aligned `model.EventOutbox` with the standardized outbox schema.
- `logic-grpc-service/repository/outbox_repo.go`: records publish/dead-letter timestamps, exposes backlog stats, defines 30/90-day retention cutoffs, and adds terminal-event cleanup helpers.
- `logic-grpc-service/repository/outbox_repo_test.go`: added coverage for publish timestamps, dead-letter timestamps, stats, and retention cleanup.
- `logic-grpc-service/service/outbox_publisher.go`: writes the standard envelope metadata, preserves legacy top-level payload fields, stores nested envelope payload, and dead-letters events after retry budget exhaustion.
- `logic-grpc-service/service/outbox_publisher_test.go`: added payload compatibility/envelope assertions and retry-budget dead-letter coverage.
- `.knowledge/architecture/persistence-and-migrations.md`: documented outbox schema alignment requirements.
- `.knowledge/domains/notification-outbox.md`: documented standardized outbox metadata, compatibility payload shape, retry/dead-letter behavior, observability, and retention.
- `.knowledge/pitfalls/migration-model-drift.md`: added outbox-specific drift prevention.
- `.knowledge/runbooks/debug-recruitment-lifecycle.md`: added retry/dead-letter status and outbox stats to notification debugging.
- `.knowledge/runbooks/protobuf-and-migration-change.md`: added transactional outbox migration review checks.

## Scope Result

- Scope check: passed.
- Out-of-scope changes: none.
- Forbidden files modified: none.
- Human confirmation: required by task-scope and satisfied by the user's global gate override in this thread.
- Public API, frontend, package, dependency, auth, and deployment traffic changes: none.
- Database schema changes: yes; explicitly allowed by TASK-BDME-020 scope and covered by human-gate override.

## SPEC / SDD / Acceptance Comparison

- SPEC §5 FR-006: satisfied by standardizing transactional event records for cross-domain event collaboration.
- SPEC §5 FR-011: satisfied by idempotency keys, retry counts, terminal dead-letter timestamps, last error tracking, and backlog stats.
- SPEC §5 FR-012: satisfied by standardizing existing Outbox writes and persistence before service extraction depends on them.
- SPEC §9 Error Handling and Fallback Requirements: satisfied by bounded retries, dead-letter state, retained diagnostic metadata, and retention-ready terminal timestamps.
- SDD §6.2 Domain Event Flow: satisfied by aligning Outbox records and publisher payloads with the standard domain-event envelope.
- SDD §9 Error Handling and Fallback Design: satisfied by preserving committed events, retrying transient failures, dead-lettering poison messages, and exposing repair/observability hooks.
- Acceptance:
  - TASK goal implemented as scoped: passed.
  - Existing behavior remains compatible: passed; existing consumers still receive legacy top-level business payload fields while envelope metadata is added.
  - Report/evidence include scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact: passed.

## Checks

- `go test ./repository -run 'TestOutbox'`: passed.
- `go test ./service -run 'TestBuildOutboxEvent|TestOutbox'`: passed.
- `cd logic-grpc-service && go test ./...`: passed.
- `git diff --name-only && git ls-files --others --exclude-standard`: passed.
- `TASK_BASE_TREE=6a7041dfd81973388c27b20e51803df490741e98 bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-020`: passed.
- `bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh`: passed.
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 6a7041dfd81973388c27b20e51803df490741e98`: passed.
- `node .knowledge/scripts/validate-knowledge.mjs --root .`: passed.

## Knowledge Impact

- Impact result: `update_required`.
- Updated active knowledge documents:
  - `.knowledge/domains/notification-outbox.md`: UPDATED.
  - `.knowledge/architecture/persistence-and-migrations.md`: UPDATED.
  - `.knowledge/pitfalls/migration-model-drift.md`: UPDATED.
  - `.knowledge/runbooks/debug-recruitment-lifecycle.md`: UPDATED.
  - `.knowledge/runbooks/protobuf-and-migration-change.md`: UPDATED.
- Reviewed but unchanged due broad `model.go` and service routes: `agent-runtime`, `agent-skill`, `ai-configuration-governance`, `debug-ai-configuration`, `embedding-fallback`, `knowledge-coverage-audit`, `local-development`, `mcp-policy-audit`, `mcp-tool-governance`, `recruitment`, `recruitment-lifecycle`, `semantic-retrieval`, `service-boundaries`, `status-notification-drift`, `system-overview`.
- Coverage gap: false.

## Risks

- The new migration uses MySQL JSON/index features consistent with the existing migration style; production rollout should still verify migration execution on a MySQL-compatible database before deployment.
- Existing consumers remain compatible because top-level business fields are preserved, but future consumers should read the nested standard `payload` field.
- Retention helpers are implemented but not scheduled in this TASK; later operational tasks should decide execution cadence and safety controls.

## Next TASK

Next TASK can start: yes. TASK-BDME-021 requires human confirmation in `task-scope.json`, but the user explicitly cancelled later human gates after implementation and self-review pass; pipeline state records that override.
