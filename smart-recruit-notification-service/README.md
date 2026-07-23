# smart-recruit-notification-service

Independent Notification service source root.

## Responsibility

- Own notification persistence, unread counts, realtime delivery integration, email delivery integration, and notification event handling.
- Preserve current notification and SSE semantics through gateway compatibility.
- Use RabbitMQ/outbox paths for asynchronous side effects.

## Startup

This module now contains an independently buildable Notification gRPC runtime:

```bash
GOWORK=off go test ./...
GOWORK=off go run ./cmd/notification-service --check
GOWORK=off go run ./cmd/notification-service --serve --addr :50065
```

At runtime it reuses the shared Smart Recruit MySQL schema, registers `NotificationService`, exposes gRPC health, starts metrics and tracing, registers the `notification` instance through Nacos discovery when configured, and starts NotificationRuntime components for persistence, realtime cache publication, outbox dispatch, inbox-backed notification consumption, and email coordination.
