# Backend DDD Microservices Evolution Notification Runtime Extraction

Last verified: 2026-07-11

TASK-BDME-028 extracts the current Notification runtime composition while preserving existing monolith behavior and avoiding production traffic cutover.

## Runtime Boundary

`logic-grpc-service/service/notification_runtime.go` now owns the composition of:

- `NotificationService` for notification persistence, unread counts, mark-read behavior, and realtime event publication through the notification cache;
- `NotificationWorkerPool` for asynchronous notification writes;
- `OutboxPublisher` for durable event dispatch used by notification/email coordination and existing domain workflows;
- `NotificationConsumer` for `notification.create` event consumption through Inbox idempotency;
- `EmailConsumer` for `email.send` event consumption and email log recording through Inbox idempotency.

`service.NewServices` creates this runtime and exposes the same fields as before for compatibility. `logic-grpc-service/main.go` starts the Notification runtime before the remaining non-notification consumers in the existing background-worker block.

## Compatibility Guarantees

This TASK does not change:

- protobuf or HTTP contracts;
- gateway routing;
- table schemas or migrations;
- Dockerfiles or Kubernetes routed deployments;
- RabbitMQ queue names, routing keys, or message payload shapes;
- frontend behavior.

Normal monolith worker startup remains behaviorally equivalent:

1. Start Notification runtime outbox dispatcher.
2. Start Notification consumer.
3. Start Email consumer.
4. Start existing resume parse, embedding, and agent-run consumers.
5. Keep RabbitMQ keepalive in the existing worker block.

## Verification

Run:

```bash
cd logic-grpc-service
go test ./service -run 'TestNewNotificationRuntime|TestNotificationRuntime'
go test ./...
```

The focused tests verify that the runtime wires Notification service, worker, outbox publisher, notification consumer, and email consumer, and that nil runtime/MQ failure paths are explicit.

## Future Work

- A later Notification deployment TASK can move this runtime behind `cmd/notification-service` startup.
- A gateway cutover TASK must still own any route changes and rollback evidence.
- A worker cutover TASK must still own consumer split, readiness probes, deployment manifests, and traffic/queue ownership changes.

