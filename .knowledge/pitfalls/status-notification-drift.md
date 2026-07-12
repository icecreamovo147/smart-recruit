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
  - outbox
  - analytics
applies_to:
  - logic-grpc-service/model/status.go
  - logic-grpc-service/service/application_service.go
  - logic-grpc-service/service/interview_service.go
  - logic-grpc-service/service/offer_service.go
  - logic-grpc-service/service/notification_service.go
source_refs:
  - logic-grpc-service/model/status.go
  - logic-grpc-service/service/application_service.go
  - logic-grpc-service/service/recruitment_lifecycle_process_manager.go
  - logic-grpc-service/service/offer_service.go
  - logic-grpc-service/service/notification_service.go
  - logic-grpc-service/service/outbox_publisher.go
last_verified: 2026-07-10
review_after: 2026-10-08
---

# Status and Notification Drift Pitfall

Application status, notifications, offer events, timeline, and analytics are tightly coupled. A status change that does not update side effects can look correct in one view while silently breaking another.

## Trigger Conditions

- Adding or renaming application status keys.
- Changing offer or interview logic that mutates application state.
- Changing notification payloads, account type, links, or outbox topics.
- Changing analytics that depend on status timestamps or transition records.
- Changing candidate-safe or HR-facing labels.

## Risk

Candidates may miss a status update, HR may see timeline gaps, analytics may count an old state, or notifications may be delivered to the wrong account type. These bugs often survive local route tests because the primary mutation succeeds.

## Prevention

- Review status transition validation, transition audit rows, outbox writes, notification records, timeline composition, and analytics together.
- Keep Interview and Offer application-status side effects behind `RecruitmentLifecycleProcessManager`; direct writes in those services risk missing transition rows, terminal round closure, or future event-projection behavior.
- Keep candidate-facing labels intentionally less specific than HR labels.
- Check offer and interview side effects when application states change.
- Verify notification account type and link target for candidate, staff, and interviewer recipients.

## Verification

This pitfall was verified from status model, application service, recruitment lifecycle process manager, offer service, notification service, and outbox publisher on 2026-07-11.
