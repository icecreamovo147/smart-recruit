# TASK-BDME-041 Report - Interview Gateway Cutover

## Summary

Added a rollback-safe Interview gateway route switch. The gateway now keeps InterviewService traffic on the logic gRPC target by default and can explicitly route only the generated InterviewService client to `INTERVIEW_GRPC_ADDR` when `INTERVIEW_ROUTE_MODE=interview`.

## Modified Files

- `web-gin-service/config/config.go`: added `INTERVIEW_ROUTE_MODE` / `INTERVIEW_GRPC_ADDR` loading and validation.
- `web-gin-service/config/config_test.go`: verified default logic routing, extracted-service routing, missing address failure, and invalid mode failure.
- `web-gin-service/rpc/client.go`: added Interview route target selection, separate gRPC connection, readiness health check, target metadata, and close handling.
- `web-gin-service/rpc/client_test.go`: verified Interview cutover connection selection, validation failures, and readiness health failure handling.
- `web-gin-service/main.go`: passed Interview route config to RPC clients and logged route mode / target.
- `deploy/k8s/configmap.yaml`: added rollback-safe checked-in defaults.
- `docker/docker-compose.yml`: added rollback-safe local defaults.
- `docs/backend-ddd-microservices-evolution-interview-gateway-cutover.md`: documented routing modes, routed client, rollback, and verification.
- `docs/backend-ddd-microservices-evolution-service-binary-convention.md`: documented that Interview routing can be explicitly enabled and rolled back.
- `.knowledge/architecture/api-contracts-and-gateway.md`: documented the Interview route switch and target.
- `.knowledge/architecture/service-boundaries.md`: documented default monolith routing and explicit Interview cutover behavior.
- `.knowledge/domains/recruitment.md`: documented Interview route mode behavior in the recruitment domain.
- `.knowledge/domains/recruitment-lifecycle.md`: documented Interview cutover behavior and lifecycle rollback.
- `.knowledge/runbooks/service-binary-convention.md`: documented the Interview gateway routing switch.
- `.knowledge/manifest.yaml`: routed the new Interview gateway cutover doc to service-binary knowledge.
- `.spec/backend-ddd-microservices-evolution/pipeline-state.json`: recorded TASK baseline and completion state.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-041-report.md`: this report.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-041-evidence.json`: machine-readable evidence.

## Scope

Scope check passed. All changed files are allowed by TASK-BDME-041 scope. No frontend, dependency manifest, `go.mod`, `go.sum`, database schema, protobuf source/generated code, public HTTP contract, auth/RBAC behavior, SPEC/SDD/TASK, acceptance, prompt, or Harness script files were modified.

Human confirmation was required and is recorded as satisfied by the user's global gate override.

## SPEC / SDD / Acceptance

- SPEC comparison: aligned with FR-002 and FR-021..FR-025 by adding a controlled routed cutover with documented rollback and safe defaults.
- SDD comparison: aligned with API/interface and compatibility strategy by keeping HTTP/protobuf behavior stable and using gateway-level service-specific routing.
- Acceptance comparison: the TASK goal is implemented as scoped; existing behavior remains compatible by default; report, evidence, checks, risks, and knowledge impact are recorded.

## Checks

- `git diff --name-only && git ls-files --others --exclude-standard`: passed; listed tracked and untracked TASK files.
- `cd web-gin-service && go test ./config ./rpc`: passed.
- `node .knowledge/scripts/validate-knowledge.mjs --root .`: passed.
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree d82500e1abd33fe141ab16a62c32acf0899ea88a --json`: passed; update required, no coverage gaps.
- `cd web-gin-service && go test ./...`: passed.
- `cd logic-grpc-service && go test ./...`: passed.
- `TASK_BASE_TREE=d82500e1abd33fe141ab16a62c32acf0899ea88a bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-041`: passed.
- `bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh`: passed.
- `git diff --check`: passed.

## Knowledge Impact

Result: update_required.

Updated API gateway, service-boundary, recruitment domain, recruitment lifecycle, service-binary, and manifest knowledge. Reviewed system overview, local development, notification outbox, and knowledge coverage routes; no additional edits were required.

## Self-Review

Verdict: 通过.

Findings: none. The implementation follows the established gateway cutover pattern, keeps `INTERVIEW_ROUTE_MODE=logic` as the checked-in default, requires `INTERVIEW_GRPC_ADDR` before cutover, routes only InterviewService to the extracted target, preserves all other generated clients on their existing targets, and records rollback evidence.

## Risks

- Standalone Interview service serving readiness remains dependent on later operational wiring; this TASK only adds gateway target selection.
- When enabled, InterviewService lifecycle side effects must be validated against the extracted runtime dependencies and monitored before production rollout.
- Rollback is configuration-only: set `INTERVIEW_ROUTE_MODE=logic`, clear `INTERVIEW_GRPC_ADDR`, and roll the gateway.

## Next TASK

TASK-BDME-042 can start after this TASK is committed.
