# TASK-BDME-036 Report - Identity Gateway Cutover

## Summary

Added rollback-safe gateway routing controls for Identity authentication and authorization traffic. Default mode keeps traffic on `GRPC_ADDR`; `IDENTITY_ROUTE_MODE=identity` routes AuthService plus the Identity-owned AdminService subset to `IDENTITY_GRPC_ADDR` and fails fast when the target address is missing.

## Modified Files

- `web-gin-service/config/config.go`: added `IDENTITY_ROUTE_MODE` and `IDENTITY_GRPC_ADDR` config loading and validation.
- `web-gin-service/config/config_test.go`: covered default logic routing, extracted Identity routing, missing address validation, and invalid mode rejection.
- `web-gin-service/main.go`: passes Identity routing options to gateway gRPC client construction and logs selected route mode/target.
- `web-gin-service/rpc/client.go`: added optional Identity gRPC connection, routed AuthService to the Identity target in cutover mode, added Identity health readiness checks, and preserved rollback defaults.
- `web-gin-service/rpc/client_test.go`: verified Identity cutover connection selection, validation failures, and readiness health behavior.
- `web-gin-service/rpc/identity_admin_client.go`: added a hybrid AdminService client that sends Identity-owned AdminService methods to the Identity target while leaving unrelated AdminService calls on the logic target.
- `deploy/k8s/configmap.yaml`: added rollback-safe Identity routing defaults.
- `docker/docker-compose.yml`: exposed rollback-safe Identity routing env defaults for local composition.
- `docs/backend-ddd-microservices-evolution-identity-gateway-cutover.md`: documented routing modes, routed clients, rollback, and verification.
- `.knowledge/architecture/api-contracts-and-gateway.md`: documented the Identity gateway switch and hybrid AdminService routing.
- `.knowledge/architecture/auth-rbac-security.md`: documented Identity auth/RBAC cutover behavior.
- `.knowledge/runbooks/service-binary-convention.md`: documented the Identity route switch in the service binary runbook.
- `.knowledge/manifest.yaml`: routed the Identity gateway cutover document through service-binary knowledge.
- `.spec/backend-ddd-microservices-evolution/pipeline-state.json`: recorded TASK baseline and completion state.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-036-report.md`: this report.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-036-evidence.json`: machine-readable evidence.

## Scope

Scope check passed. All changed files are allowed by TASK-BDME-036 scope. No frontend, dependency manifest, `go.mod`, `go.sum`, database schema, protobuf source/generated code, cookie format, SPEC/SDD/TASK, acceptance, prompt, or Harness script files were modified.

Human confirmation was required by the TASK and is recorded as satisfied by the user's global gate override.

## SPEC / SDD / Acceptance

- SPEC comparison: aligned with security and safety requirements by preserving existing auth/RBAC behavior unless `IDENTITY_ROUTE_MODE=identity` is explicitly configured, and by keeping rollback to `logic` as the default.
- SDD comparison: aligned with API/interface and compatibility strategy by keeping protobuf and public HTTP contracts stable while allowing direct gateway-to-Identity routing in a scoped cutover TASK.
- Acceptance comparison: the TASK goal is implemented as scoped; current behavior remains compatible by default; report, evidence, checks, risks, and knowledge impact are recorded.

## Checks

- `git diff --name-only && git ls-files --others --exclude-standard`: passed; listed tracked and untracked TASK files.
- `TASK_BASE_TREE=2c0a99eb9b1682bdfa221dec41101675d613380f bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-036`: passed.
- `bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh`: passed.
- `cd web-gin-service && go test ./...`: passed.
- `cd logic-grpc-service && go test ./...`: passed.
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 2c0a99eb9b1682bdfa221dec41101675d613380f --json`: passed; update required, no coverage gaps.
- `node .knowledge/scripts/validate-knowledge.mjs --root .`: passed.

## Knowledge Impact

Result: update_required.

Updated gateway, auth/RBAC, service-binary, and manifest knowledge for the Identity routing switch. Reviewed service-boundary, system overview, local development, notification outbox, and knowledge coverage routes; no additional knowledge edits were required.

## Self-Review

Verdict: 通过.

Findings: none. The implementation keeps rollback-safe defaults, validates missing cutover targets, routes only Identity-owned generated clients/methods, preserves non-Identity AdminService methods on the logic target, and adds focused regression tests.

## Risks

- The hybrid AdminService routing depends on the Identity runtime continuing to own the documented AdminService subset only; later changes must update both the runtime and gateway adapter together.
- Full production cutover still requires operational deployment validation against a running `identity-service`; this TASK validates compile-time behavior, config safety, readiness checks, and rollback controls.

## Next TASK

TASK-BDME-037 can start after this TASK is committed.
