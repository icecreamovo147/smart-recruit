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
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/embedding_outbox.go
  - smart-recruit-ai-agent-service/internal/infrastructure/embeddingqueue/**
source_refs:
  - smart-recruit-commons/internal/platform/events/envelope.go
  - smart-recruit-commons/mq/consumer.go
  - smart-recruit-commons/mq/publisher.go
  - smart-recruit-commons/migrations/archive/pre-baseline-000089/000051_standardize_event_outbox.sql
  - smart-recruit-commons/migrations/archive/pre-baseline-000089/000052_add_event_inbox.sql
  - smart-recruit-worker-service/internal/runtime/runtime.go
  - smart-recruit-worker-service/internal/runtime/workload_profile.go
  - smart-recruit-notification-service/internal/domain/repository/notification.go
  - smart-recruit-analytics-service/internal/infrastructure/projection/gorm_store.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/embedding_outbox.go
  - smart-recruit-ai-agent-service/internal/infrastructure/embeddingqueue/consumer.go
  - smart-recruit-ai-agent-service/internal/infrastructure/embeddingqueue/store.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/platform_ai_embedding_readiness.go
  - smart-recruit-worker-service/internal/outbox/dispatcher.go
  - smart-recruit-commons/mq/rabbitmq.go
last_verified: 2026-07-30
review_after: 2026-10-14
---

# Event Replay and Dead-Letter Runbook

Before replay, verify event identity, aggregate identity, producer, idempotency key, correlation/causation IDs, retry count, next retry time, failure reason, and consumer checkpoint state. Do not bypass idempotency or replay safeguards without scoped evidence and rollback.

For Agent Skill `embedding.upsert`, also verify that the aggregate/object points to the immutable `agent_skill_version`, the version still exists, and its current compiled hash matches the event idempotency key. Separate these stages:

1. `event_outbox` pending/processing/retrying/dead means the Worker has not durably completed broker publication.
2. A published outbox row with no AI Agent inbox row points to queue binding, consumer startup, or delivery failure.
3. An `ai-agent-embedding-consumer` inbox row in failed/processing state points to payload, configuration, provider, persistence, or interrupted-consumer failure. A broker redelivery reclaims those states and increments attempts; processed/dead rows are not reclaimed.
4. RabbitMQ increments the retry header, delays through the queue-specific retry queue, and routes exhausted messages to the embedding queue DLQ. This broker DLQ is not the same as an outbox dead row or inbox failed row.
5. After a governed redelivery succeeds, verify ready embeddings for the version and every section under the release model, correct scope IDs, matching compiled-hash metadata, dimension/vector validity, and publication readiness.

Prefer redriving the original message through the existing outbox or queue repair procedure so its event ID and idempotency key remain stable. Never mark inbox/outbox rows successful by hand, skip the consumer claim, paste payloads into issue trackers, or create a replacement event until an authorized operator has identified why the original identity cannot be reused.

## Verification

Verified against the Worker outbox state machine, Commons RabbitMQ retry/DLQ topology, AI Agent embedding event envelope/inbox consumer, and publication readiness gate on 2026-07-30.
