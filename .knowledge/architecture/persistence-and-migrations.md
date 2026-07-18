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
  - smart-recruit-commons/migration/**
  - smart-recruit-commons/migrations/**
  - smart-recruit-*-service/internal/infrastructure/persistence/**
  - smart-recruit-deploy/mysql-table-ownership.json
  - db.sql
source_refs:
  - smart-recruit-commons/migration/runner.go
  - smart-recruit-commons/migration/runner_test.go
  - smart-recruit-commons/migration/mysql_consistency_test.go
  - smart-recruit-commons/migrations/000051_standardize_event_outbox.sql
  - smart-recruit-commons/migrations/000052_add_event_inbox.sql
  - smart-recruit-commons/migrations/000053_add_analytics_projection_events.sql
  - smart-recruit-commons/migrations/000060_add_llm_model_catalog.sql
  - smart-recruit-deploy/mysql-table-ownership.json
  - db.sql
  - smart-recruit-notification-service/internal/infrastructure/persistence/notification_repository.go
  - smart-recruit-interview-service/internal/infrastructure/persistence/interview_repository.go
  - smart-recruit-interview-service/internal/infrastructure/mq/outbox_publisher.go
  - smart-recruit-offer-service/internal/infrastructure/persistence/offer_repository.go
  - smart-recruit-offer-service/internal/infrastructure/mq/outbox_publisher.go
  - smart-recruit-recruitment-service/internal/infrastructure/persistence/native_adapters.go
  - smart-recruit-analytics-service/internal/infrastructure/projection/gorm_store.go
  - smart-recruit-recruitment-service/internal/domain/repository/recruitment.go
last_verified: 2026-07-19
review_after: 2026-10-14
---

# Persistence and Migration Architecture

Shared SQL migrations and the migration runner live in `smart-recruit-commons/`. Bounded services own repository ports and persistence adapters for their contexts. The current single-MySQL ownership model is documented in `smart-recruit-deploy/mysql-table-ownership.json`.

Schema changes must keep migrations, `db.sql`, service persistence code, table ownership, and focused tests aligned.

LLM model metadata uses `llm_model_catalog` for reviewed reusable facts, `llm_model_metadata_observations` for deduplicated field-level evidence, and `llm_models` for the user-confirmed runtime snapshot. Changing catalog data belongs in the versioned catalog import rather than migration seed SQL; migrations define only the durable schema.

GORM table records that are needed by a bounded service should stay private to that service's infrastructure adapter. Recruitment's active runtime uses a local native persistence bundle for job, taxonomy, candidate/resume, application, invite-code, usage-audit, and `event_outbox` records under `smart-recruit-recruitment-service/internal/infrastructure/persistence/`. Interview's active persistence keeps `interview_schedules`, `interview_feedbacks`, and local `event_outbox` records under `smart-recruit-interview-service/internal/infrastructure/**`; Offer's active persistence keeps `offers`, `offer_events`, and local `event_outbox` records under `smart-recruit-offer-service/internal/infrastructure/**`. Domain packages continue to use repository and publisher ports rather than GORM models.

## Verification

Verified against current repository files on 2026-07-19.
