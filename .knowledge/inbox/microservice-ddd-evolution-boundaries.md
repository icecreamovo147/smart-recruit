---
schema_version: 1
id: microservice-ddd-evolution-boundaries
title: Microservice DDD evolution boundary candidate
kind: candidate
status: draft
owners:
  - engineering-platform
tags:
  - candidate
  - architecture
  - services
  - ddd
applies_to:
  - smart-recruit-*-service/**
  - smart-recruit-domain-go/**
  - smart-recruit-gateway/**
  - smart-recruit-platform-go/**
  - smart-recruit-proto/**
  - docs/architecture/**
  - scripts/check-backend-boundaries.mjs
source_refs:
  - .spec/microservice-ddd-evolution/microservice-ddd-evolution-SPEC.md
  - .spec/microservice-ddd-evolution/microservice-ddd-evolution-SDD.md
  - .spec/microservice-ddd-evolution/TASKS.md
  - docs/architecture/microservice-ddd-evolution-guidelines.md
  - scripts/check-backend-boundaries.mjs
  - go.work
last_verified: 2026-07-13
review_after: 2026-10-11
---

# Microservice DDD Evolution Boundary Candidate

## Candidate Conclusion

The active knowledge base should be updated to describe the current backend service roots under `smart-recruit-*-service/` and the DDD migration path defined by `.spec/microservice-ddd-evolution`. Existing active documents still provide useful history, but several source references describe the older `logic-grpc-service` / `web-gin-service` layout and should be revised before they are treated as current navigation for this feature.

## Discovery Context

TASK-001 of `microservice-ddd-evolution` created the architecture guideline and strengthened backend boundary checks. The feature contract identifies the current state as independent Go service roots plus a shared `smart-recruit-domain-go` business kernel, with a target of service-local `domain/application/infrastructure/interfaces/runtime` boundaries.

## Evidence

- `go.work` includes independent backend modules such as `smart-recruit-offer-service`, `smart-recruit-interview-service`, `smart-recruit-notification-service`, `smart-recruit-identity-service`, `smart-recruit-recruitment-service`, `smart-recruit-analytics-service`, `smart-recruit-ai-agent-service`, and `smart-recruit-worker-service`.
- `.spec/microservice-ddd-evolution/microservice-ddd-evolution-SPEC.md` fixes the migration order: Offer, Interview, Notification, Identity, Recruitment, Analytics, AI Agent, Worker, then final commons rename.
- `.spec/microservice-ddd-evolution/microservice-ddd-evolution-SDD.md` defines the target service-local DDD shape.
- `docs/architecture/microservice-ddd-evolution-guidelines.md` records the DDD layering, shared kernel rules, Hard Stop list, validation commands, and knowledge impact handling.
- `scripts/check-backend-boundaries.mjs` now checks DDD layer imports for service-local `internal/domain`, `internal/application`, and `internal/interfaces` paths.

## Potential Impact

Future service migration TASKs will route through knowledge documents that currently emphasize legacy monolith names. Without an active update, Agents may over-read historical `logic-grpc-service` guidance or miss the new `smart-recruit-*-service` boundaries.

## Conflicts

No direct conflict with higher-authority source was found. This candidate exists because active knowledge is partially stale for navigation, while the active SPEC/SDD remains the executable contract.

## Proposed Destination

Update or supersede:

- `.knowledge/architecture/system-overview.md`
- `.knowledge/architecture/service-boundaries.md`
- `.knowledge/runbooks/local-development.md`

Add route coverage for:

- `smart-recruit-*-service/**`
- `smart-recruit-domain-go/**`
- `smart-recruit-gateway/**`
- `smart-recruit-platform-go/**`
- `smart-recruit-proto/**`
- `docs/architecture/**`
- `scripts/check-backend-boundaries.mjs`

## Required Approval

Approval is required before promoting this candidate to active knowledge because it changes repository-level architecture navigation and service ownership descriptions.
