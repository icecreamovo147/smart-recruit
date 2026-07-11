---
schema_version: 1
id: analytics-projection-knowledge-candidate
title: Analytics projection knowledge candidate
kind: candidate
status: draft
owners:
  - analytics-domain
tags:
  - analytics
  - projection
  - domain-events
  - read-model
applies_to:
  - logic-grpc-service/internal/analytics/**
  - logic-grpc-service/repository/analytics_projection_repo.go
  - logic-grpc-service/migrations/000053_add_analytics_projection_events.sql
  - db.sql
proposed_destination: .knowledge/domains/analytics-projections.md
provenance_task: TASK-BDME-024
source_refs:
  - logic-grpc-service/internal/analytics/application/event_ingestor.go
  - logic-grpc-service/internal/analytics/infrastructure/projection_repository.go
  - logic-grpc-service/repository/analytics_projection_repo.go
  - logic-grpc-service/migrations/000053_add_analytics_projection_events.sql
created_at: 2026-07-11
last_verified: 2026-07-11
review_after: 2026-10-09
---

# Analytics Projection Knowledge Candidate

## Candidate Claim

Analytics projection ingestion is now represented by an application-layer ingestor that validates standard domain-event envelopes and writes Analytics-owned projection events and checkpoints through infrastructure/repository adapters.

## Evidence

- `EventProjectionIngestor` stores envelope identity, aggregate identity, producer, idempotency, correlation, payload, metadata, occurrence time, and projection time.
- `analytics_projection_events` is idempotent by `event_id`.
- `analytics_projection_checkpoints` stores a projection cursor keyed by projection name.
- Boundary tests prevent Analytics application code from importing source service packages or introducing `service_read` files.

## Impact

Future Analytics work should promote a formal active document before switching reporting reads or workers to production projection ingestion. Promotion should define projection ownership, replay/backfill, checkpoint semantics, and diagnostic runbooks.
