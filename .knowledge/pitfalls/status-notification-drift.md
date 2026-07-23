---
schema_version: 1
id: status-notification-drift
title: Status and notification drift pitfall
kind: pitfall
status: active
owners:
  - recruitment-domain
tags:
  - status
  - notification
  - events
applies_to:
  - smart-recruit-recruitment-service/**
  - smart-recruit-interview-service/**
  - smart-recruit-offer-service/**
  - smart-recruit-notification-service/**
  - smart-recruit-commons/internal/platform/events/**
source_refs:
  - smart-recruit-recruitment-service/internal/domain/model/recruitment.go
  - smart-recruit-recruitment-service/internal/application/service/application_collaboration_taxonomy_service.go
  - smart-recruit-interview-service/internal/domain/service/interview_policy.go
  - smart-recruit-offer-service/internal/domain/model/status.go
  - smart-recruit-offer-service/internal/application/service/offer_service.go
  - smart-recruit-notification-service/internal/application/service/notification_service.go
  - smart-recruit-commons/internal/platform/events/envelope.go
last_verified: 2026-07-14
review_after: 2026-10-14
---

# Status and Notification Drift Pitfall

Status transitions, timeline events, notifications, analytics projections, and frontend labels can drift when only one context is updated. Review Recruitment, Interview, Offer, Notification, and Analytics impact together.

## Verification

Verified against current repository files on 2026-07-14.
