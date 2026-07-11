---
schema_version: 1
id: persistence-and-migrations
title: Persistence and migration architecture
kind: architecture
status: active
owners:
  - engineering-platform
tags:
  - database
  - migrations
  - model
  - repository
applies_to:
  - logic-grpc-service/migration/**
  - logic-grpc-service/migrations/**
  - logic-grpc-service/model/**
  - logic-grpc-service/repository/**
  - db.sql
source_refs:
  - logic-grpc-service/main.go
  - logic-grpc-service/migration/runner.go
  - logic-grpc-service/migration/runner_test.go
  - logic-grpc-service/migration/mysql_consistency_test.go
  - logic-grpc-service/model/model.go
  - logic-grpc-service/migrations/000051_standardize_event_outbox.sql
  - logic-grpc-service/migrations/000052_add_event_inbox.sql
  - logic-grpc-service/migrations/000053_add_analytics_projection_events.sql
  - logic-grpc-service/repository/outbox_repo.go
  - logic-grpc-service/repository/inbox_repo.go
  - logic-grpc-service/repository/analytics_projection_repo.go
  - logic-grpc-service/repository/application_repo.go
  - docs/backend-ddd-microservices-evolution-table-ownership-manifest.json
  - scripts/check-table-ownership.mjs
  - db.sql
last_verified: 2026-07-12
review_after: 2026-10-08
---

# Persistence and Migration Architecture

The logic service owns persistence. Database structure is represented by SQL migrations, `db.sql`, GORM models, repositories, and service-level transactions. The HTTP gateway should not encode persistence rules.

## Persistence Layers

- `logic-grpc-service/migrations/` contains ordered SQL migration pairs.
- `logic-grpc-service/migration/runner.go` loads, applies, tracks, baselines, and rolls back migrations.
- `logic-grpc-service/main.go` embeds migrations and applies pending migrations before starting service registration.
- `logic-grpc-service/model/` contains GORM models for users, RBAC, recruitment, notification/outbox, AI, resume intelligence, MCP, and configuration tables.
- `logic-grpc-service/repository/` owns database access, transactions, pagination, and query shapes.
- `logic-grpc-service/service/` owns business invariants and orchestrates repository calls.
- Outbox schema changes must align `event_outbox` migrations, `db.sql`, `model.EventOutbox`, `repository.OutboxRepo`, publisher payload compatibility, and tests because the table is used for transactional event delivery and retry diagnostics.
- Inbox schema changes must align `event_inbox` migrations, `db.sql`, `model.EventInbox`, `repository.InboxRepo`, consumer entrypoint wiring, and tests because the table is used for consumer idempotency and duplicate-delivery diagnostics.
- Analytics projection schema changes must align `analytics_projection_events`, `analytics_projection_checkpoints`, `db.sql`, GORM models, `repository.AnalyticsProjectionRepo`, projection infrastructure adapters, and ingestion tests because those tables are the Analytics-owned event-projection read-model input.
- `docs/backend-ddd-microservices-evolution-table-ownership-manifest.json` is the current table ownership manifest for the DDD/microservices evolution. Every table in `db.sql` must have an owner, allowed readers, allowed writers, and explicit transitional shared access when a non-owner writer remains during extraction.

## Impact Guidance

- Schema changes require migration files, model alignment, repository review, and MySQL consistency tests.
- Transactional workflow changes should be made in services and repositories, not handlers.
- Cursor or pagination changes should check affected repository queries and gateway handler parsing.
- `db.sql` must remain aligned with migrations when it represents the current baseline.
- Table additions or ownership changes must update the ownership manifest and pass `node scripts/check-table-ownership.mjs`.
- Test helpers using `AutoMigrate` are not a replacement for production migrations.

## Verification

Verified against migration runner, migration tests, MySQL consistency test, `model.go`, `000051_standardize_event_outbox.sql`, `000052_add_event_inbox.sql`, `000053_add_analytics_projection_events.sql`, representative repositories, the table ownership manifest/check script, and `db.sql` on 2026-07-12.
