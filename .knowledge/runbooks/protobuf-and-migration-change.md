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
  - contract
  - database
applies_to:
  - logic-grpc-service/proto/**
  - web-gin-service/proto/**
  - logic-grpc-service/recruitment/pb/**
  - web-gin-service/recruitment/pb/**
  - logic-grpc-service/migrations/**
  - logic-grpc-service/model/**
  - logic-grpc-service/repository/**
source_refs:
  - logic-grpc-service/proto/recruitment.proto
  - logic-grpc-service/recruitment/pb/recruitment.pb.go
  - web-gin-service/recruitment/pb/recruitment.pb.go
  - logic-grpc-service/migration/runner.go
  - logic-grpc-service/migration/mysql_consistency_test.go
  - logic-grpc-service/model/model.go
last_verified: 2026-07-11
review_after: 2026-10-08
---

# Protobuf and Migration Change Runbook

Use this runbook when a TASK changes public contracts or persistence structure.

## Protobuf Path

1. Update the source `.proto` contract first.
2. Regenerate Go code for both logic and gateway service trees.
3. Update logic service implementations and gateway handlers/clients together.
4. Update frontend API clients and types only after the server contract is known.
5. Run targeted Go tests in both services.

## Migration Path

1. Add a new ordered migration pair under `logic-grpc-service/migrations/`.
2. Update GORM models only when the runtime model needs the changed columns or tables.
3. Update repositories and services that own the new persistence behavior.
4. Keep `db.sql` aligned when it serves as current schema reference.
5. Run migration runner tests and MySQL consistency checks when available.
6. For transactional outbox changes, also run outbox repository and publisher tests to verify retry/dead-letter, retention, and payload compatibility.

## Review Questions

- Is this a public contract change, a database schema change, or both?
- Are generated protobuf files synchronized in both service trees?
- Are model fields, repository queries, indexes, and constraints aligned with SQL?
- Does the gateway need new body limits, timeouts, route permissions, or frontend types?
- For outbox changes, do existing consumers still accept the published payload shape?

## Safety

Do not treat generated code edits, migration files, or schema changes as incidental documentation work. They require explicit TASK scope.
