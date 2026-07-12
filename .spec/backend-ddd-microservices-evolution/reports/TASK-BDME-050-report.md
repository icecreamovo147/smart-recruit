# TASK-BDME-050 Report - Readiness Worker Health And Dependency Degradation

## Summary

Implemented dependency-aware readiness semantics for backend services and workers. Logic service health now evaluates hard dependencies separately from degraded soft dependencies, while worker-only mode can expose HTTP `/livez` and `/readyz` through `WORKER_HEALTH_ADDR`. Kubernetes worker probes now use the health HTTP endpoint instead of process-only `kill -0`, and RabbitMQ is explicitly soft for request-serving logic but hard for worker readiness.

## Modified Files

- `logic-grpc-service/server/health.go`: added reusable readiness reports, dependency severity/status modeling, worker health HTTP server, and service-vs-worker dependency semantics.
- `logic-grpc-service/server/health_test.go`: added tests for RabbitMQ degraded service readiness, hard worker readiness failure, and hard MySQL failure.
- `logic-grpc-service/mq/rabbitmq.go`: added `IsReady` so readiness can distinguish reconnecting/unavailable MQ from a usable connection.
- `logic-grpc-service/config/config.go`, `logic-grpc-service/config/config.example.yaml`: added `WORKER_HEALTH_ADDR` / `observability.worker_health_addr` configuration.
- `logic-grpc-service/main.go`: starts and gracefully shuts down worker-only health server when configured.
- `deploy/k8s/configmap.yaml`, `deploy/k8s/worker-deployment.yaml`: configured worker health address, declared health/metrics ports, and replaced process-only probes with HTTP probes.
- `deploy/k8s/README-service-binaries.md`, `docs/backend-ddd-microservices-evolution-deployment-readiness-baseline.md`, `docs/backend-ddd-microservices-evolution-service-binary-convention.md`: documented implemented readiness/degradation behavior.
- `.knowledge/**`: updated readiness/service-boundary/local-development knowledge and routing.
- `.spec/backend-ddd-microservices-evolution/pipeline-state.json`, `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-050-report.md`, `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-050-evidence.json`: recorded TASK state, report, and evidence.

## Scope

Scope check passed with `TASK_BASE_TREE=2f2e822ecf5a223757d3f21f117b10f4d34490d3`. All changed files are allowed by TASK-BDME-050 scope. No frontend, dependency manifest, `go.mod`, `go.sum`, database schema, protobuf, public product API, SPEC/SDD/TASK, acceptance, prompt, or Harness script files were modified.

Human confirmation was required by this TASK and is recorded as satisfied by the user's global gate override.

## SPEC / SDD / Acceptance

- SPEC comparison: aligned with non-functional readiness and observability/debug requirements by making health checks dependency-aware.
- SDD comparison: aligned with observability/debug output and testing strategy by keeping health semantics in platform/server code and deployment manifests.
- Acceptance comparison: passed. The TASK goal is implemented as scoped; behavior changes are limited to operational readiness/degradation semantics and worker probes; required checks and evidence are recorded.

## Checks

- `git diff --name-only && git ls-files --others --exclude-standard`: passed. Listed tracked and untracked TASK files, including new health tests and report/evidence files.
- `cd logic-grpc-service && go test ./server ./mq ./config`: passed. Focused readiness, MQ state, and config tests passed.
- `cd logic-grpc-service && go test ./...`: passed. All logic-grpc-service packages passed.
- `cd web-gin-service && go test ./...`: passed. All web-gin-service packages passed.
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 2f2e822ecf5a223757d3f21f117b10f4d34490d3`: passed. impact_result: update_required; readiness and service-boundary knowledge routes matched.
- `node .knowledge/scripts/validate-knowledge.mjs --root .`: passed. knowledge_result: PASS (36 formal documents).
- `TASK_BASE_TREE=2f2e822ecf5a223757d3f21f117b10f4d34490d3 bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-050`: passed. scope_result: PASS (TASK-BDME-050).
- `bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh`: passed. Feature validation, gofmt, logic go test, and knowledge validation passed.
- `git diff --check`: passed. No whitespace errors.

## Knowledge Impact

Result: update_required.

Updated service-boundary, service-binary, local-development, and manifest routing knowledge. Reviewed routed auth, notification/outbox, knowledge coverage, and system overview documents; no further edits were required because auth, product API, and domain workflow behavior are unchanged.

## Self-Review

Verdict: 通过.

Findings: none. Readiness semantics are centralized and tested, worker probes now check real dependencies, request-serving logic preserves synchronous availability during RabbitMQ degradation, and no forbidden surfaces were changed.

## Risks

- Worker readiness now fails when RabbitMQ is unavailable; rollout environments must ensure RabbitMQ is reachable before expecting worker pods to become Ready.
- Request-serving gRPC health still exposes only SERVING/NOT_SERVING, so degraded RabbitMQ detail is not visible through the standard gRPC health response.
- Queue backlog, dead-letter counts, and per-consumer heartbeat semantics remain follow-up work.

## Next TASK

TASK-BDME-051 can start after this TASK is committed.
