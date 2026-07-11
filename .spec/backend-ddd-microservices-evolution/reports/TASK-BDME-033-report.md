# TASK-BDME-033 Report - AI Agent Gateway Cutover

## Summary

Added a gateway-side AI Agent routing switch so AI Agent-owned generated gRPC clients stay on `GRPC_ADDR` by default and can be routed to an extracted AI Agent service with `AI_AGENT_ROUTE_MODE=ai-agent` plus `AI_AGENT_GRPC_ADDR`.

## Modified Files

- `web-gin-service/config/config.go`: added `AI_AGENT_ROUTE_MODE` and `AI_AGENT_GRPC_ADDR` configuration with fail-fast validation for extracted-service routing.
- `web-gin-service/config/config_test.go`: covered default logic routing, extracted-service routing, missing address, and invalid route mode validation.
- `web-gin-service/rpc/client.go`: added a separate AI Agent gRPC connection, routed AI Agent-owned generated clients through it when enabled, included AI Agent readiness health checks, and preserved existing logic routing by default.
- `web-gin-service/rpc/client_test.go`: verified AI Agent routing, validation errors, and readiness behavior for the independent AI Agent connection.
- `web-gin-service/main.go`: passed AI Agent route configuration into gateway RPC clients and logged the active route target.
- `deploy/k8s/configmap.yaml`: documented rollback-safe deployment defaults with `AI_AGENT_ROUTE_MODE=logic` and an empty AI Agent address.
- `docker/docker-compose.yml`: added local runtime overrides for AI Agent route mode and address while keeping the default on logic.
- `docs/backend-ddd-microservices-evolution-ai-agent-gateway-cutover.md`: documented rollout modes, routed clients, checks, rollback, and residual risks.
- `.knowledge/architecture/api-contracts-and-gateway.md`: documented the gateway's AI Agent routing boundary and rollback switch.
- `.knowledge/architecture/agent-runtime.md`: documented that gateway cutover remains public-contract compatible and configuration-reversible.
- `.knowledge/runbooks/service-binary-convention.md`: added the AI Agent gateway routing switch to service binary conventions.
- `.knowledge/manifest.yaml`: routed the AI Agent gateway cutover document into service-binary knowledge review.
- `.spec/backend-ddd-microservices-evolution/pipeline-state.json`: recorded TASK completion and prepared the next TASK.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-033-report.md`: this report.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-033-evidence.json`: machine-readable evidence.

## Scope

Scope check passed. All changed files are allowed by TASK-BDME-033 scope. No frontend, dependency manifest, `go.mod`, `go.sum`, database schema, protobuf source/generated code, SPEC/SDD/TASK, acceptance, prompt, or Harness script files were modified.

Human confirmation was required by the TASK and is recorded as satisfied by the user's global gate override.

## SPEC / SDD / Acceptance

- SPEC comparison: aligned with FR-002 and FR-021..FR-025 by introducing controlled gateway routing for an extracted backend service, retaining feature-flag-style rollback, and preserving public behavior by default.
- SDD comparison: aligned with API/interface and compatibility strategy by keeping HTTP/protobuf contracts stable while replacing selected generated-client targets through gateway routing configuration.
- Acceptance comparison: the AI Agent gateway cutover is implemented exactly as scoped, rollback evidence is documented, all required checks passed, and knowledge impact is recorded.

## Checks

- `git diff --name-only && git ls-files --others --exclude-standard`: passed; listed tracked and untracked TASK files.
- `TASK_BASE_TREE=041eec41977ce2cbf78da0862035fb432a2de390 bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-033`: passed.
- `bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh`: passed.
- `cd logic-grpc-service && go test ./...`: passed.
- `cd web-gin-service && go test ./...`: passed.
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 041eec41977ce2cbf78da0862035fb432a2de390 --json`: passed; update required, no coverage gaps.
- `node .knowledge/scripts/validate-knowledge.mjs --root .`: passed.

## Knowledge Impact

Result: update_required.

Reviewed and updated gateway/runtime/service-binary knowledge to include the AI Agent routing switch and rollback path. Related triggered knowledge documents with no required changes were reviewed for local development, service boundaries, notification outbox, system overview, and knowledge coverage.

## Self-Review

Verdict: 通过.

Findings: none. The implementation preserves public HTTP/protobuf behavior, keeps `logic` as the default target, requires a target address before extracted-service routing, and adds readiness checking for the independent AI Agent connection.

## Risks

- Runtime traffic remains on the monolith unless operators explicitly enable `AI_AGENT_ROUTE_MODE=ai-agent`; rollout metrics and parity evidence still need to be collected in the target environment.
- The gateway still uses the existing protobuf surface. That is intentional for this TASK, but future service-contract evolution must remain explicitly scoped.

## Next TASK

TASK-BDME-034 can start after this TASK is committed.
