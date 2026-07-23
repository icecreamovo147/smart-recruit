---
schema_version: 1
id: debug-recruitment-lifecycle
title: Debug recruitment lifecycle issues
kind: runbook
status: active
owners:
  - recruitment-domain
tags:
  - recruitment
  - lifecycle
  - debug
applies_to:
  - smart-recruit-recruitment-service/**
  - smart-recruit-interview-service/**
  - smart-recruit-offer-service/**
  - smart-recruit-notification-service/**
  - smart-recruit-gateway/router/router.go
source_refs:
  - smart-recruit-recruitment-service/internal/domain/model/recruitment.go
  - smart-recruit-recruitment-service/internal/application/service/application_collaboration_taxonomy_service.go
  - smart-recruit-interview-service/internal/application/service/interview_service.go
  - smart-recruit-interview-service/internal/domain/service/interview_policy.go
  - smart-recruit-offer-service/internal/application/service/offer_service.go
  - smart-recruit-offer-service/internal/domain/model/status.go
  - smart-recruit-notification-service/internal/application/service/notification_service.go
  - smart-recruit-commons/internal/platform/events/envelope.go
  - smart-recruit-gateway/router/router.go
last_verified: 2026-07-14
review_after: 2026-10-14
---

# Debug Recruitment Lifecycle Issues

Trace lifecycle bugs from gateway route/permission through generated gRPC client into the owning service policy/application service, then application status update, event publication, notification handling, analytics projection, and frontend labels.

## Verification

Verified against current repository files on 2026-07-14.
