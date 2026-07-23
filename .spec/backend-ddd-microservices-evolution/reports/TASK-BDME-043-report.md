# TASK-BDME-043 Report - Offer Gateway Cutover

## Summary

Added a rollback-safe Offer gateway route switch. The gateway now keeps OfferService traffic on the logic gRPC target by default and can explicitly route only the generated OfferService client to `OFFER_GRPC_ADDR` when `OFFER_ROUTE_MODE=offer`.

## Modified Files

- `web-gin-service/config/config.go`: added `OFFER_ROUTE_MODE` / `OFFER_GRPC_ADDR` loading and validation.
- `web-gin-service/config/config_test.go`: verified default logic routing, extracted-service routing, missing address failure, and invalid mode failure.
- `web-gin-service/rpc/client.go`: added Offer route target selection, separate gRPC connection, readiness health check, target metadata, and close handling.
- `web-gin-service/rpc/client_test.go`: verified Offer cutover connection selection, validation failures, and readiness health failure handling.
- `web-gin-service/main.go`: passed Offer route config to RPC clients and logged route mode / target.
- `deploy/k8s/configmap.yaml`: added rollback-safe checked-in defaults.
- `docker/docker-compose.yml`: added rollback-safe local defaults.
- `docs/backend-ddd-microservices-evolution-offer-gateway-cutover.md`: documented routing modes, routed client, rollback, and verification.
- `docs/backend-ddd-microservices-evolution-service-binary-convention.md`: documented that Offer routing can be explicitly enabled and rolled back.
- `.knowledge/architecture/api-contracts-and-gateway.md`: documented the Offer route switch and target.
- `.knowledge/architecture/service-boundaries.md`: documented default monolith routing and explicit Offer cutover behavior.
- `.knowledge/domains/recruitment.md`: documented Offer route mode behavior.
- `.knowledge/domains/recruitment-lifecycle.md`: documented Offer cutover behavior and lifecycle rollback.
- `.knowledge/runbooks/service-binary-convention.md`: documented the Offer gateway routing switch.
- `.knowledge/manifest.yaml`: routed the new Offer gateway cutover doc to service-binary knowledge.
- `.spec/backend-ddd-microservices-evolution/pipeline-state.json`: recorded TASK baseline and completion state.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-043-report.md`: this report.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-043-evidence.json`: machine-readable evidence.

## Scope

Scope check passed. All changed files are allowed by TASK-BDME-043 scope. No frontend, dependency manifest, `go.mod`, `go.sum`, database schema, protobuf source/generated code, public HTTP contract, auth/RBAC behavior, SPEC/SDD/TASK, acceptance, prompt, or Harness script files were modified.

Human confirmation was required and is recorded as satisfied by the user's global gate override.

## SPEC / SDD / Acceptance

- SPEC comparison: aligned with FR-002 and FR-021..FR-025 by adding a controlled routed cutover with documented rollback and safe defaults.
- SDD comparison: aligned with API/interface and compatibility strategy by keeping HTTP/protobuf behavior stable and using gateway-level service-specific routing.
- Acceptance comparison: the TASK goal is implemented as scoped; existing behavior remains compatible by default; report, evidence, checks, risks, and knowledge impact are recorded.

## Checks

- `git diff --name-only && git ls-files --others --exclude-standard`: passed; listed tracked and untracked TASK files.
- `cd web-gin-service && go test ./config ./rpc`: passed.
- `node .knowledge/scripts/validate-knowledge.mjs --root .`: passed.
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 0736f8c5d4f74d15712bf03a912c05c2d2d7e921 --json`: passed; update required, no coverage gaps.
- `cd web-gin-service && go test ./...`: passed.
- `cd logic-grpc-service && go test ./...`: passed.
- `TASK_BASE_TREE=0736f8c5d4f74d15712bf03a912c05c2d2d7e921 bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-043`: passed.
- `git diff --check`: passed.
- `bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh`: passed.

## Knowledge Impact

Result: update_required.

Updated API gateway, service-boundary, recruitment domain, recruitment lifecycle, service-binary, and manifest knowledge. Reviewed system overview, local development, notification outbox, and knowledge coverage routes; no additional edits were required.

## Self-Review

Verdict: 通过.

Findings: none. The implementation follows the established gateway cutover pattern, keeps `OFFER_ROUTE_MODE=logic` as the checked-in default, requires `OFFER_GRPC_ADDR` before cutover, routes only OfferService to the extracted target, preserves all other generated clients on their existing targets, and records rollback evidence.

## Risks

- Standalone Offer service serving readiness remains dependent on later operational wiring; this TASK only adds gateway target selection.
- When enabled, Offer lifecycle side effects must be validated against the extracted runtime dependencies and monitored before production rollout.
- Rollback is configuration-only: set `OFFER_ROUTE_MODE=logic`, clear `OFFER_GRPC_ADDR`, and roll the gateway.

## Next TASK

TASK-BDME-044 can start after this TASK is committed.
