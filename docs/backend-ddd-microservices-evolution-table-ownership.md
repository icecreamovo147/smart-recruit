# Backend DDD Microservices Evolution Table Ownership

This document summarizes the executable table ownership manifest for the backend DDD and microservices evolution feature. The machine-readable source of truth is `docs/backend-ddd-microservices-evolution-table-ownership-manifest.json`.

## Scope

- Source schema: `db.sql`.
- Current table count: 67.
- Target owner contexts: `identity`, `recruitment`, `interview`, `offer`, `notification`, `ai-agent`, `analytics`, and `platform-workers`.
- Verification command: `node scripts/check-table-ownership.mjs`.

## Ownership Summary

| Owner | Tables |
| --- | ---: |
| identity | 9 |
| recruitment | 18 |
| interview | 2 |
| offer | 2 |
| notification | 2 |
| ai-agent | 30 |
| analytics | 2 |
| platform-workers | 2 |

## Transitional Shared Access

The manifest treats transitional shared database access as explicit debt, not implicit ownership.

| Table | Owner | Accessor | Removal Plan |
| --- | --- | --- | --- |
| `applications` | recruitment | interview/offer via RecruitmentLifecycleProcessManager | Replace with domain events or owned Recruitment API after Interview/Offer cutover parity is validated. |
| `application_status_transitions` | recruitment | interview/offer via RecruitmentLifecycleProcessManager | Route lifecycle updates through Recruitment-owned API/events only. |
| `event_outbox` | platform-workers | source domains | Move producers behind per-service outbox ports and enforce event contract checks. |
| `event_inbox` | platform-workers | all MQ consumers | Assign per-worker/service inbox ownership during worker cutover. |
| `analytics_projection_events` | analytics | platform-workers projection consumers | Move projection workers into Analytics service runtime with lag/readiness evidence. |
| `analytics_projection_checkpoints` | analytics | platform-workers projection consumers | Own checkpoint updates inside Analytics projection consumer runtime. |

## Review Rules

- Any new table in `db.sql` must be added to the manifest with an owner, allowed readers, and allowed writers.
- Any shared writer must be represented as transitional shared access with owner, accessor, reason, risk, and removal plan.
- Schema, migration, repository, and model changes should run `node scripts/check-table-ownership.mjs` together with the migration checks for the affected backend service.
- Schema or physical database separation must follow `docs/backend-ddd-microservices-evolution-schema-separation-plan.md`.
- Gateway or public API changes must not infer ownership from transport routing. Table ownership follows the target domain context in the manifest.
