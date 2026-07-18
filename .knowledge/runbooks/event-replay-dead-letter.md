---
schema_version: 1
id: event-replay-dead-letter
title: Event replay and dead-letter runbook
kind: runbook
status: active
owners:
  - engineering-platform
tags:
  - events
  - outbox
  - inbox
  - replay
applies_to:
  - smart-recruit-commons/internal/platform/events/**
  - smart-recruit-commons/mq/**
  - smart-recruit-worker-service/**
  - smart-recruit-notification-service/**
  - smart-recruit-analytics-service/**
source_refs:
  - smart-recruit-commons/internal/platform/events/envelope.go
  - smart-recruit-commons/mq/consumer.go
  - smart-recruit-commons/mq/publisher.go
  - smart-recruit-commons/migrations/000051_standardize_event_outbox.sql
  - smart-recruit-commons/migrations/000052_add_event_inbox.sql
  - smart-recruit-worker-service/internal/runtime/runtime.go
  - smart-recruit-worker-service/internal/runtime/workload_profile.go
  - smart-recruit-notification-service/internal/domain/repository/notification.go
  - smart-recruit-analytics-service/internal/infrastructure/projection/gorm_store.go
last_verified: 2026-07-14
review_after: 2026-10-14
---

# Event Replay and Dead-Letter Runbook

Before replay, verify event identity, aggregate identity, producer, idempotency key, correlation/causation IDs, retry count, next retry time, failure reason, and consumer checkpoint state. Do not bypass idempotency or replay safeguards without scoped evidence and rollback.

## Verification

Verified against current repository files on 2026-07-14.
