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
applies_to:
  - smart-recruit-commons/migration/**
  - smart-recruit-commons/migrations/**
  - smart-recruit-*-service/internal/infrastructure/persistence/**
  - db.sql
source_refs:
  - smart-recruit-commons/migration/runner_test.go
  - smart-recruit-commons/migration/mysql_consistency_test.go
  - smart-recruit-commons/migrations/000051_standardize_event_outbox.sql
  - smart-recruit-commons/migrations/000052_add_event_inbox.sql
  - smart-recruit-commons/migrations/000053_add_analytics_projection_events.sql
  - smart-recruit-notification-service/internal/infrastructure/persistence/notification_repository.go
  - smart-recruit-analytics-service/internal/infrastructure/projection/gorm_store.go
  - smart-recruit-deploy/mysql-table-ownership.json
  - db.sql
last_verified: 2026-07-14
review_after: 2026-10-14
---

# Migration and Model Drift Pitfall

Drift happens when migrations, `db.sql`, service persistence models, repositories, and table ownership diverge. Test-only migration behavior is not production schema evidence. Update migrations, baseline schema, adapters, ownership, and tests together.

## Verification

Verified against current repository files on 2026-07-14.
