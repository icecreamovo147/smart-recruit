# TASK-BDME-035 Report - Identity Auth RBAC API Extraction

## Summary

Extracted the current Identity auth/RBAC runtime surface into `identity-service` without gateway cutover. The new Identity runtime registers the existing AuthService and the Identity-owned AdminService subset only when `cmd/identity-service --serve` is explicitly used; default execution remains fail-closed and unrouted.

## Modified Files

- `logic-grpc-service/internal/identity/interfaces/auth_server.go`: added an Identity-owned generated gRPC adapter for `AuthService`.
- `logic-grpc-service/internal/identity/interfaces/admin_api.go`: defined Identity-owned Admin and audit API subsets for RBAC, scopes, staff identity, and security-audit queries.
- `logic-grpc-service/internal/identity/interfaces/admin_server.go`: added an AdminService adapter that forwards only Identity-owned methods and leaves unrelated AdminService methods unimplemented.
- `logic-grpc-service/internal/identity/interfaces/auth_server_test.go`: verified current Auth/Admin/Analytics services satisfy the Identity interfaces and adapters forward calls.
- `logic-grpc-service/internal/identity/runtime/runtime.go`: added the Identity runtime and `RegisterGRPC` registration for AuthService and AdminService.
- `logic-grpc-service/internal/identity/runtime/runtime_test.go`: verified the runtime registers generated AuthService and AdminService descriptors.
- `logic-grpc-service/internal/identity/runtime/skeleton.go`: updated the descriptor to list extracted APIs and explicit-serve-only startup.
- `logic-grpc-service/internal/identity/runtime/skeleton_test.go`: updated descriptor assertions for extracted APIs and no gateway traffic.
- `logic-grpc-service/cmd/identity-service/main.go`: added explicit `--serve` startup that loads existing config, validates internal gRPC auth, connects MySQL/optional Redis, registers Identity runtime APIs, and exposes health without default traffic.
- `docs/backend-ddd-microservices-evolution-identity-api-extraction.md`: documented extracted APIs, startup, compatibility, and verification.
- `docs/backend-ddd-microservices-evolution-identity-service-skeleton.md`: documented the new explicit serve mode while preserving fail-closed default behavior.
- `.knowledge/architecture/auth-rbac-security.md`: documented extracted Identity runtime surfaces and non-Identity AdminService exclusions.
- `.knowledge/architecture/service-boundaries.md`: documented the unrouted Identity runtime boundary.
- `.knowledge/runbooks/service-binary-convention.md`: documented explicit `--serve` behavior for `identity-service`.
- `.knowledge/manifest.yaml`: routed the Identity API extraction document through service-binary knowledge.
- `.spec/backend-ddd-microservices-evolution/pipeline-state.json`: recorded TASK baseline and completion state.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-035-report.md`: this report.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-035-evidence.json`: machine-readable evidence.

## Scope

Scope check passed. All changed files are allowed by TASK-BDME-035 scope. No frontend, dependency manifest, `go.mod`, `go.sum`, database schema, protobuf source/generated code, gateway route, deployment traffic, SPEC/SDD/TASK, acceptance, prompt, or Harness script files were modified.

Human confirmation was required by the TASK and is recorded as satisfied by the user's global gate override.

## SPEC / SDD / Acceptance

- SPEC comparison: aligned with security and safety requirements by extracting Identity runtime registration while preserving current auth/RBAC behavior and internal token enforcement.
- SDD comparison: aligned with API/interface and compatibility strategy by keeping existing protobuf contracts stable and avoiding gateway cutover.
- Acceptance comparison: the TASK goal is implemented as scoped; current behavior remains compatible; report, evidence, checks, risks, and knowledge impact are recorded.

## Checks

- `git diff --name-only && git ls-files --others --exclude-standard`: passed; listed tracked and untracked TASK files.
- `cd logic-grpc-service && go test ./internal/identity/interfaces ./internal/identity/runtime ./cmd/identity-service`: passed.
- `cd logic-grpc-service && go run ./cmd/identity-service --check && go run ./cmd/identity-service --describe`: passed.
- `TASK_BASE_TREE=29429418d841762a81d9d0002dc661529ea0bd83 bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-035`: passed.
- `bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh`: passed.
- `cd logic-grpc-service && go test ./...`: passed.
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 29429418d841762a81d9d0002dc661529ea0bd83 --json`: passed; update required, no coverage gaps.
- `node .knowledge/scripts/validate-knowledge.mjs --root .`: passed.

## Knowledge Impact

Result: update_required.

Updated auth/RBAC security, service boundary, service-binary, and manifest knowledge. Reviewed auth permission alignment, debug auth permissions, local development, notification outbox, system overview, and knowledge coverage documents; no additional edits were required.

## Self-Review

Verdict: 通过.

Findings: none. The implementation registers existing Identity APIs in an explicit runtime path only, leaves gateway routing unchanged, avoids protobuf/schema changes, preserves token-version and audit behavior, and keeps non-Identity AdminService methods on the monolith target.

## Risks

- `--serve` requires a real config, MySQL, and optional Redis; this TASK verifies compile-time and descriptor/runtime registration but does not run a full integration instance.
- Later TASK-BDME-036 must add gateway routing controls and rollback evidence before any production traffic reaches `identity-service`.

## Next TASK

TASK-BDME-036 can start after this TASK is committed.
