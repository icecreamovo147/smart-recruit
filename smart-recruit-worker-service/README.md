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

## Monolith Relationship

Current worker execution remains rollback-safe because workload startup is controlled by explicit toggles and readiness checks. Full Outbox/Inbox/DLQ handler cutover is completed in later runtime TASKs.
