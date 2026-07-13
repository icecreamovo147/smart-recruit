# Notification Service Contract Inventory

## 1. Purpose

本文档记录 `.spec/microservice-ddd-evolution` 中 Notification 服务进入 DDD 迁移前的契约基线。

TASK-009 只创建本地 DDD 骨架并盘点当前通知、邮件、Outbox、Inbox、SSE/realtime 契约；不改变 protobuf、runtime wiring、数据库 schema、RabbitMQ queue、Redis channel、邮件发送行为或用户可见 API。

## 2. Current Local DDD Layers

TASK-009 在 `smart-recruit-notification-service/internal/` 下建立目标 DDD 包边界：

```text
internal/
  domain/
    model/
    repository/
    service/
    event/
  application/
    command/
    query/
    port/
    service/
  infrastructure/
    persistence/
    mq/
    cache/
    email/
    client/
  interfaces/
    grpc/
    mapper/
    event/
  runtime/
```

TASK-011 将 active runtime 收敛到 service-local application、
infrastructure、interfaces 和 runtime adapters。TASK-029 进一步移除了对
`smart-recruit-domain-go/repository` 与 `smart-recruit-domain-go/service` 的
active import。

后续任务边界：

- TASK-010：迁移 Notification domain/application 规则和用例编排。
- TASK-011：迁移 persistence/mq/cache/email/interfaces/runtime adapters，并收敛 active runtime。

## 3. Protobuf, Gateway, and Runtime Contract

当前 protobuf service：

- `recruitment.NotificationService`
- `ListNotifications`
- `UnreadNotificationCount`
- `NotificationSummary`
- `MarkNotificationRead`
- `MarkAllNotificationsRead`

当前 runtime registration：

- `smart-recruit-notification-service/internal/runtime.Runtime.RegisterGRPC`
- `pb.RegisterNotificationServiceServer`
- Service descriptor: `pb.NotificationService_ServiceDesc.ServiceName`

当前 Gateway HTTP/SSE routes：

- `GET /api/v1/candidate/notifications`
- `GET /api/v1/candidate/notifications/unread-count`
- `GET /api/v1/candidate/notifications/summary`
- `GET /api/v1/candidate/notifications/stream`
- `PATCH /api/v1/candidate/notifications/:notification_id/read`
- `PATCH /api/v1/candidate/notifications/read-all`
- `GET /api/v1/hr/notifications`
- `GET /api/v1/hr/notifications/unread-count`
- `GET /api/v1/hr/notifications/summary`
- `GET /api/v1/hr/notifications/stream`
- `PATCH /api/v1/hr/notifications/:notification_id/read`
- `PATCH /api/v1/hr/notifications/read-all`

Compatibility rule:

- TASK-009 不改变 protobuf request/response、rpc 名称、HTTP route、Gateway route mode、SSE event framing 或错误响应结构。
- SSE event 仍由 Gateway 订阅 Redis channel 并输出 `event: notification`。

## 4. Current Compatibility Dependency Baseline

当前 `cmd/notification-service/main.go` 直接依赖：

- `smart-recruit-domain-go/email`
- `smart-recruit-domain-go/mq`
- `smart-recruit-domain-go/pkg/cache`
- `smart-recruit-platform-go/*`
- `smart-recruit-proto/recruitment/pb`

当前 service-local runtime construction：

- `buildNotificationRuntime` 构造本地 `notificationruntime.Runtime`。
- `appservice.New` 构造本地 notification application service。
- `notificationmq.NewOutboxPublisher` 负责 event_outbox claim/publish/retry。
- `notificationmq.NewNotificationConsumer` 与 `NewEmailConsumer` 负责 Inbox
  幂等消费。
  - `NotificationConsumer`
  - `EmailConsumer`

当前 shared repositories/adapters：

- `repository.NewUserRepo`
- `repository.NewNotificationRepo`
- `repository.NewOutboxRepo`
- `repository.NewInboxRepo`
- `repository.NewEmailLogRepo`
- `repository.NewAuthzRepo`
- `cache.NewNotificationCacheWithOptions`
- `mq.New`
- `email.NewRenderer`
- `email.NewSender`

Migration implication:

- TASK-009 只记录这些依赖。后续 TASK 必须逐步把 Notification-owned business behavior 从 shared `service/model/repository` 迁入本服务本地层。
- 真实 SMTP sender 初始化仍在 runtime 中；本 TASK 不新增任何邮件发送路径。

## 5. Behavior Dependency Map

### List notifications

- Protobuf: `ListNotificationsRequest` / `ListNotificationsResponse`
- Legacy method: `service.NotificationService.ListNotifications`
- Key dependencies: actor metadata match, `NotificationRepo.ListCursor`, `NotificationRepo.List`, page/page_size normalization, cursor pagination, and fallback-on-query-error behavior.
- Current compatibility details:
  - `user_id == 0` returns bad request.
  - empty `account_type` returns bad request.
  - page size outside `1..50` defaults to `20`.
  - cursor mode is used when `cursor != ""` or `page <= 0`.
  - DB list errors are logged and return empty list responses instead of hard gRPC errors.

### Unread count and summary

- Protobuf: `UnreadNotificationCountRequest`, `UnreadNotificationCountResponse`, `NotificationSummaryRequest`, `NotificationSummaryResponse`
- Legacy methods: `UnreadNotificationCount`, `NotificationSummary`
- Key dependencies: Redis unread cache, `NotificationRepo.UnreadCount`, `NotificationRepo.Latest`.
- Current compatibility details:
  - unread cache key: `notif:unread:<receiverID>:<accountType>`
  - unread count query failures return `0`.
  - summary latest query failures return unread count only.

### Mark read and mark all read

- Protobuf: `MarkNotificationReadRequest`, `MarkAllNotificationsReadRequest`, `CommonResponse`
- Legacy methods: `MarkNotificationRead`, `MarkAllNotificationsRead`
- Key dependencies: `NotificationRepo.MarkRead`, `NotificationRepo.MarkAllReadBatch`, Redis cache invalidation.
- Current compatibility details:
  - single mark read returns `403/无权限或通知不存在` when no row is affected.
  - mark read DB errors are logged and return `success`.
  - mark all read updates in batches of `1000` and always returns `success`.

### Notification consumer

- Queue: RabbitMQ notification queue, default `recruitment.notification.create`
- Routing key: `notification.create`
- Legacy consumer: `service.NotificationConsumer`
- Inbox consumer name: `notification-consumer`
- Payload fields: `event_id`, `receiver_id`, `receiver_role`, `receiver_account_type`, `type`, `title`, `content`, `link`, `biz_type`, `biz_id`.
- Idempotency:
  - `consumeWithInbox` claims `(consumer_name, event_id)` before handling.
  - `NotificationRepo.CreateOnceWithResult` uses `event_id` and business unique keys to avoid duplicate notifications.
- Realtime side effects:
  - Invalidates unread cache.
  - Recomputes unread count and stores it in Redis.
  - Publishes Redis event to `notif:event:<accountType>:<receiverID>`.

### Email consumer

- Queue: RabbitMQ email queue, default `recruitment.email.send`
- Routing key: `email.send`
- Legacy consumer: `service.EmailConsumer`
- Inbox consumer name: `email-consumer`
- Payload fields: notification fields plus template fields such as `job_title`, `recipient_name`, `interview_date`, `offer_title`, and `expiry_date`.
- Idempotency:
  - `consumeWithInbox` claims `(consumer_name, event_id)`.
  - `EmailLogRepo.ExistsByEventID` prevents duplicate sends.
  - `email_logs.event_id` is unique.
- Current skip behavior:
  - Missing template is skipped without retry.
  - Missing user email is logged as skipped without retry.
  - Duplicate event is logged as skipped without retry.
- Send failures are logged to `email_logs` with status `failed` and returned to RabbitMQ retry/DLQ flow.

### Outbox publisher

- Legacy publisher: `service.OutboxPublisher`
- Source table: `event_outbox`
- Poll interval: `5s`
- Batch size: `50`
- Lock timeout: `2m`
- Publish timeout: `10s`
- Max retry count: `10`
- Backoff: exponential, base `5s`, max `10m`
- Signal channel wakes the poll loop after transactional writes.
- MQ publish success marks outbox row as published; mark-published failure can cause duplicate delivery and requires consumer idempotency.
- MQ publish failure marks retryable failure or dead-letters after retry budget.

## 6. Queue, Retry, and DLQ Contract

RabbitMQ topology is declared by `smart-recruit-domain-go/mq`:

- Exchange: default `recruitment.events`
- Retry exchange: default `recruitment.events.retry`
- DLX exchange: default `recruitment.events.dlx`
- Notification queue: default `recruitment.notification.create`
- Email queue: default `recruitment.email.send`
- Notification retry queue: `<notification queue>.retry`
- Notification DLQ: `<notification queue>.dlq`
- Email retry queue: `<email queue>.retry`
- Email DLQ: `<email queue>.dlq`

Consumer retry flow:

- Handler error publishes to retry exchange while `x-retry-count < MaxRetries`.
- Retry queue TTL returns the message to the main exchange/routing key.
- After retry budget is exhausted, consumer `Nack(false, false)` dead-letters the message.

TASK-009 does not change exchanges, queues, routing keys, prefetch, retry count, retry delay, DLQ behavior, or publisher confirm behavior.

## 7. Table Access Boundary

Source: `smart-recruit-deploy/mysql-table-ownership.json` and current shared repositories.

Notification-owned or notification-runtime-owned tables:

- `notifications`
- `email_logs`

Platform/shared async tables used by Notification runtime:

- `event_outbox`
- `event_inbox`

Transitional read dependency:

- `users` for email receiver lookup.
- `authz` tables for request actor/permission checks through `ServiceAuthorizer`.

Current ownership debt:

- `users` email lookup is a transitional direct read used by `EmailConsumer`; the current table ownership manifest does not declare Notification as a `users` reader. TASK-010/TASK-011 should replace this with an owner-service query/client port or explicitly record the transitional read debt.
- Notification read APIs primarily use gRPC metadata actor verification. `AuthzRepo` is wired through `ServiceAuthorizer`, but the current read path does not perform a separate permission-table lookup in `NotificationService`.

Current write behavior:

- `notifications`: create/read/mark read/mark all read.
- `email_logs`: create send/skipped/failed logs and check duplicate `event_id`.
- `event_outbox`: claim/publish/retry/dead-letter rows.
- `event_inbox`: claim/process/fail/dead-letter consumer records.

No TASK-009 change modifies table ownership, schema, indexes, or migration files.

## 8. Current Test Coverage

Existing Notification service tests:

- `cmd/notification-service/main_test.go`
- `internal/runtime/runtime_test.go`

Current test focus:

- CLI/discovery runtime behavior.
- MQ config maps notification/email queues and retry settings.
- Runtime requires Notification dependency.
- Runtime registers `NotificationService` with gRPC.
- Runtime component inspection requires persistence, realtime delivery, outbox, notification inbox, and email coordination.
- Idempotency semantics are documented in `runtime.IdempotencySemantics`.

Local tests covering Notification behavior:

- `internal/application/service/*_test.go`
- `internal/domain/model/*_test.go`
- `internal/runtime/runtime_test.go`
- `cmd/notification-service/main_test.go`

Current local test gaps:

- No local gRPC mapper/interface tests yet.
- No local consumer replay/idempotency tests yet.

Expected next tests:

- TASK-010 should cover unread/summary/mark read/all read/idempotency in local domain/application.
- TASK-011 should cover persistence/mq/cache/email adapters, gRPC mapping, runtime wiring, and no-real-SMTP behavior.

## 9. Out-of-Scope Confirmation

TASK-009 intentionally does not:

- Move Notification business rules from `smart-recruit-domain-go`.
- Change protobuf or generated Go contracts.
- Change database schema, migrations, or table ownership.
- Change Gateway routing, SSE framing, Redis channel naming, or public HTTP behavior.
- Change RabbitMQ exchanges, queues, routing keys, retry, or DLQ behavior.
- Send real email or modify SMTP configuration.
- Delete or deprecate shared legacy implementation.
- Start using the new skeleton packages from runtime.
