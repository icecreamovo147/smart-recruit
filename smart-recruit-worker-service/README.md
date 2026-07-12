# smart-recruit-worker-service

Independent Worker service source root.

## Responsibility

- Own asynchronous worker runtime for outbox publishing, notifications, email, resume parsing, embeddings, agent runs, and analytics projection consumers.
- Treat RabbitMQ as a hard readiness dependency for active consumers.
- Keep workers independently scalable from request-serving services.

## Startup

Later TASKs add worker runtime, consumer toggles, health/readiness endpoints, metrics, tracing, and Docker support. Until then, this root is a scaffolded Go module.

## Monolith Relationship

Current worker execution remains in `logic-grpc-service --worker-only` until worker extraction and smoke evidence is complete.
