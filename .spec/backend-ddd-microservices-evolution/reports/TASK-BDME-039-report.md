# TASK-BDME-039 Report - Recruitment Gateway Cutover

## Summary

Added rollback-safe gateway routing controls for Recruitment APIs. Default mode keeps JobService, CandidateService, and ApplicationService traffic on `GRPC_ADDR`; `RECRUITMENT_ROUTE_MODE=recruitment` routes those generated clients to `RECRUITMENT_GRPC_ADDR` and fails fast when the target address is missing.

## Modified Files

- `web-gin-service/config/config.go`: added `RECRUITMENT_ROUTE_MODE` and `RECRUITMENT_GRPC_ADDR` config loading and validation.
- `web-gin-service/config/config_test.go`: covered default logic routing, extracted Recruitment routing, missing address validation, and invalid mode rejection.
- `web-gin-service/main.go`: passes Recruitment routing options into gRPC client construction and logs selected route mode/target.
- `web-gin-service/rpc/client.go`: added optional Recruitment gRPC connection, routed JobService/CandidateService/ApplicationService to the Recruitment target in cutover mode, and added Recruitment readiness checks.
- `web-gin-service/rpc/client_test.go`: verified Recruitment route selection, validation failures, and readiness health behavior.
- `deploy/k8s/configmap.yaml`: added rollback-safe Recruitment routing defaults.
- `docker/docker-compose.yml`: exposed rollback-safe Recruitment routing env defaults for local composition.
- `docs/backend-ddd-microservices-evolution-recruitment-gateway-cutover.md`: documented routing modes, routed clients, rollback, and verification.
- `.knowledge/architecture/api-contracts-and-gateway.md`: documented the Recruitment gateway switch.
- `.knowledge/architecture/service-boundaries.md`: documented default monolith routing and explicit Recruitment route mode.
- `.knowledge/domains/recruitment.md`: documented Recruitment route mode behavior.
- `.knowledge/domains/recruitment-lifecycle.md`: documented ApplicationService route mode behavior and rollback.
- `.knowledge/runbooks/service-binary-convention.md`: documented the Recruitment gateway route switch.
- `.knowledge/manifest.yaml`: routed the Recruitment gateway cutover document through service-binary knowledge.
- `.spec/backend-ddd-microservices-evolution/pipeline-state.json`: recorded TASK baseline and completion state.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-039-report.md`: this report.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-039-evidence.json`: machine-readable evidence.

## Scope

Scope check passed. All changed files are allowed by TASK-BDME-039 scope. No frontend, dependency manifest, `go.mod`, `go.sum`, database schema, protobuf source/generated code, public HTTP contract, SPEC/SDD/TASK, acceptance, prompt, or Harness script files were modified.

Human confirmation was required and is recorded as satisfied by the user's global gate override.

## SPEC / SDD / Acceptance

- SPEC comparison: aligned with FR-002 and FR-021..FR-025 by adding explicit cutover and rollback controls while preserving default behavior.
- SDD comparison: aligned with API/interface and compatibility strategy by keeping protobuf and public HTTP contracts stable and routing only scoped generated clients.
- Acceptance comparison: the TASK goal is implemented as scoped; existing behavior remains compatible by default; report, evidence, checks, risks, and knowledge impact are recorded.

## Checks

- `git diff --name-only && git ls-files --others --exclude-standard`: passed; listed tracked and untracked TASK files.
- `cd web-gin-service && go test ./config ./rpc`: passed.
- `TASK_BASE_TREE=fdc6491610228ff62082e6d83dd3fc9ebbf7ec5a bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-039`: passed.
- `bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh`: passed.
- `cd web-gin-service && go test ./...`: passed.
- `cd logic-grpc-service && go test ./...`: passed.
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree fdc6491610228ff62082e6d83dd3fc9ebbf7ec5a --json`: passed; update required, no coverage gaps.
- `node .knowledge/scripts/validate-knowledge.mjs --root .`: passed.

## Knowledge Impact

Result: update_required.

Updated gateway, service-boundary, recruitment domain, recruitment lifecycle, service-binary, and manifest knowledge. Reviewed local development, notification outbox, system overview, and knowledge coverage routes; no additional edits were required.

## Self-Review

Verdict: 通过.

Findings: none. The implementation keeps rollback-safe defaults, validates missing cutover targets, routes only Recruitment-owned generated clients, preserves other service targets, and adds focused regression tests.

## Risks

- The gateway switch assumes a healthy `recruitment-service` target is deployed and serving the extracted Job/Candidate/Application APIs before `RECRUITMENT_ROUTE_MODE=recruitment` is enabled.
- Rollback is configuration-only (`RECRUITMENT_ROUTE_MODE=logic`) but still requires a gateway restart or rollout to take effect.

## Next TASK

TASK-BDME-040 can start after this TASK is committed.
