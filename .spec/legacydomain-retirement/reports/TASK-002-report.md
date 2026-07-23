# TASK Report - TASK-002

## 1. TASK ID

TASK-002 - Owner contracts for lifecycle and authorization

## 2. Modified File List

- `smart-recruit-proto/proto/recruitment.proto`
- `smart-recruit-proto/recruitment/pb/recruitment.pb.go`
- `smart-recruit-proto/recruitment/pb/recruitment_grpc.pb.go`
- `smart-recruit-identity-service/internal/interfaces/grpc/identity_server.go`
- `smart-recruit-identity-service/internal/runtime/runtime.go`
- `smart-recruit-identity-service/internal/runtime/runtime_test.go`
- `smart-recruit-identity-service/cmd/identity-service/main.go`
- `smart-recruit-recruitment-service/internal/application/command/recruitment.go`
- `smart-recruit-recruitment-service/internal/application/dto/recruitment.go`
- `smart-recruit-recruitment-service/internal/application/service/application_collaboration_taxonomy_service.go`
- `smart-recruit-recruitment-service/internal/application/service/application_collaboration_taxonomy_service_test.go`
- `smart-recruit-recruitment-service/internal/domain/repository/recruitment.go`
- `smart-recruit-recruitment-service/internal/interfaces/grpc/application_owner_contract.go`
- `smart-recruit-recruitment-service/internal/interfaces/grpc/adapters_test.go`
- `smart-recruit-recruitment-service/internal/runtime/runtime.go`
- `smart-recruit-recruitment-service/internal/runtime/runtime_test.go`
- `smart-recruit-gateway/rpc/client.go`
- `.knowledge/architecture/api-contracts-and-gateway.md`
- `.knowledge/architecture/auth-rbac-security.md`
- `.knowledge/architecture/service-boundaries.md`
- `.knowledge/domains/recruitment.md`
- `.knowledge/domains/recruitment-lifecycle.md`
- `.knowledge/pitfalls/protobuf-synchronization.md`
- `.knowledge/runbooks/protobuf-and-migration-change.md`
- `.spec/legacydomain-retirement/pipeline-state.json`
- `.spec/legacydomain-retirement/scripts/check-task-scope.sh`
- `.spec/legacydomain-retirement/reports/TASK-002-report.md`
- `.spec/legacydomain-retirement/reports/TASK-002-evidence.json`

## 3. Change Summary by File

- `smart-recruit-proto/**`: added `AuthService.AuthorizeInternal` and a separate internal `ApplicationOwnerService` for application snapshots and conditional lifecycle transitions; regenerated Go contracts.
- `smart-recruit-identity-service/**`: implemented `AuthorizeInternal` from existing principal/RBAC data, including permission/scope evaluation and allow/deny audit recording; updated runtime/noop/test fakes.
- `smart-recruit-recruitment-service/**`: added native application snapshot and lifecycle transition methods, gRPC adapter, runtime registration, and focused tests. The existing public `ApplicationService` interface remains unchanged.
- `smart-recruit-gateway/rpc/client.go`: exposed a generated `ApplicationOwnerServiceClient` without changing HTTP routes or handlers.
- `.knowledge/**`: updated active architecture/domain/runbook/pitfall docs for internal owner contracts and protobuf sync guidance.
- `.spec/legacydomain-retirement/scripts/check-task-scope.sh`: repaired harness scope checking to use per-TASK `pipeline-state.task_runs[*].base_tree` and to allow harness runtime state; this prevents completed TASK-001 changes from polluting later scope checks.

## 4. Scope Check Result

Passed.

```text
Scope check passed for TASK-002. Changed files: 28
```

The scope script repair is harness-control-plane support for the required TASK base-tree invariant. It does not alter business runtime behavior.

## 5. SPEC Comparison Result

Passed. TASK-002 adds explicit owner contracts for Identity authorization/principal checks and Recruitment application snapshot/lifecycle behavior without changing frontend routes, gateway HTTP behavior, database schema, or legacydomain files.

## 6. SDD Comparison Result

Passed. The implementation follows the staged debt-retirement design: contracts are additive, GORM records stay out of domain, and the Recruitment owner contract is separated from the public-facing `ApplicationService` to avoid unnecessary client blast radius.

## 7. Acceptance Comparison Result

Passed.

- Recruitment contract supports Offer/Interview snapshot and lifecycle needs through `ApplicationOwnerService`, including job owner/scope metadata, actor account type, and close-current-round semantics.
- Identity contract supports principal, permission, scope, and audit needs through `AuthorizeInternal`.
- Public gateway/frontend behavior is unchanged.
- Protobuf changes are additive and generated code is synchronized.
- Human confirmation was recorded before implementation.

## 8. Test Commands and Results

| Command | Result |
|---|---|
| `go test ./...` in `smart-recruit-proto` | Passed. |
| `go test ./...` in `smart-recruit-recruitment-service` | Passed. |
| `go test ./...` in `smart-recruit-identity-service` | Passed. |
| `go test ./...` in `smart-recruit-gateway` | Passed. |
| `node scripts/check-backend-boundaries.mjs` | Passed with expected staging warning for remaining legacydomain roots/imports. |
| `node scripts/check-mysql-table-ownership.mjs` | Passed. |
| `node .knowledge/scripts/validate-knowledge.mjs --root .` | Passed. |
| `node .knowledge/scripts/check-references.mjs --root .` | Passed. |
| `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 3d5eeeab29b0502ae468f41c2fb230d3c1c21b2e` | Passed; `impact_result: update_required`. |
| `bash .spec/legacydomain-retirement/scripts/check-task-scope.sh TASK-002` | Passed; final changed file count 28 including report/evidence. |
| `bash .spec/legacydomain-retirement/scripts/agent-check.sh` | Passed. |

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: update_required
  triggered_by:
    - smart-recruit-proto/**
    - smart-recruit-identity-service/**
    - smart-recruit-recruitment-service/**
    - smart-recruit-gateway/rpc/**
    - .knowledge/**
  reviewed_documents:
    - api-contracts-and-gateway: UPDATED
    - auth-rbac-security: UPDATED
    - service-boundaries: UPDATED
    - recruitment: UPDATED
    - recruitment-lifecycle: UPDATED
    - protobuf-synchronization: UPDATED
    - protobuf-and-migration-change: UPDATED
    - auth-permission-alignment: UNCHANGED
    - debug-auth-permissions: UNCHANGED
    - debug-recruitment-lifecycle: UNCHANGED
    - knowledge-coverage-audit: UNCHANGED
    - local-development: UNCHANGED
    - notification-outbox: UNCHANGED
    - service-binary-convention: UNCHANGED
    - status-notification-drift: UNCHANGED
    - system-overview: UNCHANGED
  update_paths:
    - .knowledge/architecture/api-contracts-and-gateway.md
    - .knowledge/architecture/auth-rbac-security.md
    - .knowledge/architecture/service-boundaries.md
    - .knowledge/domains/recruitment.md
    - .knowledge/domains/recruitment-lifecycle.md
    - .knowledge/pitfalls/protobuf-synchronization.md
    - .knowledge/runbooks/protobuf-and-migration-change.md
  coverage_gap: false
  evidence:
    - impact_result: update_required
    - knowledge_result: PASS (37 formal documents)
    - reference_result: PASS
```

## 10. Risks

- `ApplicationOwnerService` is registered by Recruitment runtime, but the active `cmd/recruitment-service` graph still uses legacy services until TASK-005. The owner contract adapter is ready for native runtime wiring.
- `AuthorizeInternal` performs permission and broad/resource scope evaluation from Identity-owned principal data; job/application-specific ownership decisions still require owner service data or downstream scope-aware ports.
- Remaining legacydomain roots/imports are expected until TASK-003 through TASK-009.

## Self-Review

Reviewer type: self-review.

Findings repaired:

- Medium: adding owner lifecycle RPCs to public `ApplicationService` widened gateway test fake compile scope. Repaired by moving lifecycle/snapshot methods to separate internal `ApplicationOwnerService`.
- Medium: TASK scope checker did not honor per-TASK `base_tree`, which made completed TASK-001 files appear out-of-scope for TASK-002. Repaired in the feature harness script.
- Medium: first owner lifecycle contract did not expose close-current-round and actor account type semantics needed by Offer. Repaired by appending fields and routing them through Recruitment application service.
- Medium: first application snapshot contract did not expose job owner/department/location metadata needed to replace Offer application-access authorization. Repaired by appending snapshot fields and mapping them from Recruitment job data.

Verdict:

```text
verdict: 通过
```

## 11. Follow-up Items

- TASK-003 can use `ApplicationOwnerService` and `AuthorizeInternal` clients while retiring Offer legacydomain.
- TASK-005 must wire Recruitment native persistence/application services into active runtime so `ApplicationOwnerService` is backed by non-legacy adapters.

## 12. Whether the Next TASK Can Start

Yes. TASK-002 is complete and TASK-003 does not require human confirmation under the current user instruction unless a blocking issue appears.
