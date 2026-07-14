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
  - smart-recruit-commons/migration/**
  - smart-recruit-commons/migrations/**
  - smart-recruit-*-service/internal/infrastructure/persistence/**
  - db.sql
source_refs:
  - smart-recruit-proto/proto/recruitment.proto
  - smart-recruit-proto/recruitment/pb/recruitment.pb.go
  - smart-recruit-proto/scripts/generate-go.sh
  - smart-recruit-commons/migration/runner.go
  - smart-recruit-commons/migration/mysql_consistency_test.go
  - smart-recruit-commons/migrations/000053_add_analytics_projection_events.sql
  - smart-recruit-deploy/mysql-table-ownership.json
  - db.sql
last_verified: 2026-07-14
review_after: 2026-10-14
---

# Protobuf and Migration Change Runbook

For proto changes, edit `smart-recruit-proto/proto/recruitment.proto`, regenerate Go contracts, update gateway/service usages, and run proto contract tests. If a new method is internal-only, consider defining a separate service so existing public-facing generated client interfaces and unrelated handler fakes do not need to change. For schema changes, add Commons migration pairs, update `db.sql`, update service persistence adapters, review table ownership, and run migration consistency tests.

## Verification

Verified against current repository files on 2026-07-14.
