---
schema_version: 1
id: notification-outbox
title: Notification and outbox domain
kind: domain
status: active
owners:
  - recruitment-domain
tags:
  - notification
  - outbox
  - sse
  - email
applies_to:
  - logic-grpc-service/service/notification_service.go
  - logic-grpc-service/service/notification_worker.go
  - logic-grpc-service/service/outbox_publisher.go
  - logic-grpc-service/service/inbox_consumer.go
  - logic-grpc-service/internal/platform/events/envelope.go
  - logic-grpc-service/repository/outbox_repo.go
  - logic-grpc-service/repository/inbox_repo.go
  - logic-grpc-service/repository/notification_repo.go
  - logic-grpc-service/model/model.go
  - web-gin-service/handler/notification.go
source_refs:
  - logic-grpc-service/service/application_service.go
  - logic-grpc-service/service/offer_service.go
  - logic-grpc-service/service/notification_service.go
  - logic-grpc-service/service/notification_worker.go
  - logic-grpc-service/service/outbox_publisher.go
  - logic-grpc-service/service/inbox_consumer.go
  - logic-grpc-service/internal/platform/events/envelope.go
  - logic-grpc-service/repository/outbox_repo.go
  - logic-grpc-service/repository/inbox_repo.go
  - logic-grpc-service/migrations/000051_standardize_event_outbox.sql
  - logic-grpc-service/migrations/000052_add_event_inbox.sql
  - web-gin-service/handler/notification.go
  - logic-grpc-service/model/model.go
last_verified: 2026-07-11
review_after: 2026-10-08
---

# Notification and Outbox Domain

Notifications are produced by recruitment workflows and delivered through database records, cache invalidation, Redis/SSE events, and email outbox events where applicable. Notification behavior is part of recruitment correctness because users rely on it to see application, interview, offer, and collaboration events.

## Main Responsibilities

- Recruitment services write notification and email intents through `OutboxPublisher` inside workflow transactions.
- `logic-grpc-service/internal/platform/events/` defines the target domain-event envelope fields used to standardize event identity, aggregate identity, producer, actor context, idempotency keys, correlation/causation IDs, payload, and diagnostics metadata across Outbox, future Inbox records, consumers, and projections.
- `event_outbox` stores envelope metadata, retry diagnostics, publish/dead-letter timestamps, and retention-ready terminal state. Published events default to 30-day retention and dead-letter events to 90-day retention through repository retention helpers.
- `event_inbox` records consumer-side event claims, attempts, processed/failed/dead statuses, and idempotency keys per consumer. Processed records default to 30-day retention and dead-letter records to 90-day retention through repository retention helpers.
- `OutboxPublisher` keeps legacy top-level payload fields for existing consumers while also writing the standard envelope fields and nested `payload` object.
- MQ consumers call the shared Inbox helper from their `Start` entrypoints. Direct unit tests that invoke `handle` bypass the helper intentionally and test only business handling.
- Publish failures retry with bounded exponential backoff; after the retry budget is exhausted, events are marked dead-letter with `dead_lettered_at`.
- `NotificationService` reads, summarizes, marks, and publishes notification-created events with account-type scoping.
- `NotificationWorkerPool` throttles asynchronous notification writes and publishes created events through the cache layer.
- `web-gin-service/handler/notification.go` exposes list, unread count, summary, mark-read, mark-all-read, and stream endpoints.
- Frontend notification components subscribe to stream endpoints and display unread counts.

## Impact Guidance

- Workflow changes that create, suppress, or reorder notifications should review outbox payloads and user-facing notification surfaces.
- Account type matters. Candidate, staff, and interviewer notifications should not share cache keys or cookie assumptions.
- SSE changes should check gateway stream handling and frontend event parsing.
- Email outbox changes should be reviewed with notification changes because some workflows emit both.
- Outbox or Inbox schema changes should keep migrations, `db.sql`, GORM model fields, repository stats/retention helpers, consumer idempotency, and publisher payload compatibility aligned.

## Verification

Verified against application and offer services, notification service, notification worker, outbox publisher, shared Inbox consumer helper, outbox/inbox repositories, domain-event envelope contract, notification handler, migrations, `db.sql`, and model definitions on 2026-07-11.
