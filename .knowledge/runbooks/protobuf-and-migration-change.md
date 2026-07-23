---
schema_version: 1
id: protobuf-and-migration-change
title: Protobuf and migration change runbook
kind: runbook
status: active
owners:
  - engineering-platform
tags:
  - protobuf
  - migration
  - database
applies_to:
  - smart-recruit-proto/**
  - scripts/check-proto-sync.mjs
  - .github/workflows/ci.yml
  - smart-recruit-commons/migration/**
  - smart-recruit-commons/migrations/**
  - smart-recruit-*-service/internal/infrastructure/persistence/**
  - db.sql
source_refs:
  - smart-recruit-proto/proto/recruitment.proto
  - smart-recruit-proto/recruitment/pb/recruitment.pb.go
  - smart-recruit-proto/scripts/tool-versions.env
  - smart-recruit-proto/scripts/bootstrap-tools.sh
  - smart-recruit-proto/scripts/generate-go.sh
  - scripts/check-proto-sync.mjs
  - .github/workflows/ci.yml
  - smart-recruit-commons/migration/runner.go
  - smart-recruit-commons/migration/schema_compare.go
  - smart-recruit-commons/migration/mysql_consistency_test.go
  - smart-recruit-commons/migrations/000089_schema_baseline.sql
  - smart-recruit-commons/migrations/baseline-lock.json
  - smart-recruit-commons/migrations/README.md
  - scripts/check-migration-baseline.mjs
  - smart-recruit-commons/migrations/archive/pre-baseline-000089/000053_add_analytics_projection_events.sql
  - smart-recruit-commons/migrations/archive/pre-baseline-000089/000084_candidate_profile_screening_fields.sql
  - smart-recruit-commons/migrations/archive/pre-baseline-000089/000085_candidate_profile_structured_history.sql
  - smart-recruit-commons/migrations/archive/pre-baseline-000089/000086_extend_profile_city_length.sql
  - smart-recruit-deploy/mysql-table-ownership.json
  - db.sql
last_verified: 2026-07-23
review_after: 2026-10-21
---

# Protobuf and Migration Change Runbook

For proto changes, edit `smart-recruit-proto/proto/recruitment.proto`, run `smart-recruit-proto/scripts/bootstrap-tools.sh`, regenerate with `smart-recruit-proto/scripts/generate-go.sh`, update gateway/service usages, run `node scripts/check-proto-sync.mjs --check`, run Proto module tests, and confirm a second generation produces no Git diff. Never regenerate with an arbitrary `protoc` from `PATH`; the generator intentionally fails on version mismatch. If a new method is internal-only, consider defining a separate service so existing public-facing generated client interfaces and unrelated handler fakes do not need to change.

For schema changes, start at migration `000090`, add a Commons up/down pair,
update `db.sql`, update service persistence adapters, review table ownership,
run `node scripts/check-migration-baseline.mjs`, and run MySQL migration
consistency tests. Do not edit `000089_schema_baseline.sql` or archived
`000001`–`000088` files. A database with the full historical migration records
must use `--adopt-baseline 89`; a matching `db.sql` import with no history uses
`--baseline 89`.

## Verification

Verified against current pinned protobuf tooling, generated contracts, CI regeneration checks, the v89 baseline/adoption runner, ownership manifest, and baseline schema on 2026-07-23.
