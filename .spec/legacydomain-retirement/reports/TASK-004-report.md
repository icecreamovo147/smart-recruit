# TASK Report - TASK-004

## 1. TASK ID

TASK-004 - Retire Interview legacydomain

## 2. Modified File List

- `smart-recruit-interview-service/cmd/interview-service/main.go`
- `smart-recruit-interview-service/internal/application/port/interview_ports.go`
- `smart-recruit-interview-service/internal/infrastructure/client/application_adapter.go`
- `smart-recruit-interview-service/internal/infrastructure/client/authorizer.go`
- `smart-recruit-interview-service/internal/infrastructure/client/staff_directory.go`
- `smart-recruit-interview-service/internal/infrastructure/mq/outbox_publisher.go`
- `smart-recruit-interview-service/internal/infrastructure/mq/outbox_publisher_test.go`
- `smart-recruit-interview-service/internal/infrastructure/persistence/interview_repository.go`
- `smart-recruit-interview-service/internal/legacydomain/**`
- `.knowledge/architecture/persistence-and-migrations.md`
- `.knowledge/architecture/service-boundaries.md`
- `.knowledge/domains/notification-outbox.md`
- `.knowledge/domains/recruitment-lifecycle.md`
- `.spec/legacydomain-retirement/pipeline-state.json`
- `.spec/legacydomain-retirement/reports/TASK-004-report.md`
- `.spec/legacydomain-retirement/reports/TASK-004-evidence.json`

## 3. Change Summary by File

- `cmd/interview-service/main.go`: wired runtime to local Interview persistence/outbox/staff adapters plus Recruitment `ApplicationOwnerService` and Identity `AuthService` clients.
- `internal/application/port/interview_ports.go`: extended application snapshot metadata for owner-scope authorization.
- `internal/infrastructure/client/application_adapter.go`: replaced legacy application repository usage with Recruitment owner contract calls.
- `internal/infrastructure/client/authorizer.go`: replaced copied authz/job/application repositories with Identity principal/permission checks, Recruitment snapshots, and a local Interview assignment reader.
- `internal/infrastructure/client/staff_directory.go`: replaced legacy user repo with an explicit local read adapter for active interviewer staff.
- `internal/infrastructure/mq/outbox_publisher.go`: replaced copied legacy outbox model with a local `event_outbox` record/store.
- `internal/infrastructure/persistence/interview_repository.go`: replaced the copied legacy Interview repository with local private GORM records and repository adapter methods.
- `internal/legacydomain/**`: deleted the Interview service legacydomain copy.
- `.knowledge/**`: updated active boundary, lifecycle, outbox, and persistence knowledge for Interview retirement.

## 4. Scope Check Result

Passed.

```text
Scope check passed for TASK-004. Changed files: 60
```

## 5. SPEC Comparison Result

Passed. Interview no longer has an `internal/legacydomain` directory or non-test `legacydomain` imports, and active runtime dependencies now use local adapters plus explicit owner contracts.

## 6. SDD Comparison Result

Passed. GORM records remain private to infrastructure adapters, application lifecycle/snapshot calls go through Recruitment owner contracts, and Identity-owned RBAC data is accessed through service contracts for authorization decisions.

## 7. Acceptance Comparison Result

Passed.

- Interview non-test code has no `legacydomain` import.
- `smart-recruit-interview-service/internal/legacydomain` is absent.
- Interview owner persistence uses local infrastructure records/adapters.
- Candidate-facing filtering and feedback tests pass.
- Existing Interview gRPC tests pass.

## 8. Test Commands and Results

| Command | Result |
|---|---|
| `go test ./...` in `smart-recruit-interview-service` | Passed. |
| `node scripts/check-backend-boundaries.mjs` | Passed with expected staging warning for Recruitment and AI Agent remaining legacydomain roots. |
| `node scripts/check-mysql-table-ownership.mjs` | Passed. |
| `node .knowledge/scripts/validate-knowledge.mjs --root .` | Passed. |
| `node .knowledge/scripts/check-references.mjs --root .` | Passed. |
| `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 59853a980cae1269fc2e1522ba302e1de7b65a44` | Passed; `impact_result: update_required`. |
| `bash .spec/legacydomain-retirement/scripts/check-task-scope.sh TASK-004` | Passed; final changed file count 60. |
| `bash .spec/legacydomain-retirement/scripts/agent-check.sh` | Passed. |

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: update_required
  triggered_by:
    - smart-recruit-interview-service/**
    - smart-recruit-interview-service/internal/infrastructure/**
    - .knowledge/**
  reviewed_documents:
    - service-boundaries: UPDATED
    - recruitment-lifecycle: UPDATED
    - notification-outbox: UPDATED
    - persistence-and-migrations: UPDATED
    - debug-recruitment-lifecycle: UNCHANGED
    - knowledge-coverage-audit: UNCHANGED
    - local-development: UNCHANGED
    - migration-model-drift: UNCHANGED
    - protobuf-and-migration-change: UNCHANGED
    - recruitment: UNCHANGED
    - service-binary-convention: UNCHANGED
    - status-notification-drift: UNCHANGED
    - system-overview: UNCHANGED
  update_paths:
    - .knowledge/architecture/service-boundaries.md
    - .knowledge/domains/recruitment-lifecycle.md
    - .knowledge/domains/notification-outbox.md
    - .knowledge/architecture/persistence-and-migrations.md
  coverage_gap: false
  evidence:
    - impact_result: update_required
    - knowledge_result: PASS (37 formal documents)
    - reference_result: PASS
```

## 10. Risks

- `ListInterviewers` remains a local read adapter over Identity-owned tables because the existing Identity protobuf contract does not expose role/keyword-filtered interviewer lookup. This preserves behavior without changing proto in TASK-004, but should be converted to an owner contract in a future internal-contract task.
- Interview list/detail reads still join Recruitment and Identity-owned tables to preserve existing response shape. This removes legacydomain coupling but leaves read-model hardening for later.
- Internal gRPC clients are service-lifetime clients; explicit shutdown ownership can be tightened if the service binary gains cleanup hooks.

## Self-Review

Reviewer type: self-review.

Findings:

- No scope violation found: TASK-004 changed only Interview service files, `.knowledge/**`, and TASK report/evidence/runtime state.
- No direct `legacydomain` import remains in Interview Go files.
- Required checks passed.
- Residual owner-contract opportunities are documented above.

Verdict:

```text
verdict: 通过
```

## 11. Follow-up Items

- TASK-005 can proceed to Recruitment runtime and repository/service graph retirement.
- A later contract-focused task should consider Identity `ListInterviewers` to remove local staff table reads.

## 12. Whether the Next TASK Can Start

Yes. TASK-004 is complete and TASK-005 can start without an additional human gate under the current user instruction unless a blocking issue appears.
