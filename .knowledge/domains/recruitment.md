---
schema_version: 1
id: recruitment
title: Recruitment domain model
kind: domain
status: active
owners:
  - recruitment-domain
tags:
  - recruitment
  - jobs
  - applications
  - interviews
  - offers
applies_to:
  - logic-grpc-service/service/job_service.go
  - logic-grpc-service/service/application_service.go
  - logic-grpc-service/service/interview_service.go
  - logic-grpc-service/service/offer_service.go
  - hr-frontend/src/views/hr/**
  - user-frontend/src/views/candidate/**
source_refs:
  - README.md
  - logic-grpc-service/proto/recruitment.proto
  - logic-grpc-service/service/application_service.go
  - db.sql
last_verified: 2026-07-10
review_after: 2026-10-08
---

# Recruitment Domain Model

The recruitment domain centers on jobs, candidates, applications, interviews, offers, notifications, and analytics. HR users manage jobs and application flow. Candidates browse jobs, maintain profile/resume data, apply, track progress, attend interviews, and respond to offers. Interviewers use staff-facing interview task and feedback flows.

Application state transitions and offer/interview lifecycle behavior are business rules. Verify them in service code, protobuf contracts, database schema, and tests before documenting or changing behavior.

## Core Concepts

- Job: published or managed recruitment opening with department and location context.
- Application: candidate-to-job relationship with status transitions and timeline.
- Interview: scheduled evaluation event, including interviewer feedback.
- Offer: post-interview decision artifact with send, withdraw, accept, and reject flows.
- Resume profile and match evaluation: AI-assisted extraction and candidate-job comparison.
- Notification: user-facing event delivery, including SSE and unread state.

## Change Impact Hints

- Status changes normally require service, repository, frontend display, and analytics review.
- Resume and candidate intelligence changes may touch AI quota, parsing, storage, and evaluation flows.
- Offer and interview changes usually affect both HR and candidate/interviewer surfaces.
- Permission changes belong to route/RBAC review and should not be inferred only from UI visibility.

## Verification

This domain summary was verified against `README.md`, protobuf definitions, application service code, and database schema on 2026-07-10.
