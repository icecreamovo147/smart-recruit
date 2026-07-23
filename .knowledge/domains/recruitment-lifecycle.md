---
schema_version: 1
id: recruitment-lifecycle
title: Recruitment lifecycle and status transitions
kind: domain
status: active
owners:
  - recruitment-domain
tags:
  - recruitment
  - lifecycle
  - status
  - notification
applies_to:
  - smart-recruit-recruitment-service/**
  - smart-recruit-interview-service/**
  - smart-recruit-offer-service/**
  - smart-recruit-notification-service/**
  - smart-recruit-commons/internal/platform/events/**
source_refs:
  - smart-recruit-recruitment-service/internal/domain/model/recruitment.go
  - smart-recruit-recruitment-service/internal/domain/policy/recruitment.go
  - smart-recruit-recruitment-service/internal/application/service/application_collaboration_taxonomy_service.go
  - smart-recruit-interview-service/internal/domain/service/interview_policy.go
  - smart-recruit-interview-service/internal/application/service/interview_service.go
  - smart-recruit-offer-service/internal/domain/model/status.go
  - smart-recruit-offer-service/internal/application/service/offer_service.go
  - smart-recruit-interview-service/internal/infrastructure/client/application_adapter.go
  - smart-recruit-offer-service/internal/infrastructure/client/application_adapter.go
  - smart-recruit-commons/internal/platform/events/envelope.go
last_verified: 2026-07-14
review_after: 2026-10-14
---

# Recruitment Lifecycle and Status Transitions

Recruitment owns application records and status history. Interview and Offer own local lifecycle policies and coordinate application side effects through adapters and events. Status, notification, timeline, and analytics changes should be reviewed together.

The internal Recruitment owner contract is `ApplicationOwnerService`: `GetApplicationSnapshot` returns the current application, candidate, job, job owner/scope metadata, resume, status key, round, and current-round flag; `ApplyApplicationLifecycleTransition` applies a conditional status change with an expected status key, actor account type, and optional close-current-round flag through the Recruitment application service.

Interview service no longer carries a local application repository in `internal/legacydomain`. Interview lifecycle code reads application state through `internal/infrastructure/client.ApplicationAdapter`, applies application status side effects through `ApplicationLifecycleAdapter`, and keeps Interview-owned schedule and feedback rows in `internal/infrastructure/persistence`.

Offer service no longer carries a local application repository in `internal/legacydomain`. Offer lifecycle code reads application state through `internal/infrastructure/client.ApplicationAdapter`, applies application status side effects through `ApplicationLifecycleAdapter`, and keeps Offer-owned rows in `internal/infrastructure/persistence`.

## Verification

Verified against current repository files on 2026-07-14.
