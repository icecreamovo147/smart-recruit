# Backend DDD Microservices Evolution Schema Separation Plan

Last verified: 2026-07-12

This plan prepares service-specific schema or physical database separation after bounded contexts, service binaries, table ownership, and cutover controls are proven. It does not change migrations, GORM models, `db.sql`, public APIs, deployment traffic, or runtime configuration.

## Authority

- Source schema: `db.sql`.
- Ownership source: `docs/backend-ddd-microservices-evolution-table-ownership-manifest.json`.
- Drift check: `node scripts/check-table-ownership.mjs`.
- Target RTO: 30 minutes.
- Target RPO: 5 minutes.

## Preconditions

Schema or physical database separation may start only after all of these are true for the affected context:

- The table ownership manifest validates against `db.sql`.
- The service binary exists, compiles, and has rollback-safe routing controls where it receives traffic.
- Public HTTP and protobuf behavior remains compatible or has an explicitly scoped compatibility TASK.
- Cross-domain writes are removed or represented by approved transitional access with owner, reason, risk, and removal plan.
- Consumers that replace shared writes are idempotent and have replay or dead-letter recovery evidence.
- Read paths that cross boundaries use owned APIs, projections, or approved compatibility adapters instead of new direct joins.
- Backups, point-in-time restore, migration dry run, reconciliation, and rollback ownership are assigned before production execution.

## Target Schema Units

The initial separation unit is schema-per-context. Physical databases may follow only after operational evidence proves backup, restore, latency, connection limits, and dependency failure behavior for the schema unit.

| Context | Initial schema unit | Owned tables |
| --- | --- | --- |
| identity | `identity` | `users`, `refresh_tokens`, `invite_codes`, `roles`, `permissions`, `role_permissions`, `user_roles`, `user_data_scopes`, `authorization_audit_logs` |
| recruitment | `recruitment` | `jobs`, `candidate_profiles`, `resumes`, `resume_parse_runs`, `resume_profiles`, `resume_educations`, `resume_experiences`, `resume_projects`, `resume_skills`, `applications`, `application_status_transitions`, `departments`, `job_locations`, `department_locations`, `candidate_notes`, `candidate_tags`, `candidate_tag_assignments`, `follow_up_tasks` |
| interview | `interview` | `interview_schedules`, `interview_feedback` |
| offer | `offer` | `offers`, `offer_events` |
| notification | `notification` | `notifications`, `email_logs` |
| ai-agent | `ai_agent` | `ai_chat_sessions`, `ai_chat_history`, `ai_session_summaries`, `ai_tool_traces`, `agent_runs`, `agent_run_events`, `agent_run_steps`, `candidate_match_evaluations`, `candidate_match_evidence`, `ai_memories`, `ai_embeddings`, `third_party_usage_logs`, `ai_usage_auth_contexts`, `llm_providers`, `llm_models`, `embedding_providers`, `embedding_models`, `prompt_templates`, `prompt_versions`, `agent_configs`, `agent_tool_bindings`, `mcp_servers`, `mcp_tool_logs`, `mcp_tool_policies`, `agent_capability_bindings`, `ai_skills`, `ai_skill_versions`, `ai_skill_tools`, `agent_skills`, `agent_skill_versions` |
| analytics | `analytics` | `analytics_projection_events`, `analytics_projection_checkpoints` |
| platform-workers | `platform_workers` | `event_outbox`, `event_inbox` |

## Transitional Shared Access

These shared database access points must be removed or re-approved before physical database separation:

| Table | Owner | Transitional accessor | Required exit |
| --- | --- | --- | --- |
| `applications` | recruitment | interview/offer via RecruitmentLifecycleProcessManager | Replace lifecycle writes with Recruitment-owned API or domain events. |
| `application_status_transitions` | recruitment | interview/offer via RecruitmentLifecycleProcessManager | Move audit writes behind Recruitment-owned API or events. |
| `event_outbox` | platform-workers | source domains | Move producers behind service-owned outbox ports with event contract checks. |
| `event_inbox` | platform-workers | all MQ consumers | Assign inbox ownership per worker or service consumer group. |
| `analytics_projection_events` | analytics | platform-workers projection consumers | Move projection ingestion into Analytics-owned worker runtime. |
| `analytics_projection_checkpoints` | analytics | platform-workers projection consumers | Move checkpoint updates into Analytics-owned worker runtime. |

## Expand-Contract Flow

Every separation TASK must use expand-contract steps so rollback remains possible:

1. Expand: add only additive migration objects for the new schema unit, such as empty target tables, compatibility views, or copy-safe indexes.
2. Backfill: copy owner tables in deterministic batches with resumable cursors, stable ordering, row counts, and checksum records.
3. Dual write or event replay: mirror owner writes through service-owned ports, Outbox events, or idempotent consumers. Direct dual writes are allowed only as a scoped transitional adapter with explicit rollback.
4. Dual read or shadow read: compare reads from current and target storage without changing user-visible behavior.
5. Reconcile: compare row counts, owner key sets, checksums, event positions, and projection lag.
6. Cut over: route the owning service to the separated schema only after compatibility, reconciliation, readiness, and rollback evidence pass.
7. Contract: remove compatibility views, old write paths, or direct joins only in later scoped TASKs after rollback windows close.

Non-additive changes, destructive migrations, table moves, table renames, dropped columns, and foreign-key rewrites are contract steps. They must not be bundled with expand/backfill/cutover work.

## Rollback Plan

Rollback must prioritize correctness over partial separation:

- Traffic rollback: route gateway or service clients back to the previous monolith or schema source using the scoped route-mode switch.
- Write rollback: disable target-schema writes or consumers before re-enabling old writes to avoid divergent updates.
- Data rollback: preserve target copies for diagnosis; do not delete separated schema data until reconciliation confirms the rollback state.
- Event rollback: pause affected consumers, record last safe offsets/checkpoints, and replay from the last committed source-domain event when resuming.
- Config rollback: revert service database DSNs and worker ownership flags through scoped config changes only.
- Time objective: restore stable user-visible behavior within 30 minutes.
- Data objective: recover to a point no more than 5 minutes behind committed source data.

## Reconciliation

Each separation TASK must define reconciliation evidence before cutover:

- Row counts per owner table before and after backfill.
- Primary key set comparison for copied tables.
- Deterministic checksum per table or per batch for high-volume tables.
- Referential reference checks for IDs that cross service boundaries.
- Outbox and Inbox checkpoint comparison for migrated event flows.
- Analytics projection lag and duplicate handling checks for read-model separation.
- Manual exception log for rows intentionally excluded by retention, soft delete, or privacy policy.

## Migration and Model Rules

TASK-BDME-047 does not modify migrations, models, repositories, or `db.sql`.

Any later TASK that changes schema or models must update and verify all affected surfaces:

- SQL migrations under `logic-grpc-service/migrations/`.
- `db.sql` when it represents the current schema baseline.
- GORM models under `logic-grpc-service/model/`.
- Repositories and transaction boundaries under `logic-grpc-service/repository/`.
- Table ownership manifest and `node scripts/check-table-ownership.mjs`.
- Migration runner, MySQL consistency, repository, and affected service tests.
- Rollback, reconciliation, RTO, and RPO evidence in that TASK report.

## Human Gate

The feature pipeline has a user-provided global gate override, but production schema separation remains a protected activity. A future TASK that performs actual schema, database, deployment, traffic, public API, or security changes must record scoped approval and evidence before implementation.
