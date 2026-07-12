# smart-recruit-notification-service

Independent Notification service source root.

## Responsibility

- Own notification persistence, unread counts, realtime delivery integration, email delivery integration, and notification event handling.
- Preserve current notification and SSE semantics through gateway compatibility.
- Use RabbitMQ/outbox paths for asynchronous side effects.

## Startup

Later TASKs add service runtime, config, health, metrics, tracing, consumers, and Docker support. Until then, this root is a scaffolded Go module.

## Monolith Relationship

Notification traffic and workers remain on the monolith until extraction and route-mode cutover evidence is complete.
