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
  - smart-recruit-commons/migrations/000070_add_platform_ai_control_plane.sql
  - smart-recruit-commons/migrations/000071_add_structured_ai_release_trace.sql
  - smart-recruit-commons/migrations/000078_repair_hr_capability_prompt_releases.sql
  - smart-recruit-commons/migrations/000079_add_ai_billing_settlement_outbox.sql
  - smart-recruit-commons/migrations/000082_standardize_utc8_time_semantics.sql
  - smart-recruit-commons/migrations/000084_candidate_profile_screening_fields.sql
  - smart-recruit-commons/migrations/000085_candidate_profile_structured_history.sql
  - smart-recruit-commons/migrations/000086_extend_profile_city_length.sql
  - smart-recruit-commons/cmd/time-preflight/main.go
  - smart-recruit-platform-go/mysqltime/mysql.go
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
last_verified: 2026-07-23
review_after: 2026-10-21
---

# Persistence and Migration Architecture

Shared SQL migrations and the migration runner live in `smart-recruit-commons/`. Bounded services own repository ports and persistence adapters for their contexts. The current single-MySQL ownership model is documented in `smart-recruit-deploy/mysql-table-ownership.json`.

Schema changes must keep migrations, `db.sql`, service persistence code, table ownership, and focused tests aligned.

The pre-launch platform AI cutover is migration `000070`; it promotes only the default tenant's technical AI configuration after a fail-closed conflict check. Migration `000071` adds fixed capability-release/model trace columns to structured resume-parse and candidate-match evidence. Both must remain reversible independently and match the cold-start schema.

Migration `000078` repairs HR Agent-backed capability releases whose initial immutable snapshot omitted the Agent-bound Prompt. It appends corrected published versions, advances capability and entitlement pointers, preserves the original release rows for audit, and keeps `db.sql` cold-start data aligned without rewriting the historical `000070` migration.

Migration `000079` adds AI Agent's durable Billing settlement outbox. Its reservation number is unique and references Billing's reservation, while ownership remains with AI Agent. Billing's only declared access is read-only maintenance protection for unresolved settlement, cancellation, and dead-letter states. The migration and `db.sql` must retain matching status checks and indexes.

Migration `000082` establishes the platform UTC+8 persistence contract without rewriting historical migrations. It records each provable UTC-wall-clock conversion in `utc8_time_conversion_audit`, converts only migration-owned publications, 000081 immediate Billing publications, compatibility tenant subscriptions, and 000076 reconciliation fields, and restores exact old values on rollback. `cmd/time-preflight` is the required read-only deployment gate: it inventories `DATETIME`/`TIMESTAMP` columns and fails on ambiguous publications, invalid lifecycle ordering, future open orders, or negative payment age. Runtime MySQL DSNs are normalized to `parseTime=true`, `loc=Asia%2FShanghai`, and per-connection `time_zone='+08:00'`; startup rejects a mismatched session.

Candidate profile schema growth is owned by Recruitment and must stay synchronized across migrations, `db.sql`, ownership, and persistence adapters. Migration `000084` adds screening fields on `candidate_profiles` (`city`, years of experience, job status, expected position/salary, available-from, and summary). Migration `000085` introduces candidate-owned `candidate_educations` and `candidate_experiences` tables. Migration `000086` widens `candidate_profiles.city` to `VARCHAR(128)` for province/city/district paths; `db.sql` and ownership must reflect the final `city` width plus the two structured-history tables.

LLM model metadata uses `llm_model_catalog` for reviewed reusable facts, `llm_model_metadata_observations` for deduplicated field-level evidence, and `llm_models` for the user-confirmed runtime snapshot. Changing catalog data belongs in the versioned catalog import rather than migration seed SQL; migrations define only the durable schema.

GORM table records that are needed by a bounded service should stay private to that service's infrastructure adapter. Recruitment's active runtime uses a local native persistence bundle for job, taxonomy, candidate/resume, application, invite-code, usage-audit, and `event_outbox` records under `smart-recruit-recruitment-service/internal/infrastructure/persistence/`. Interview's active persistence keeps `interview_schedules`, `interview_feedbacks`, and local `event_outbox` records under `smart-recruit-interview-service/internal/infrastructure/**`; Offer's active persistence keeps `offers`, `offer_events`, and local `event_outbox` records under `smart-recruit-offer-service/internal/infrastructure/**`. Domain packages continue to use repository and publisher ports rather than GORM models.

## Verification

Verified against current repository files on 2026-07-23, including migrations `000084`-`000086`, `db.sql` candidate profile tables, and Recruitment ownership entries.
