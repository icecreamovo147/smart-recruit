# TASK-BDME-044 Report - Analytics Service Extraction

## Summary

Created a compile-safe `analytics-service` binary skeleton and extracted the Analytics-owned reporting runtime backed by projection/read-model ports. The runtime explicitly registers only the Analytics reporting subset of `AdminService` and does not introduce transitional service-read adapters.

## Modified Files

- `logic-grpc-service/cmd/analytics-service/main.go`: added `--check`, `--describe`, and fail-closed default execution.
- `logic-grpc-service/internal/analytics/interfaces/reporting_server.go`: added the Analytics reporting `AdminService` subset adapter.
- `logic-grpc-service/internal/analytics/interfaces/reporting_server_test.go`: verified forwarding, dependency validation, and current service compatibility.
- `logic-grpc-service/internal/analytics/runtime/skeleton.go`: added descriptor, projection mode, extracted API list, and safety notes.
- `logic-grpc-service/internal/analytics/runtime/skeleton_test.go`: verified descriptor invariants, projection/read-model mode, no traffic, and no Identity-owned audit claim.
- `logic-grpc-service/internal/analytics/runtime/runtime.go`: added runtime construction, validation, and explicit gRPC registration.
- `logic-grpc-service/internal/analytics/runtime/runtime_test.go`: verified runtime service registration and required dependency validation.
- `docs/backend-ddd-microservices-evolution-analytics-service-extraction.md`: documented extracted APIs, projection contract, compatibility, and verification.
- `docs/backend-ddd-microservices-evolution-service-binary-convention.md`: added `analytics-service` to compiled skeletons.
- `.knowledge/architecture/service-boundaries.md`: documented Analytics runtime extraction and projection/read-model requirement.
- `.knowledge/runbooks/service-binary-convention.md`: documented the Analytics service skeleton/runtime.
- `.knowledge/manifest.yaml`: added Analytics service docs, command, interfaces, and runtime paths to service-binary routing.
- `.spec/backend-ddd-microservices-evolution/pipeline-state.json`: recorded TASK baseline and completion state.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-044-report.md`: this report.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-044-evidence.json`: machine-readable evidence.

## Scope

Scope check passed. All changed files are allowed by TASK-BDME-044 scope. No frontend, dependency manifest, `go.mod`, `go.sum`, database schema, protobuf source/generated code, gateway routing, deployment traffic, SPEC/SDD/TASK, acceptance, prompt, or Harness script files were modified. No `logic-grpc-service/internal/analytics/*service_read*` file was created.

Human confirmation was required and is recorded as satisfied by the user's global gate override.

## SPEC / SDD / Acceptance

- SPEC comparison: aligned with FR-020 and AC-014 by keeping Analytics extraction tied to projection/read-model ports and avoiding transitional service-read APIs.
- SDD comparison: aligned with target service architecture and shadow/cutover strategy by adding explicit runtime registration only.
- Acceptance comparison: the TASK goal is implemented as scoped; existing behavior remains compatible; report, evidence, checks, risks, and knowledge impact are recorded.

## Checks

- `git diff --name-only && git ls-files --others --exclude-standard`: passed.
- `cd logic-grpc-service && go test ./internal/analytics/interfaces ./internal/analytics/runtime ./cmd/analytics-service`: passed.
- `cd logic-grpc-service && go run ./cmd/analytics-service --check`: passed.
- `cd logic-grpc-service && go run ./cmd/analytics-service --describe`: passed.
- `find logic-grpc-service/internal/analytics -type f -name '*service_read*' -print`: passed; no files printed.
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 99789d836d5d6259ac4f516ef74dcbb45997689b --json`: passed; update required, no coverage gaps.
- `node .knowledge/scripts/validate-knowledge.mjs --root .`: passed.
- `cd logic-grpc-service && go test ./...`: passed.
- `cd web-gin-service && go test ./...`: passed.
- `TASK_BASE_TREE=99789d836d5d6259ac4f516ef74dcbb45997689b bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-044`: passed.
- `git diff --check`: passed.
- `bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh`: passed.

## Knowledge Impact

Result: update_required.

Updated service-boundary, service-binary, and manifest knowledge. Reviewed system overview, local development, notification outbox, and knowledge coverage routes; no additional edits were required.

## Self-Review

Verdict: 通过.

Findings: none. The implementation registers only Analytics reporting methods, excludes Identity-owned `QueryAuthAuditLogs`, keeps `TrafficEnabled=false`, documents projection/read-model mode, and avoids forbidden service-read adapters.

## Risks

- Later Analytics gateway cutover must add rollback controls and verify projection freshness/lag before routing traffic.
- The runtime prepares the reporting surface; production projection consumer wiring remains later operational work.

## Next TASK

TASK-BDME-045 can start after this TASK is committed.
