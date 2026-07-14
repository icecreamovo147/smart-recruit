# TASK Report - TASK-003

## 1. TASK ID

TASK-003 - Retire Offer legacydomain

## 2. Modified File List

- `smart-recruit-offer-service/cmd/offer-service/main.go`
- `smart-recruit-offer-service/internal/application/port/offer_ports.go`
- `smart-recruit-offer-service/internal/infrastructure/client/application_adapter.go`
- `smart-recruit-offer-service/internal/infrastructure/client/authorizer.go`
- `smart-recruit-offer-service/internal/infrastructure/mq/outbox_publisher.go`
- `smart-recruit-offer-service/internal/infrastructure/mq/outbox_publisher_test.go`
- `smart-recruit-offer-service/internal/infrastructure/persistence/offer_repository.go`
- `smart-recruit-offer-service/internal/legacydomain/**`
- `.knowledge/architecture/persistence-and-migrations.md`
- `.knowledge/architecture/service-boundaries.md`
- `.knowledge/domains/notification-outbox.md`
- `.knowledge/domains/recruitment-lifecycle.md`
- `.spec/legacydomain-retirement/pipeline-state.json`
- `.spec/legacydomain-retirement/reports/TASK-003-report.md`
- `.spec/legacydomain-retirement/reports/TASK-003-evidence.json`

## 3. Change Summary by File

- `cmd/offer-service/main.go`: wired Offer runtime to local Offer persistence/outbox adapters plus Recruitment `ApplicationOwnerService` and Identity `AuthService` owner clients.
- `internal/application/port/offer_ports.go`: extended the application snapshot port with job owner, department, and location metadata required for owner-scope authorization.
- `internal/infrastructure/client/application_adapter.go`: replaced legacy application repository usage with `ApplicationOwnerService` snapshot and lifecycle adapters.
- `internal/infrastructure/client/authorizer.go`: replaced copied authz repository usage with Identity owner contract calls and local scope checks against Recruitment-owned application snapshots.
- `internal/infrastructure/mq/outbox_publisher.go`: replaced copied legacy outbox model/repository usage with a local GORM outbox store.
- `internal/infrastructure/mq/outbox_publisher_test.go`: updated the outbox test to assert against the local outbox record.
- `internal/infrastructure/persistence/offer_repository.go`: replaced the copied legacy Offer repository with local private GORM records and an Offer-domain repository adapter.
- `internal/legacydomain/**`: deleted the Offer service legacydomain copy.
- `.knowledge/**`: updated active boundary, lifecycle, outbox, and persistence knowledge for the Offer retirement.

## 4. Scope Check Result

Passed.

```text
Scope check passed for TASK-003. Changed files: 59
```

## 5. SPEC Comparison Result

Passed. Offer no longer has an `internal/legacydomain` directory or non-test `legacydomain` imports, and active runtime dependencies now use explicit local infrastructure adapters and owner contracts.

## 6. SDD Comparison Result

Passed. The implementation keeps GORM records out of domain packages, preserves the public gRPC surface, and uses TASK-002 owner contracts for application lifecycle, snapshot, and authorization seams.

## 7. Acceptance Comparison Result

Passed.

- Offer non-test code has no `legacydomain` import.
- `smart-recruit-offer-service/internal/legacydomain` is absent.
- Offer owner persistence is implemented as local private records/adapters under `internal/infrastructure/persistence`.
- Outbox, lifecycle, snapshot, and authz dependencies use explicit adapters.
- Existing Offer gRPC tests pass.

## 8. Test Commands and Results

| Command | Result |
|---|---|
| `go test ./...` in `smart-recruit-offer-service` | Passed. |
| `node scripts/check-backend-boundaries.mjs` | Passed with expected staging warning for Interview, Recruitment, and AI Agent remaining legacydomain roots. |
| `node scripts/check-mysql-table-ownership.mjs` | Passed. |
| `node .knowledge/scripts/validate-knowledge.mjs --root .` | Passed. |
| `node .knowledge/scripts/check-references.mjs --root .` | Passed. |
| `node .knowledge/scripts/detect-impact.mjs --root . --base-tree f24aa86a716b6df8cf5ad880e6bd4cde582507ce` | Passed; `impact_result: update_required`. |
| `bash .spec/legacydomain-retirement/scripts/check-task-scope.sh TASK-003` | Passed; final changed file count 59. |
| `bash .spec/legacydomain-retirement/scripts/agent-check.sh` | Passed. |

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: update_required
  triggered_by:
    - smart-recruit-offer-service/**
    - smart-recruit-offer-service/internal/infrastructure/**
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

- Offer authorization now evaluates application access from Recruitment snapshot metadata plus Identity data scopes. The old copied authz repository could also resolve `assigned_interviews` through `interview_schedules`; preserving that exact interviewer-scope behavior requires a future Interview owner contract or read-model adapter and should be handled with TASK-004 or later owner-boundary work.
- Offer list/detail queries still join read-only candidate, user, job, and application tables to preserve response shape. This removes legacydomain coupling but leaves cross-context display reads as a later read-model hardening opportunity.
- The runtime opens long-lived internal gRPC clients for Identity and Recruitment. They are service-lifetime clients and not per-request leaks, but shutdown ownership can be tightened if the service binary lifecycle grows explicit cleanup hooks.

## Self-Review

Reviewer type: self-review.

Findings:

- No scope violation found: TASK-003 changed only Offer service files, `.knowledge/**`, and TASK report/evidence/runtime state.
- No direct `legacydomain` import remains in Offer Go files.
- Required checks passed.
- Residual compatibility risk is documented above for `assigned_interviews` data scope parity.

Verdict:

```text
verdict: 通过
```

## 11. Follow-up Items

- TASK-004 should retire Interview legacydomain and decide whether Interview-owned assigned-interviewer access becomes an owner contract consumed by Offer/Recruitment.
- TASK-005 should continue removing Recruitment active runtime dependencies on legacy services and repositories.

## 12. Whether the Next TASK Can Start

Yes. TASK-003 is complete and TASK-004 can start without an additional human gate under the current user instruction unless a blocking issue appears.
