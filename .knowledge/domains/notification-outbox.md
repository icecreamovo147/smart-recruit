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
  - events
  - email
applies_to:
  - smart-recruit-notification-service/**
  - smart-recruit-worker-service/**
  - smart-recruit-commons/internal/platform/events/**
  - smart-recruit-commons/mq/**
  - smart-recruit-commons/migrations/00005*.sql
  - smart-recruit-gateway/handler/notification.go
source_refs:
  - smart-recruit-notification-service/internal/application/service/notification_service.go
  - smart-recruit-notification-service/internal/infrastructure/persistence/notification_repository.go
  - smart-recruit-notification-service/internal/interfaces/grpc/notification_server.go
  - smart-recruit-notification-service/internal/runtime/runtime.go
  - smart-recruit-worker-service/internal/runtime/runtime.go
  - smart-recruit-worker-service/internal/runtime/workload_profile.go
  - smart-recruit-commons/internal/platform/events/envelope.go
  - smart-recruit-commons/mq/publisher.go
  - smart-recruit-commons/migrations/000051_standardize_event_outbox.sql
  - smart-recruit-commons/migrations/000052_add_event_inbox.sql
  - smart-recruit-gateway/handler/notification.go
last_verified: 2026-07-14
review_after: 2026-10-14
---

# Notification and Outbox Domain

Notification APIs are served by Notification service. Shared outbox/inbox schema, event envelope, and MQ support live in Commons and Worker service profiles. Gateway notification endpoints expose list, unread count, summary, mark-read, mark-all-read, and stream behavior through generated clients.

## Verification

Verified against current repository files on 2026-07-14.
