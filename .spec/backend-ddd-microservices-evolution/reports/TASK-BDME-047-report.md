# TASK-BDME-047 Report - Schema Separation Plan

## Summary

Created the schema separation plan for the backend DDD/microservices evolution. The plan keeps this TASK non-executing: no migrations, models, repositories, `db.sql`, deployment traffic, or runtime configuration were changed. It documents schema-per-context separation, physical database prerequisites, expand-contract flow, rollback, reconciliation, RTO/RPO targets, and the required alignment with the table ownership manifest.

## Modified Files

- `docs/backend-ddd-microservices-evolution-schema-separation-plan.md`: added the schema/database separation plan with ownership mapping, preconditions, expand-contract flow, rollback, reconciliation, and RTO/RPO requirements.
- `docs/backend-ddd-microservices-evolution-table-ownership.md`: linked future schema or physical database separation to the separation plan.
- `.knowledge/architecture/persistence-and-migrations.md`: documented the separation plan as the required guardrail before schema/model/`db.sql` changes.
- `.knowledge/architecture/service-boundaries.md`: documented schema separation as a service-boundary concern requiring evidence before runtime storage changes.
- `.knowledge/pitfalls/migration-model-drift.md`: added ungoverned schema separation as a migration/model drift pitfall.
- `.knowledge/runbooks/protobuf-and-migration-change.md`: added separation plan checks to migration work.
- `.knowledge/manifest.yaml`: routed the separation plan to persistence, migration drift, migration runbook, and service-boundary knowledge.
- `.spec/backend-ddd-microservices-evolution/pipeline-state.json`: recorded TASK completion and next TASK state.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-047-report.md`: this report.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-047-evidence.json`: machine-readable evidence.

## Scope

Scope check passed. All changed files are allowed by TASK-BDME-047 scope. No frontend, dependency manifest, `go.mod`, `go.sum`, database schema, migration, model, repository, protobuf, deployment traffic, SPEC/SDD/TASK, acceptance, prompt, or Harness script files were modified.

Human confirmation was required by the TASK and is recorded as satisfied by the user's global gate override.

## SPEC / SDD / Acceptance

- SPEC comparison: aligned with FR-001..FR-008 and AC-004..AC-008 by grounding schema/database separation in explicit table ownership and requiring removal or approval of shared database access.
- SDD comparison: aligned with Phase 4 data ownership/HA and the target ownership matrix by defining schema-per-context units, physical database prerequisites, RTO 30 minutes, and RPO 5 minutes.
- Acceptance comparison: passed. The plan follows the ownership manifest, documents expand-contract, rollback, reconciliation, RTO, and RPO, and records that no migration/model/`db.sql` change occurred in this TASK.

## Checks

- `git diff --name-only && git ls-files --others --exclude-standard`: passed.
- `node scripts/check-table-ownership.mjs`: passed; 67 tables and 6 transitional shared access entries validated.
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree ca89fa42179f8b63fbe7c843f0c2e6540b57592a --json`: passed; update required, no coverage gaps.
- `node .knowledge/scripts/validate-knowledge.mjs --root .`: passed.
- `cd logic-grpc-service && go test ./...`: passed.
- `TASK_BASE_TREE=ca89fa42179f8b63fbe7c843f0c2e6540b57592a bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-047`: passed.
- `bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh`: passed.
- `git diff --check`: passed.

## Knowledge Impact

Result: update_required.

Updated persistence/migration, service-boundary, migration drift, migration runbook, and manifest routing knowledge. Reviewed knowledge coverage audit, system overview, and local development due mechanical routes; no changes were required there.

## Self-Review

Verdict: 通过.

Findings: none. The plan is aligned to the ownership manifest, includes expand-contract/rollback/reconciliation/RTO/RPO, avoids unscoped schema changes, and leaves runtime behavior unchanged.

## Risks

- This TASK creates governance and readiness criteria only; actual schema or physical database separation still requires scoped implementation TASKs and evidence.
- Later separation work must resolve the documented transitional shared access before physical database isolation.
- RTO/RPO targets depend on future backup, restore, replay, and reconciliation drills.

## Next TASK

TASK-BDME-048 can start after this TASK is committed.
