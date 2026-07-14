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
  - smart-recruit-recruitment-service/**
  - smart-recruit-interview-service/**
  - smart-recruit-offer-service/**
  - smart-recruit-proto/**
  - hr-frontend/src/views/hr/**
  - user-frontend/src/views/candidate/**
source_refs:
  - smart-recruit-proto/proto/recruitment.proto
  - smart-recruit-proto/recruitment/pb/recruitment.pb.go
  - smart-recruit-recruitment-service/internal/runtime/runtime.go
  - smart-recruit-recruitment-service/internal/domain/policy/recruitment.go
  - smart-recruit-recruitment-service/internal/interfaces/grpc/adapters.go
  - smart-recruit-recruitment-service/internal/infrastructure/persistence/native_adapters.go
  - smart-recruit-recruitment-service/cmd/recruitment-service/main.go
  - smart-recruit-interview-service/internal/runtime/runtime.go
  - smart-recruit-offer-service/internal/runtime/runtime.go
  - smart-recruit-deploy/mysql-table-ownership.json
  - db.sql
last_verified: 2026-07-14
review_after: 2026-10-14
---

# Recruitment Domain Model

Recruitment owns jobs, candidates, resumes, applications, taxonomy, invite/admin subsets, and collaboration APIs. Interview owns schedules and feedback. Offer owns offer lifecycle and offer events. The shared wire contract is in `smart-recruit-proto/`, and cross-context table access is governed by the ownership manifest.

Application records and application status transitions are Recruitment-owned data. Services that need application state should read `ApplicationOwnerService.GetApplicationSnapshot`; services that need to advance application lifecycle should call `ApplicationOwnerService.ApplyApplicationLifecycleTransition` instead of carrying local application repositories.

Recruitment service runtime is no longer bootstrapped from `internal/legacydomain` repositories/services, and the Recruitment service-local legacy directory has been retired. `cmd/recruitment-service` wires a local native persistence bundle that owns job, taxonomy/admin, candidate/resume, application lifecycle, collaboration, usage audit, outbox, and application-owner contract behavior for the active runtime.

## Verification

Verified against current repository files on 2026-07-14.
