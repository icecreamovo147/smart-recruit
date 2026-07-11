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
  - debug
  - status
  - notification
applies_to:
  - logic-grpc-service/service/application_service.go
  - logic-grpc-service/service/interview_service.go
  - logic-grpc-service/service/offer_service.go
  - logic-grpc-service/service/collaboration_service.go
  - logic-grpc-service/service/notification_service.go
source_refs:
  - logic-grpc-service/model/status.go
  - logic-grpc-service/service/application_service.go
  - logic-grpc-service/service/interview_service.go
  - logic-grpc-service/service/offer_service.go
  - logic-grpc-service/service/collaboration_service.go
  - logic-grpc-service/service/notification_service.go
  - logic-grpc-service/service/outbox_publisher.go
  - logic-grpc-service/service/inbox_consumer.go
  - logic-grpc-service/repository/outbox_repo.go
  - logic-grpc-service/repository/inbox_repo.go
  - web-gin-service/router/router.go
last_verified: 2026-07-11
review_after: 2026-10-08
---

# Debug Recruitment Lifecycle Issues

Use this runbook when application status, interview, offer, collaboration, analytics, or notification behavior appears inconsistent.

## Diagnostic Path

1. Identify the actor surface: HR, candidate, interviewer, or background worker.
2. Locate the domain service that owns the mutation: application, interview, offer, collaboration, or notification.
3. For status issues, check `model/status.go`, transition validation, terminal status logic, and application transition records.
4. For interview issues, check scheduling/update/cancel/feedback service logic and interviewer/candidate surfaces.
5. For offer issues, check offer status, application status mutation, offer events, notifications, and email outbox writes in the same transaction.
6. For missing notifications, check outbox writes, inbox duplicate/failed/dead status, worker processing, retry/dead-letter status, notification records, account type, cache invalidation, and SSE stream.
7. For analytics drift, check whether status timestamps or transition records changed.
8. For authorization-like symptoms, also review RBAC and data scope knowledge.

## Useful Anchors

- Status keys and labels: `logic-grpc-service/model/status.go`
- Application round and status mutation: `logic-grpc-service/service/application_service.go`
- Interview lifecycle: `logic-grpc-service/service/interview_service.go`
- Offer lifecycle: `logic-grpc-service/service/offer_service.go`
- Collaboration workspace/timeline: `logic-grpc-service/service/collaboration_service.go`
- Notification reads/events: `logic-grpc-service/service/notification_service.go`
- Outbox retry, dead-letter, and backlog stats: `logic-grpc-service/repository/outbox_repo.go`
- Inbox duplicate, failed, and dead-letter diagnostics: `logic-grpc-service/repository/inbox_repo.go`

## Safety

Use sanitized IDs and status keys in reports. Do not include candidate personal data, resume text, email content from production, or raw notification payloads.
