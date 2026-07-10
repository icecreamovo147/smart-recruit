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
  - logic-grpc-service/repository/repo_test_helper.go
  - db.sql
last_verified: 2026-07-10
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

## Risk

The logic service may compile while failing at runtime against a migrated database. Conversely, tests using in-memory or auto-migrated schemas can miss MySQL-specific constraints, generated columns, indexes, or collation behavior.

## Prevention

- Treat migrations, `db.sql`, models, repositories, and tests as one persistence surface.
- Prefer explicit SQL migrations for production schema changes.
- Run migration runner and MySQL consistency tests for schema work.
- Keep repository transactions and query filters aligned with service invariants.

## Verification

This pitfall was verified from migration consistency tests, migration runner tests, GORM models, repository test helpers, and `db.sql` on 2026-07-10.
