# TASK-BDME-048 Report - Internal Service Security Hardening

## Summary

Hardened internal gRPC service-to-service security with token plus TLS controls. Logic gRPC can now serve with TLS when cert/key files are configured, web-gin gRPC clients use CA-backed TLS when configured, and both services fail fast when `GRPC_INTERNAL_TLS=required` is missing required certificate material. Local Docker remains compatible with `GRPC_INTERNAL_TLS=optional`; Kubernetes examples require internal TLS and mount a dedicated TLS secret.

## Modified Files

- `logic-grpc-service/config/config.go`: added gRPC TLS cert/key config and production TLS validation.
- `logic-grpc-service/server/transport_security.go`: added TLS server credential option construction.
- `logic-grpc-service/main.go`: wired TLS server option into gRPC startup and logs TLS state.
- `logic-grpc-service/config/config.example.yaml`: documented TLS cert/key config keys.
- `logic-grpc-service/config/config_test.go`, `logic-grpc-service/server/transport_security_test.go`: added production TLS validation and server option tests.
- `web-gin-service/config/config.go`: added internal TLS mode, CA file, and server-name config validation.
- `web-gin-service/rpc/client.go`: added TLS/insecure transport credential selection and fail-fast required TLS behavior.
- `web-gin-service/main.go`: passed TLS config to gRPC clients and logged TLS enabled state.
- `web-gin-service/config/config_test.go`, `web-gin-service/rpc/client_test.go`: added required CA and TLS client tests.
- `deploy/k8s/configmap.yaml`: enabled required internal TLS and configured cert/CA paths.
- `deploy/k8s/logic-deployment.yaml`, `deploy/k8s/web-deployment.yaml`: mounted the internal gRPC TLS secret; logic probe moved to TCP to avoid plaintext gRPC probe mismatch.
- `deploy/k8s/secret.example.yaml`: added the `recruitment-grpc-tls` example secret shape.
- `docker/docker-compose.yml`, `docker/.env.example`: added local optional TLS knobs while preserving required internal token.
- `docs/backend-ddd-microservices-evolution-internal-service-security.md`: added the internal TLS/token operating contract.
- `docs/backend-ddd-microservices-evolution-secret-config-safety.md`, `docs/backend-ddd-microservices-evolution-service-binary-convention.md`: documented production TLS/token requirements.
- `.knowledge/**`: updated security, gateway, service-boundary, local-development, service-binary, and routing knowledge.
- `.spec/backend-ddd-microservices-evolution/pipeline-state.json`: recorded TASK completion and next TASK state.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-048-report.md`: this report.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-048-evidence.json`: machine-readable evidence.

## Scope

Scope check passed. All changed files are allowed by TASK-BDME-048 scope. No frontend, dependency manifest, `go.mod`, `go.sum`, database schema, protobuf, SPEC/SDD/TASK, acceptance, prompt, or Harness script files were modified.

Human confirmation was required by the TASK and is recorded as satisfied by the user's global gate override.

## SPEC / SDD / Acceptance

- SPEC comparison: aligned with Security and Safety requirements by requiring internal token authentication and adding internal TLS controls for service-to-service gRPC traffic.
- SDD comparison: aligned with configuration design and security tests by adding explicit TLS/token env controls and tests in both services.
- Acceptance comparison: passed. The TASK goal is implemented as scoped, existing local behavior remains compatible through `GRPC_INTERNAL_TLS=optional`, and production/staged examples fail closed when TLS is required but certificate material is missing.

## Checks

- `git diff --name-only && git ls-files --others --exclude-standard`: passed.
- `cd logic-grpc-service && go test ./config ./server`: passed.
- `cd web-gin-service && go test ./config ./rpc`: passed.
- `cd logic-grpc-service && go test ./...`: passed.
- `cd web-gin-service && go test ./...`: passed.
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 8751e3c66c6cfbd0c94865527989ed6cafdbf16a --json`: passed; update required, no coverage gaps.
- `node .knowledge/scripts/validate-knowledge.mjs --root .`: passed.
- `TASK_BASE_TREE=8751e3c66c6cfbd0c94865527989ed6cafdbf16a bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-048`: passed.
- `bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh`: passed.
- `git diff --check`: passed.

## Knowledge Impact

Result: update_required.

Updated API/gateway, auth/security, service-boundary, local-development, service-binary, and manifest routing knowledge. Reviewed auth permission alignment, debug auth permissions, knowledge coverage audit, migration/persistence docs, notification outbox, and system overview; no additional edits were required.

## Self-Review

Verdict: 通过.

Findings: none. The implementation keeps local compatibility, requires configured cert material when TLS is required, preserves internal token authentication, avoids public API or schema changes, and updates deployment examples with matching secret mounts.

## Risks

- Kubernetes deployments must provide a valid `recruitment-grpc-tls` secret before using the checked-in required TLS config.
- TCP probes confirm port reachability but do not validate gRPC application health under TLS; a future TASK may add a TLS-capable health probe sidecar/tool.
- TLS cert rotation and trust bundle lifecycle remain operational responsibilities outside this TASK.

## Next TASK

TASK-BDME-049 can start after this TASK is committed.
