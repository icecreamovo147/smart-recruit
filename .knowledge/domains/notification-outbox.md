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
  - smart-recruit-interview-service/internal/infrastructure/mq/**
  - smart-recruit-offer-service/internal/infrastructure/mq/**
  - smart-recruit-recruitment-service/internal/infrastructure/persistence/**
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/embedding_outbox.go
  - smart-recruit-ai-agent-service/internal/infrastructure/embeddingqueue/**
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
  - smart-recruit-commons/migrations/archive/pre-baseline-000089/000051_standardize_event_outbox.sql
  - smart-recruit-commons/migrations/archive/pre-baseline-000089/000052_add_event_inbox.sql
  - smart-recruit-interview-service/internal/infrastructure/mq/outbox_publisher.go
  - smart-recruit-offer-service/internal/infrastructure/mq/outbox_publisher.go
  - smart-recruit-recruitment-service/internal/infrastructure/persistence/native_adapters.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/embedding_outbox.go
  - smart-recruit-ai-agent-service/internal/infrastructure/embeddingqueue/consumer.go
  - smart-recruit-ai-agent-service/internal/infrastructure/embeddingqueue/store.go
  - smart-recruit-worker-service/internal/outbox/dispatcher.go
  - smart-recruit-commons/mq/consumer.go
  - smart-recruit-commons/mq/rabbitmq.go
  - smart-recruit-deploy/mysql-table-ownership.json
  - smart-recruit-gateway/handler/notification.go
last_verified: 2026-07-30
review_after: 2026-10-14
---

# Notification and Outbox Domain

Notification APIs are served by Notification service. Shared outbox/inbox schema, event envelope, and MQ support live in Commons and Worker service profiles. Gateway notification endpoints expose list, unread count, summary, mark-read, mark-all-read, and stream behavior through generated clients.

Recruitment, Interview, and Offer write their domain events through local GORM outbox adapters using the shared `event_outbox` table shape instead of copied legacy outbox repositories. Recruitment's active outbox writes live in `smart-recruit-recruitment-service/internal/infrastructure/persistence/native_adapters.go`; Interview and Offer keep dedicated `internal/infrastructure/mq` adapters.

AI Agent is also a shared outbox producer and inbox consumer. Creating an Agent Skill Package or immutable version writes an `embedding.upsert` event alongside the version transaction. The Worker claims pending outbox rows, publishes by routing key, retries bounded failures through `next_retry_at`, and records exhausted events as dead. The AI Agent consumes the embedding queue as `ai-agent-embedding-consumer`, claims `(consumer_name, event_id)` in `event_inbox`, and generates the version plus section embeddings. Processed or dead inbox rows are not reclaimed; failed/processing rows can be reclaimed on broker redelivery. RabbitMQ retries carry a bounded retry header and eventually route the original message to the embedding queue DLQ.

Database inbox status and broker DLQ state are separate evidence. A failed inbox row records the last consumer error, while the shared MQ layer decides retry delay/count and dead-letter routing. Preserve the original event ID and idempotency key during investigation; do not bypass the claim state or manufacture an unrelated replacement event.

## Verification

Verified against the shared event schema, Worker outbox dispatcher, RabbitMQ retry/DLQ topology, AI Agent transactional embedding outbox and inbox consumer, and table-ownership registry on 2026-07-30.
