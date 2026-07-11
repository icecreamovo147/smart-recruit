---
schema_version: 1
id: recruitment-lifecycle
title: Recruitment lifecycle domain
kind: domain
status: active
owners:
  - recruitment-domain
tags:
  - recruitment
  - lifecycle
  - application
  - status
applies_to:
  - logic-grpc-service/service/job_service.go
  - logic-grpc-service/service/application_service.go
  - logic-grpc-service/service/interview_service.go
  - logic-grpc-service/service/offer_service.go
  - logic-grpc-service/service/collaboration_service.go
  - logic-grpc-service/model/status.go
source_refs:
  - logic-grpc-service/model/status.go
  - logic-grpc-service/service/application_service.go
  - logic-grpc-service/service/recruitment_lifecycle_process_manager.go
  - logic-grpc-service/service/interview_service.go
  - logic-grpc-service/service/offer_service.go
  - logic-grpc-service/service/collaboration_service.go
  - logic-grpc-service/repository/application_repo.go
  - logic-grpc-service/proto/recruitment.proto
last_verified: 2026-07-10
review_after: 2026-10-08
---

# Recruitment Lifecycle Domain

The recruitment lifecycle is centered on application rounds. Jobs receive applications, applications move through status keys, interviews and offers mutate or depend on those statuses, and collaboration surfaces aggregate notes, tags, tasks, interviews, offers, and timeline events.

## Core Flow

- `ApplicationService.ApplyJob` requires a complete candidate profile, a valid resume, and an online job, then creates a new application round.
- `model/status.go` defines stable status keys, legacy numeric mappings, candidate-safe labels, HR labels, and terminal statuses.
- `ApplicationService.UpdateApplicationStatus` validates target status keys, allowed transitions, reason requirements for closeout states, scope access, and current-round constraints.
- `RecruitmentLifecycleProcessManager` is the explicit process-manager boundary for Interview and Offer workflows that must advance application status or write application transition audit records inside an existing transaction.
- Interview scheduling and feedback use interview services/repositories and affect HR, candidate, and interviewer surfaces.
- Offer creation, send, withdraw, accept, and reject update offer state in Offer service transactions and route application lifecycle transitions through `RecruitmentLifecycleProcessManager`.
- Collaboration workspace composes applications, notes, tags, tasks, interviews, offers, and timeline events for staff workflows.

## Cross-Surface Impact

- HR surfaces include job management, application lists, candidate detail, interview scheduling, offer management, analytics, and collaboration workspace.
- Candidate surfaces include application progress, interviews, offers, notifications, profile, and resume.
- Interviewer surfaces include assigned interview tasks, details, feedback, notifications, and profile.
- Analytics depends on status history and lifecycle timestamps; lifecycle changes can alter dashboard, funnel, and time-in-stage results.

## Review Triggers

- Adding, renaming, or reclassifying application status keys.
- Changing transition rules, terminal statuses, reapplication rules, or required reasons.
- Changing interview or offer behavior that updates application status.
- Changing timeline, collaboration, analytics, or notification side effects.
- Changing candidate-facing labels or HR-facing labels.

## Verification

Verified against status model, application service, recruitment lifecycle process manager, interview service, offer service, collaboration service, application repository, and protobuf messages on 2026-07-11.
