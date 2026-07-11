# Backend DDD Microservices Evolution Deployment Readiness Baseline

Status: TASK-BDME-007 baseline, updated by TASK-BDME-050 runtime readiness implementation
Last verified: 2026-07-11

## Purpose

This document records the current deployment, local-development, liveness, and
readiness behavior before later DDD and microservice extraction tasks change
runtime shape. It is descriptive only; TASK-BDME-007 does not change manifests,
traffic routing, probes, service contracts, database schema, or local scripts.

## Current Deployment Units

### Kubernetes Gateway

Source: `deploy/k8s/web-deployment.yaml`, `deploy/k8s/web-service.yaml`, and
`web-gin-service/router/router.go`.

- Deployment name: `web-gin-service`.
- Namespace: `recruitment`.
- Replicas: `2`.
- Container image: `recruitment/web-gin-service:latest`.
- Container port: `8080`, named `http`.
- Service: `ClusterIP` on port `8080`.
- Rollout strategy: rolling update with `maxUnavailable: 1` and `maxSurge: 1`.
- Pod disruption budget: `minAvailable: 1`.
- Configuration source: `web-config` ConfigMap plus `recruitment-secrets`.
- Readiness probe: HTTP GET `/readyz` on the `http` port.
- Liveness probe: HTTP GET `/livez` on the `http` port.

The gateway process also exposes `/health`. `/health` and `/livez` return a
static `200 {"status":"ok"}` response. `/readyz` creates a 2 second context and
checks both gateway-to-logic gRPC health and Redis ping. It returns `200` with
dependency details when both checks pass, otherwise `503` with
`status: not_ready`.

### Kubernetes Logic Service

Source: `deploy/k8s/logic-deployment.yaml`, `deploy/k8s/logic-service.yaml`,
`logic-grpc-service/main.go`, and `logic-grpc-service/server/health.go`.

- Deployment name: `logic-grpc-service`.
- Namespace: `recruitment`.
- Replicas: `2`.
- Container image: `recruitment/logic-grpc-service:latest`.
- Container port: `50051`, named `grpc`.
- Service: `ClusterIP` on port `50051`.
- Rollout strategy: rolling update with `maxUnavailable: 1` and `maxSurge: 1`.
- Pod disruption budget: `minAvailable: 1`.
- Configuration source: `logic-config` ConfigMap plus `recruitment-secrets`.
- Worker behavior: `DISABLE_BACKGROUND_WORKERS=true`, so request-serving pods
  do not start background consumers.
- Readiness probe: TCP socket probe on the `grpc` port. The probe is TCP-based
  because internal gRPC TLS is required in Kubernetes and the built-in gRPC probe
  is plaintext.
- Liveness probe: TCP socket probe on the `grpc` port.
- Metrics port: `9091`, named `metrics`, when `METRICS_ADDR=:9091`.

The logic gRPC health server returns `SERVING` when hard dependencies are ready
and may still return `SERVING` in a degraded state when RabbitMQ is unavailable.
MySQL and configured Redis are hard dependencies for request-serving readiness.
RabbitMQ is a soft dependency for request-serving pods: Outbox can accumulate
pending events and reconnect loops can recover without removing the HTTP/gRPC
serving path from rotation.

The logic service still performs startup side effects before serving traffic:
configuration validation, internal token validation, MySQL connection and
migration execution, repository/service assembly, object-storage initialization,
AI client initialization, optional SMTP sender initialization, default prompt
and agent seeding, initial-admin bootstrap, and legacy RBAC migration.

### Kubernetes Worker

Source: `deploy/k8s/worker-deployment.yaml` and `logic-grpc-service/main.go`.

- Deployment name: `logic-worker`.
- Namespace: `recruitment`.
- Replicas: `1`.
- Container image: `recruitment/logic-grpc-service:latest`.
- Command: `/usr/local/bin/logic-grpc-service --worker-only`.
- Configuration source: `logic-config` ConfigMap plus `recruitment-secrets`.
- Pod disruption budget: `maxUnavailable: 1`.
- Container ports: `9091` named `metrics` and `9092` named `health`.
- Readiness probe: HTTP GET `/readyz` on the `health` port.
- Liveness probe: HTTP GET `/livez` on the `health` port.

Worker-only mode starts background workers and then waits for termination
signals instead of starting the gRPC server. TASK-BDME-050 adds a worker health
HTTP listener controlled by `WORKER_HEALTH_ADDR`. `/livez` reports process
liveness. `/readyz` checks MySQL, configured Redis, and RabbitMQ as hard worker
dependencies; if RabbitMQ is disconnected or reconnecting, worker readiness
returns `503 not_ready` while the process remains live.

### Network Access

Source: `deploy/k8s/network-policy.yaml`.

`logic-grpc-ingress` allows TCP port `50051` ingress to
`logic-grpc-service` pods from pods labeled `web-gin-service` and
`logic-worker`. The file also contains a commented default-deny ingress example;
it is not active unless an operator enables it.

## Local Development Workflow

Source: `start-dev.sh`, `stop-dev.sh`, `docker/docker-compose.yml`, and
`docker/.env.example`.

There are two current local paths:

1. `./start-dev.sh` builds and starts local Go binaries plus frontend dev
   servers after checking that MySQL, Redis, and RabbitMQ are already listening
   on `127.0.0.1:3306`, `127.0.0.1:6379`, and `127.0.0.1:5672`.
2. `cd docker && docker-compose up -d --build` starts the full containerized
   stack: MySQL, Redis, RabbitMQ, logic gRPC service, web gateway, and three
   frontend containers.

`./stop-dev.sh` stops processes started by `start-dev.sh` and falls back to
killing listeners on the known frontend, gateway, and gRPC ports. The Docker
Compose path uses `.env.example` as a placeholder template and requires
`docker/.env` for local secret values; the live `.env` file must remain
untracked and was not read or modified for this baseline.

Docker Compose infrastructure health checks:

- MySQL: `mysqladmin ping -h localhost`.
- Redis: `redis-cli ping`.
- RabbitMQ: `rabbitmq-diagnostics check_port_connectivity`.

Docker Compose service ordering waits for MySQL, Redis, and RabbitMQ health
before starting `logic-grpc-service`, and starts `web-gin-service` after the
logic container has started. Compose does not currently declare HTTP/gRPC
readiness checks for application containers.

## Hard Dependencies

### Gateway

- Configuration and secrets: `JWT_SECRET`, gRPC address, internal auth settings,
  and cookie/security settings must load at process start.
- Logic gRPC health: `/readyz` fails when `Clients.Ready` cannot call the logic
  gRPC health service or receives a non-`SERVING` status.
- Redis readiness: `/readyz` fails when Redis ping fails.

Runtime nuance: several gateway paths have degraded Redis behavior, such as
rate-limit fallback to memory and JWT validation behavior documented in code,
but the current Kubernetes readiness contract still treats Redis as hard for
serving readiness.

### Logic gRPC Service

- Configuration and secrets: production config validation requires real secrets
  and internal auth token configuration.
- MySQL: process startup opens MySQL, applies migrations, and gRPC health fails
  when MySQL ping fails.
- Object storage: startup fails if object storage initialization fails.
- AI client: startup fails when AI client initialization fails.
- Redis: when configured, gRPC health fails if Redis ping fails; Redis also backs
  presign sessions, notification unread cache, and job cache.
- Internal gRPC auth: internal token validation runs before serving.

### Worker Runtime

- Configuration and secrets are shared with `logic-grpc-service`.
- MySQL is required because worker startup uses the same application assembly,
  repositories, migrations, and outbox state.
- RabbitMQ is required for active consumer delivery and reconnect loops. In
  worker-only readiness it is a hard dependency; in request-serving logic
  readiness it is degraded so synchronous traffic can continue while Outbox
  accumulates and reconnect loops recover.
- Redis is required when configured for the cache-backed workflows used by the
  shared service assembly.
- Object storage, AI client, SMTP configuration, and frontend URL affect
  specific workers such as resume parsing, embedding/agent runs, and email.

## Soft Dependencies and Degradation

- RabbitMQ initial or runtime unavailability: request-serving logic startup logs
  a warning and continues; Outbox can accumulate pending events. Worker-only
  readiness fails until RabbitMQ is connected because consumers cannot make
  progress without queue connectivity.
- SMTP: when `SMTP_REQUIRED` is false and no SMTP host is configured, email uses
  non-sending/logging behavior instead of failing startup.
- Redis runtime failures: some gateway middleware can fall back or fail closed
  depending on security semantics, but readiness currently fails on Redis ping.
- AI provider behavior: AI configuration includes timeout, retry, concurrency,
  and circuit-breaker settings, but full metrics and readiness semantics are
  still future work.
- Object-storage operation failures surface at request or worker execution time
  after successful startup and must be diagnosed from logs until richer
  telemetry is added.

## Readiness Gaps for Later Tasks

- Worker readiness now reflects hard dependency availability for MySQL,
  configured Redis, and RabbitMQ. It still does not expose queue backlog,
  dead-letter accumulation, or per-consumer heartbeat state.
- Logic gRPC readiness checks MySQL/configured Redis as hard dependencies and
  treats RabbitMQ as degraded. gRPC health still only returns SERVING or
  NOT_SERVING; structured dependency details are available in worker HTTP
  readiness but not in the request-serving gRPC health response.
- Gateway readiness returns dependency details, but those details are not yet
  connected to metrics or tracing.
- Docker Compose lacks application-level health checks for the gateway and
  logic containers.
- Startup side effects such as migrations, seeding, and legacy RBAC migration
  still run in request-serving logic pods and need review before larger
  multi-replica production rollouts.
- Queue backlog indicators, retry/dead-letter counters, and per-worker semantic
  heartbeat health are planned by later observability/readiness tasks.

## Migration Notes

- Current deployment already separates gateway, request-serving logic, and
  worker pods, which supports the staged migration path.
- Later service extraction tasks should preserve the existing gateway/logic
  readiness contracts until the replacement service has equivalent or stronger
  liveness, readiness, logging, rollback, and dependency evidence.
- Traffic routing or probe behavior changes must be handled in their own scoped
  TASK with explicit evidence; this baseline records current behavior only.
