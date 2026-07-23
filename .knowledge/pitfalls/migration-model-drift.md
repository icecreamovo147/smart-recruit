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
  - smart-recruit-commons/migration/schema_compare.go
  - smart-recruit-commons/migration/mysql_consistency_test.go
  - smart-recruit-commons/migrations/000089_schema_baseline.sql
  - smart-recruit-commons/migrations/baseline-lock.json
  - smart-recruit-commons/migrations/archive/pre-baseline-000089/000051_standardize_event_outbox.sql
  - smart-recruit-commons/migrations/archive/pre-baseline-000089/000052_add_event_inbox.sql
  - smart-recruit-commons/migrations/archive/pre-baseline-000089/000053_add_analytics_projection_events.sql
  - smart-recruit-commons/migrations/archive/pre-baseline-000089/000084_candidate_profile_screening_fields.sql
  - smart-recruit-commons/migrations/archive/pre-baseline-000089/000085_candidate_profile_structured_history.sql
  - smart-recruit-commons/migrations/archive/pre-baseline-000089/000086_extend_profile_city_length.sql
  - smart-recruit-notification-service/internal/infrastructure/persistence/notification_repository.go
  - smart-recruit-analytics-service/internal/infrastructure/projection/gorm_store.go
  - smart-recruit-deploy/mysql-table-ownership.json
  - db.sql
last_verified: 2026-07-23
review_after: 2026-10-21
---

# Migration and Model Drift Pitfall

Drift happens when migrations, `db.sql`, service persistence models, repositories, and table ownership diverge. Test-only migration behavior is not production schema evidence. Update migrations, baseline schema, adapters, ownership, and tests together.

After the v89 squash point, `000089_schema_baseline.sql` and archived
`000001`–`000088` are immutable. New changes belong in `000090+` and `db.sql`.
Do not silence adoption failures by editing checksums: `--adopt-baseline 89`
must compare tables, columns, indexes, foreign keys, and checks before recording
the baseline. Repair a confirmed live-schema drift with reviewed,
data-preserving DDL, then rerun adoption.

## Verification

Verified against the v89 baseline lock, schema comparison gate, archived history, and matching `db.sql` / ownership surfaces on 2026-07-23.
