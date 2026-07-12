# TASK-BDME-029 Report - Notification Gateway Cutover

## Summary

Implemented a rollback-safe gateway routing control for Notification APIs. The gateway keeps the existing logic gRPC route by default and can route only the generated Notification client to an extracted Notification service when explicitly configured.

## Modified Files

- `web-gin-service/config/config.go`: added `NOTIFICATION_ROUTE_MODE` and `NOTIFICATION_GRPC_ADDR` loading with fail-fast validation.
- `web-gin-service/config/config_test.go`: covered default logic routing, extracted-service routing, missing address, and invalid mode validation.
- `web-gin-service/rpc/client.go`: added Notification-specific client connection routing, target metadata, close handling, and readiness checks for both logic and notification targets when cut over.
- `web-gin-service/rpc/client_test.go`: covered route target selection and dual-target readiness behavior.
- `web-gin-service/main.go`: passed Notification routing options into gRPC client construction and logged the selected target.
- `deploy/k8s/configmap.yaml`: added rollback-safe default Notification route settings.
- `docker/docker-compose.yml`: added local override knobs with default `logic` route mode.
- `docs/backend-ddd-microservices-evolution-notification-gateway-cutover.md`: documented routing controls, realtime Redis compatibility, rollback, and verification.
- `.knowledge/architecture/api-contracts-and-gateway.md`: documented gateway config/rpc ownership and Notification route control.
- `.knowledge/domains/notification-outbox.md`: documented Notification API routing and SSE compatibility.
- `.knowledge/runbooks/service-binary-convention.md`: documented gateway cutover review expectations.
- `.knowledge/manifest.yaml`: routed new cutover docs and gateway files to relevant knowledge.
- `.spec/backend-ddd-microservices-evolution/pipeline-state.json`: recorded TASK progress and completion.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-029-report.md`: this report.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-029-evidence.json`: machine-readable evidence.

## Scope

Scope check passed. All changed files are allowed by TASK-BDME-029 scope. No frontend, dependency manifest, `go.mod`, `go.sum`, database schema, auth policy, protobuf, or SPEC/SDD/TASK/Harness contract files were modified.

The user explicitly authorized future human gates and authorized expanding the TASK scope for out-of-scope Go formatting if `agent-check.sh` required it. No additional out-of-scope formatting was required for this TASK.

## SPEC / SDD / Acceptance

- SPEC comparison: aligned with FR-002 and FR-021..FR-025 by preserving bounded service extraction compatibility and rollback-safe behavior.
- SDD comparison: aligned with API/interface compatibility strategy by keeping public HTTP and protobuf contracts unchanged while adding gateway-to-service route control.
- Acceptance comparison: implemented gateway Notification route cutover controls, preserved existing behavior by default, documented realtime delivery compatibility, and recorded tests/risks/knowledge impact.

## Checks

- `git diff --name-only && git ls-files --others --exclude-standard`: passed; listed tracked and untracked TASK files.
- `node .knowledge/scripts/validate-knowledge.mjs --root .`: passed.
- `cd web-gin-service && go test ./config ./rpc`: passed.
- `cd logic-grpc-service && go test ./...`: passed.
- `cd web-gin-service && go test ./...`: passed.
- `TASK_BASE_TREE=b7b905153ca53eea1cf3b7bf7618a4345fc82514 bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-029`: passed.
- `bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh`: passed.
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree b7b905153ca53eea1cf3b7bf7618a4345fc82514 --json`: passed; update required, no coverage gaps.

## Self-Review

Verdict: 通过.

Findings: none open after one fix-check round. Self-review identified that gateway readiness should validate the extracted Notification target when route mode uses a separate connection; the fix was implemented and covered by tests.

## Risks

- Enabling `NOTIFICATION_ROUTE_MODE=notification` requires a reachable Notification gRPC service implementing the existing `NotificationService` contract. Checked-in defaults do not enable this route.
- SSE delivery remains Redis-channel based; extracted runtime deployments must share the same Redis notification channel contract.
- Rollback is configuration-only: set `NOTIFICATION_ROUTE_MODE=logic` and restart the gateway.

## Next TASK

TASK-BDME-030 can start after this TASK is committed.
