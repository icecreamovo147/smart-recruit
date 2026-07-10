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
  - logic-grpc-service/repository/notification_repo.go
  - logic-grpc-service/model/model.go
  - web-gin-service/handler/notification.go
source_refs:
  - logic-grpc-service/service/application_service.go
  - logic-grpc-service/service/offer_service.go
  - logic-grpc-service/service/notification_service.go
  - logic-grpc-service/service/notification_worker.go
  - logic-grpc-service/service/outbox_publisher.go
  - web-gin-service/handler/notification.go
  - logic-grpc-service/model/model.go
last_verified: 2026-07-10
review_after: 2026-10-08
---

# Notification and Outbox Domain

Notifications are produced by recruitment workflows and delivered through database records, cache invalidation, Redis/SSE events, and email outbox events where applicable. Notification behavior is part of recruitment correctness because users rely on it to see application, interview, offer, and collaboration events.

## Main Responsibilities

- Recruitment services write notification and email intents through `OutboxPublisher` inside workflow transactions.
- `NotificationService` reads, summarizes, marks, and publishes notification-created events with account-type scoping.
- `NotificationWorkerPool` throttles asynchronous notification writes and publishes created events through the cache layer.
- `web-gin-service/handler/notification.go` exposes list, unread count, summary, mark-read, mark-all-read, and stream endpoints.
- Frontend notification components subscribe to stream endpoints and display unread counts.

## Impact Guidance

- Workflow changes that create, suppress, or reorder notifications should review outbox payloads and user-facing notification surfaces.
- Account type matters. Candidate, staff, and interviewer notifications should not share cache keys or cookie assumptions.
- SSE changes should check gateway stream handling and frontend event parsing.
- Email outbox changes should be reviewed with notification changes because some workflows emit both.

## Verification

Verified against application and offer services, notification service, notification worker, outbox publisher, notification handler, and model definitions on 2026-07-10.
