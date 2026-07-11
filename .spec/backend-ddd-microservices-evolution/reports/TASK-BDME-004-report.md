# TASK-BDME-004 Report

## TASK

- TASK ID: TASK-BDME-004
- Title: Core Workflow Regression Baseline
- Status: completed
- Self-review verdict: 通过

## Modified Files

- `.spec/backend-ddd-microservices-evolution/pipeline-state.json`: recorded TASK-BDME-004 baseline, evidence, checks, and completion state.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-004-report.md`: this TASK report.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-004-evidence.json`: machine-readable TASK evidence.
- `docs/backend-ddd-microservices-evolution-core-workflow-regression-baseline.md`: added workflow-to-test coverage matrix and documented gaps.
- `logic-grpc-service/model/core_workflow_status_test.go`: added current-behavior regression tests for default application status, reapplication terminal states, and audience-specific status labels.

## Scope Result

- Scope check: passed.
- Out-of-scope changes: none.
- Forbidden files modified: none.
- Frontend, schema, proto, package, dependency, auth policy, and deployment traffic changes: none.

## SPEC / SDD / Acceptance Comparison

- SPEC §5 Functional Requirements: satisfied by mapping auth, recruitment, interview, offer, notification, and AI workflows to existing focused tests or documented gaps.
- SPEC §11 Acceptance Criteria: satisfied by passing backend test suites and recording TASK report/evidence.
- SDD §1 Existing Architecture Summary: satisfied by using the current gateway/logic test layout as the baseline.
- SDD §3.3 Migration Phases: satisfied by strengthening safety-net regression coverage before later extraction work.
- Acceptance:
  - Core workflows are mapped to tests or explicit gaps: passed.
  - New tests assert current behavior, not future behavior: passed.
  - Backend test suites pass or skipped checks have approved reasons: passed; no required backend checks were skipped.

## Checks

- `git diff --name-only && git ls-files --others --exclude-standard`: passed.
- `TASK_BASE_TREE=5b76f65926adf539d7d317e95fcef2ba2327d651 bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-004`: passed.
- `bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh`: passed.
- `cd logic-grpc-service && go test ./...`: passed.
- `cd web-gin-service && go test ./...`: passed.
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 5b76f65926adf539d7d317e95fcef2ba2327d651`: passed.

## Coverage And Gaps

- Covered: auth/RBAC/token invalidation via existing middleware, RBAC, refresh-token, and interceptor tests.
- Covered: recruitment status lifecycle via existing transition tests plus new status model baseline tests.
- Covered: interview, offer, notification, and AI workflows through existing service/repository/handler tests listed in the baseline document.
- Documented gaps: exhaustive browser cookie flows, route-by-route JSON snapshots, SSE end-to-end streaming, live provider calls, and focused offer HTTP handler behavior.

## Knowledge Impact

- Impact result: `update_required`.
- Reviewed active knowledge documents for system overview, service boundaries, persistence/migration, local development, proto/migration runbook, and migration-model drift.
- Knowledge document changes: none.
- Coverage gap: false.

## Risks

- The baseline intentionally avoids live external provider calls; future provider behavior must be covered by targeted integration tests or documented skips.
- Some workflows still rely on compile/full-suite coverage rather than focused handler tests; later extraction tasks should add parity tests where they touch behavior.

## Next TASK

Next TASK can start: yes.
