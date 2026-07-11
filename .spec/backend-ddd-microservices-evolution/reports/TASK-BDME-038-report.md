# TASK-BDME-038 Report - Recruitment Service API Extraction

## Summary

Extracted the Recruitment-owned generated gRPC API surface into `internal/recruitment/runtime` without gateway cutover. The new runtime can explicitly register JobService, CandidateService, and ApplicationService through thin adapters, while `cmd/recruitment-service` remains fail-closed by default and receives no production traffic.

## Modified Files

- `logic-grpc-service/internal/recruitment/interfaces/job_api.go`: added the job taxonomy API needed to cover the full generated JobService surface.
- `logic-grpc-service/internal/recruitment/interfaces/job_server.go`: added the generated JobService adapter for Recruitment-owned job and taxonomy methods.
- `logic-grpc-service/internal/recruitment/interfaces/candidate_server.go`: added the generated CandidateService adapter for profile and resume methods.
- `logic-grpc-service/internal/recruitment/interfaces/application_server.go`: added the generated ApplicationService adapter for application lifecycle methods.
- `logic-grpc-service/internal/recruitment/interfaces/job_api_test.go`: verified current JobService and JobTaxonomyService satisfy Recruitment interfaces.
- `logic-grpc-service/internal/recruitment/interfaces/job_server_test.go`: verified JobService adapter forwarding and dependency validation.
- `logic-grpc-service/internal/recruitment/interfaces/candidate_application_server_test.go`: verified CandidateService and ApplicationService adapter forwarding and dependency validation.
- `logic-grpc-service/internal/recruitment/runtime/runtime.go`: added Recruitment runtime construction, validation, and explicit gRPC registration for JobService, CandidateService, and ApplicationService.
- `logic-grpc-service/internal/recruitment/runtime/runtime_test.go`: verified runtime service registration and required dependency validation.
- `logic-grpc-service/internal/recruitment/runtime/skeleton.go`: updated the descriptor to list extracted APIs and explicit runtime registration mode.
- `logic-grpc-service/internal/recruitment/runtime/skeleton_test.go`: updated descriptor assertions for extracted APIs and no default traffic.
- `logic-grpc-service/cmd/recruitment-service/main.go`: prints `extracted_apis` in `--describe`; default execution remains unrouted.
- `docs/backend-ddd-microservices-evolution-recruitment-api-extraction.md`: documented extracted APIs, runtime behavior, compatibility, and verification.
- `docs/backend-ddd-microservices-evolution-recruitment-service-skeleton.md`: updated skeleton documentation for explicit runtime registration.
- `.knowledge/architecture/service-boundaries.md`: documented the Recruitment runtime registration boundary and no-traffic state.
- `.knowledge/domains/recruitment.md`: documented current monolith behavior and explicit Recruitment runtime registration.
- `.knowledge/domains/recruitment-lifecycle.md`: documented ApplicationService runtime registration without gateway traffic.
- `.knowledge/runbooks/service-binary-convention.md`: documented explicit Recruitment runtime registration.
- `.knowledge/manifest.yaml`: routed the Recruitment API extraction doc and interfaces/runtime paths through service-binary knowledge.
- `.spec/backend-ddd-microservices-evolution/pipeline-state.json`: recorded TASK baseline and completion state.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-038-report.md`: this report.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-038-evidence.json`: machine-readable evidence.

## Scope

Scope check passed. All changed files are allowed by TASK-BDME-038 scope. No frontend, dependency manifest, `go.mod`, `go.sum`, database schema, protobuf source/generated code, gateway routing, deployment manifest, Dockerfile, SPEC/SDD/TASK, acceptance, prompt, or Harness script files were modified.

Human confirmation was required and is recorded as satisfied by the user's global gate override.

## SPEC / SDD / Acceptance

- SPEC comparison: aligned with FR-002 and FR-021..FR-025 by extracting Recruitment service runtime APIs while preserving current behavior and avoiding traffic cutover.
- SDD comparison: aligned with API/interface and compatibility strategy by keeping protobuf contracts stable and adding explicit runtime registration only.
- Acceptance comparison: the TASK goal is implemented as scoped; existing behavior remains compatible; report, evidence, checks, risks, and knowledge impact are recorded.

## Checks

- `git diff --name-only && git ls-files --others --exclude-standard`: passed; listed tracked and untracked TASK files.
- `cd logic-grpc-service && go test ./internal/recruitment/interfaces ./internal/recruitment/runtime ./cmd/recruitment-service`: passed.
- `cd logic-grpc-service && go run ./cmd/recruitment-service --check`: passed.
- `cd logic-grpc-service && go run ./cmd/recruitment-service --describe`: passed; printed `traffic_enabled: false`, `cutover_mode: none`, and extracted API list.
- `TASK_BASE_TREE=851ccb5f2509c98a9ba324e764033d8b6f656424 bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-038`: passed.
- `bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh`: passed.
- `cd logic-grpc-service && go test ./...`: passed.
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 851ccb5f2509c98a9ba324e764033d8b6f656424 --json`: passed; update required, no coverage gaps.
- `node .knowledge/scripts/validate-knowledge.mjs --root .`: passed.

## Knowledge Impact

Result: update_required.

Updated service-binary, service-boundary, recruitment domain, recruitment lifecycle, and manifest knowledge. Reviewed local development, notification outbox, system overview, and knowledge coverage routes; no additional edits were required.

## Self-Review

Verdict: 通过.

Findings: none. The runtime registers only Recruitment-owned generated services when explicitly called, keeps gateway traffic on the monolith, preserves protobuf/schema/public HTTP contracts, includes focused adapter/runtime tests, and records the remaining cutover risk for TASK-BDME-039.

## Risks

- `cmd/recruitment-service` still does not start a real gRPC listener for Recruitment APIs; later startup wiring must validate DB, OSS, outbox, authz scope, and readiness before traffic.
- TASK-BDME-039 must add gateway rollback controls and production validation before routing Recruitment traffic away from the monolith.

## Next TASK

TASK-BDME-039 can start after this TASK is committed.
