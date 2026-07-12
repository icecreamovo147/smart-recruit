# Worker Service Decomposition

## Scope

TASK-BDME-045 creates a compile-safe `worker-services` command and worker workload descriptor. It does not replace the active `logic-grpc-service --worker-only` deployment, start queue consumers, change queue bindings, or alter public API behavior.

## Worker Workloads

The descriptor names these independently scalable future worker workloads:

- `outbox-dispatcher`
- `notification-consumer`
- `email-consumer`
- `resume-parse-consumer`
- `embedding-consumer`
- `agent-run-consumer`
- `analytics-projection-consumer`

## Runtime Behavior

`logic-grpc-service/cmd/worker-services` supports:

```bash
go run ./cmd/worker-services --check
go run ./cmd/worker-services --describe
```

Default execution exits non-zero and does not bind a listener or start consumers. Current worker production behavior remains `logic-grpc-service --worker-only`.

## Cutover Requirements

A later worker cutover must define queue ownership, readiness, liveness, backlog metrics, dead-letter diagnostics, graceful shutdown, and rollback evidence for each workload before moving active queue consumption away from the monolith worker deployment.

## Verification

Run:

```bash
cd logic-grpc-service && go test ./internal/platform/workers/runtime ./cmd/worker-services
cd logic-grpc-service && go run ./cmd/worker-services --check
cd logic-grpc-service && go run ./cmd/worker-services --describe
cd logic-grpc-service && go test ./...
```
