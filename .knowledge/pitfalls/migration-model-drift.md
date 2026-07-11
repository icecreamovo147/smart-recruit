---
schema_version: 1
id: migration-model-drift
title: Migration and model drift pitfall
kind: pitfall
status: active
owners:
  - engineering-platform
tags:
  - database
  - migration
  - model
  - repository
applies_to:
  - logic-grpc-service/migrations/**
  - logic-grpc-service/model/**
  - logic-grpc-service/repository/**
  - db.sql
source_refs:
  - logic-grpc-service/migration/mysql_consistency_test.go
  - logic-grpc-service/migration/runner_test.go
  - logic-grpc-service/model/model.go
  - logic-grpc-service/migrations/000051_standardize_event_outbox.sql
  - logic-grpc-service/migrations/000052_add_event_inbox.sql
  - logic-grpc-service/migrations/000053_add_analytics_projection_events.sql
  - logic-grpc-service/repository/outbox_repo.go
  - logic-grpc-service/repository/inbox_repo.go
  - logic-grpc-service/repository/analytics_projection_repo.go
  - logic-grpc-service/repository/repo_test_helper.go
  - docs/backend-ddd-microservices-evolution-table-ownership-manifest.json
  - docs/backend-ddd-microservices-evolution-schema-separation-plan.md
  - scripts/check-table-ownership.mjs
  - db.sql
last_verified: 2026-07-12
review_after: 2026-10-08
---

# Migration and Model Drift Pitfall

Database changes can pass local unit tests while production migrations, `db.sql`, GORM models, and repository queries drift apart.

## Trigger Conditions

- Adding or renaming a table, column, index, generated column, or constraint.
- Updating `model.go` without a migration.
- Updating a migration without checking repository query assumptions.
- Relying on test-only `AutoMigrate` behavior for a production schema change.
- Changing cursor, pagination, or uniqueness assumptions in repositories.
- Adding or changing a table without updating the table ownership manifest.
- Starting schema or physical database separation without expand-contract, rollback, reconciliation, RTO, and RPO evidence.
- Changing outbox status, retry, retention, or metadata columns without updating publisher writes, repository stats, and `db.sql`.
- Changing inbox status, idempotency, retention, or uniqueness columns without updating consumer wiring, repository claim semantics, and `db.sql`.
- Changing Analytics projection event/checkpoint schema without updating models, `db.sql`, projection repository idempotency, infrastructure adapters, and ingestion tests.

## Risk

The logic service may compile while failing at runtime against a migrated database. Conversely, tests using in-memory or auto-migrated schemas can miss MySQL-specific constraints, generated columns, indexes, or collation behavior.

## Prevention

- Treat migrations, `db.sql`, models, repositories, and tests as one persistence surface.
- Treat table ownership, allowed readers, allowed writers, and transitional shared access as part of the same persistence surface during service extraction.
- Prefer explicit SQL migrations for production schema changes.
- Run migration runner and MySQL consistency tests for schema work.
- Run `node scripts/check-table-ownership.mjs` for schema work that changes `db.sql`.
- Follow `docs/backend-ddd-microservices-evolution-schema-separation-plan.md` before separating schemas or physical databases.
- Keep repository transactions and query filters aligned with service invariants.
- For `event_outbox`, check that terminal timestamps, retention cutoffs, retry/dead-letter transitions, and indexes stay aligned with `OutboxRepo` and `OutboxPublisher`.
- For `event_inbox`, check that the unique consumer/event key, processed/dead duplicate handling, attempt counts, retention cutoffs, and indexes stay aligned with `InboxRepo` and MQ consumer `Start` methods.
- For `analytics_projection_events` and `analytics_projection_checkpoints`, check that the event-id uniqueness, projection cursor upsert, JSON payload/metadata fields, and occurrence/projected timestamps stay aligned with `AnalyticsProjectionRepo` and the Analytics event ingestor.

## Verification

This pitfall was verified from migration consistency tests, migration runner tests, GORM models, outbox/inbox/analytics projection migrations and repository tests, repository test helpers, the table ownership manifest/check script, the schema separation plan, and `db.sql` on 2026-07-12.
