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
  - smart-recruit-interview-service/internal/runtime/runtime.go
  - smart-recruit-offer-service/internal/runtime/runtime.go
  - smart-recruit-deploy/mysql-table-ownership.json
  - db.sql
last_verified: 2026-07-14
review_after: 2026-10-14
---

# Recruitment Domain Model

Recruitment owns jobs, candidates, resumes, applications, taxonomy, invite/admin subsets, and collaboration APIs. Interview owns schedules and feedback. Offer owns offer lifecycle and offer events. The shared wire contract is in `smart-recruit-proto/`, and cross-context table access is governed by the ownership manifest.

## Verification

Verified against current repository files on 2026-07-14.
