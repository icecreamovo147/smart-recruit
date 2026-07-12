---
schema_version: 1
id: event-replay-dead-letter
title: Event replay and dead letter runbook
kind: runbook
status: active
owners:
  - engineering-platform
tags:
  - events
  - outbox
  - inbox
  - dead-letter
  - retention
applies_to:
  - .spec/backend-ddd-microservices-evolution/docs/backend-ddd-microservices-evolution-event-replay-dead-letter-runbook.md
  - scripts/event-replay-dead-letter.sh
  - logic-grpc-service/repository/outbox_repo.go
  - logic-grpc-service/repository/inbox_repo.go
  - logic-grpc-service/model/model.go
  - db.sql
source_refs:
  - .spec/backend-ddd-microservices-evolution/docs/backend-ddd-microservices-evolution-event-replay-dead-letter-runbook.md
  - scripts/event-replay-dead-letter.sh
  - logic-grpc-service/repository/outbox_repo.go
  - logic-grpc-service/repository/inbox_repo.go
  - logic-grpc-service/model/model.go
  - db.sql
last_verified: 2026-07-11
review_after: 2026-10-09
---

# Event Replay And Dead Letter Runbook

Use this runbook when a TASK or incident touches `event_outbox`, `event_inbox`, event replay, dead-letter inspection, consumer repair, or retention cleanup. The detailed operator checklist lives in `.spec/backend-ddd-microservices-evolution/docs/backend-ddd-microservices-evolution-event-replay-dead-letter-runbook.md`; this knowledge document exists to route future Agents to the current source-backed procedure.

## Current Contract

- `event_outbox` stores durable domain events and uses status values `0=pending`, `1=published`, `2=dead`, and `3=processing`.
- `event_inbox` stores consumer idempotency records and uses status values `0=processing`, `1=processed`, `2=failed`, and `3=dead`.
- Dead outbox rows can be prepared for replay by moving them back to pending after the poison-message cause is resolved.
- Dead inbox rows should be moved back to failed, not directly to processing, so the consumer claim path can re-enter through the normal idempotency logic on redelivery.
- Stale processing outbox rows may be unlocked only after confirming the claiming worker is gone or the lock is outside the approved timeout window.
- Retention defaults are 30 days for successful terminal records and 90 days for dead-letter records.

## Safe Tooling

`scripts/event-replay-dead-letter.sh` is intentionally conservative:

- read-only inspection actions can execute `SELECT` statements through the MySQL CLI when the caller provides DB environment variables;
- mutation-related actions print SQL only;
- generated SQL includes pre-update `SELECT ... FOR UPDATE` checks for repair operations;
- retention cleanup is generated as batched SQL.

Do not bypass these guardrails in Agent changes. If a future TASK needs automated write execution, it must explicitly scope the behavior, confirmation model, audit evidence, and rollback plan.

## Review Checklist

1. Confirm affected event ids, aggregate ids, producer, routing key, retry count, and dead-letter error.
2. Confirm downstream consumers are idempotent for the replay target.
3. Confirm the root cause is fixed or explicitly accepted.
4. Generate SQL with the checked-in script and review it before execution.
5. Recheck outbox and inbox status counts after the repair window.
6. Record event ids, generated SQL intent, execution actor, window, and verification results in the incident or TASK report.

## When This Runbook Is Stale

Mark this document stale if outbox or inbox status values change, retention constants change, the repository claim/mark semantics change, or a production-safe automated replay executor replaces the SQL-generation workflow.

