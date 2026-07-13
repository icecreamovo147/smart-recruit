# smart-recruit-worker-service

Independent Worker service source root.

## Responsibility

- Own asynchronous worker runtime for outbox publishing, notifications, email, resume parsing, embeddings, agent runs, and analytics projection consumers.
- Treat RabbitMQ as a hard readiness dependency for active consumers.
- Keep workers independently scalable from request-serving services.

## Startup

```bash
GOWORK=off go test ./...
GOWORK=off go run ./cmd/worker-service --check
GOWORK=off go run ./cmd/worker-service --serve
```

`WORKER_WORKLOADS` enables a comma-separated subset of workloads. Empty means all known workloads. `WORKER_DISABLED_WORKLOADS` removes workloads from that set. Active workloads require RabbitMQ readiness and a configured starter with an idempotency policy.

Worker execution is controlled by explicit workload toggles and readiness checks. Outbox, Inbox, DLQ, notification, email, resume parsing, embedding, agent-run, and analytics projection workloads belong in this service root.

The workload profile and owner-contract inventory lives in `internal/docs/worker_workload_inventory.md`.
