# Backend DDD Microservices Evolution Bounded Context Contract

Status: TASK-BDME-008 architecture contract
Last verified: 2026-07-11

## Purpose

This document defines the target bounded-context ownership contract for the
backend DDD and microservices migration. It is an enforceable architecture
baseline for later TASKs, but TASK-BDME-008 itself does not move code, change
public APIs, change database schema, or alter runtime behavior.

The contract applies during both phases:

- Modular monolith phase: context packages may live inside the current
  `logic-grpc-service` and `web-gin-service` deployment shape.
- Extracted service phase: the same ownership rules become service, API, event,
  database, worker, and deployment boundaries.

## Boundary Rules

1. A context owns its domain invariants, state transitions, repositories,
   write-side data, application use cases, adapters, tests, and runtime
   configuration.
2. The gateway owns transport and policy only. It may translate HTTP to service
   calls and enforce route policy, but it must not own domain state machines or
   persistence rules.
3. Cross-context writes must use domain events, Outbox/Inbox, idempotent
   consumers, or explicitly approved Saga/compensation flows.
4. Cross-context reads must use owned APIs, projections, read models, or
   approved compatibility adapters. New direct cross-context joins are
   forbidden.
5. Shared database access is transitional only. Every shared table or
   cross-context repository use must have an owner, exception record, removal
   plan, and final-state alternative.
6. Public HTTP, protobuf, authentication, authorization, deployment routing,
   and schema changes require their own scoped TASK and confirmation.
7. Analytics final data source is domain-event projections or owned read models.
   It must not depend on transitional service read APIs as its final source.

## Target Context Matrix

| Context | Owned aggregates and invariants | Owned data | Owned interfaces | Forbidden ownership |
|---|---|---|---|---|
| API Gateway | None for business state; owns request policy invariants such as auth cookie handling, route RBAC attachment, rate limits, quotas, body limits, request metadata, and SSE transport behavior | No transactional domain tables; may own ephemeral transport/session helper state only when scoped | Public HTTP routes, middleware, request/response mapping, gRPC or service clients, Swagger/docs, SSE stream transport | Domain state machines, repositories, database writes, lifecycle transitions, analytics calculations, AI runtime decisions |
| Identity and Access | User identity, account type, credentials, refresh token lifecycle, token version invalidation, RBAC role/scope semantics, audit events | users, refresh tokens, roles, permissions, user-role bindings, audit logs, identity-owned token/cache keys | Auth, current principal, token refresh/logout/invalidation, RBAC/scope checks, service authorization interfaces, identity audit events | Recruitment lifecycle, candidate profile/resume/application data, interview/offer lifecycle decisions, AI memory/runtime, notification content decisions |
| Recruitment | Jobs, job taxonomy, candidate recruitment profile, resumes, applications, application rounds/status, recruitment collaboration workspace, candidate-job match requests as recruitment use cases | jobs, departments/locations needed by jobs, candidate recruitment profiles, resumes and resume text, applications, application transitions, collaboration notes/tags/tasks, candidate-job match snapshots owned by recruitment workflow | Job, candidate profile, resume, application, collaboration, and recruitment intelligence APIs/events | Auth/RBAC internals, interview feedback internals, offer decision internals, notification persistence/unread state, AI runtime configuration, final analytics read models |
| Interview | Interview schedules, interviewer assignments, interviewer feedback, interview task state, interview lifecycle events | interviews, interview feedback, interviewer task records, interview-specific status and audit fields | Interview scheduling/feedback APIs, interview lifecycle events, interviewer task interfaces | Application master lifecycle except through approved application APIs/events, offer lifecycle, candidate profile ownership, notification persistence, AI memory/runtime |
| Offer | Offer draft/send/withdraw/accept/reject lifecycle, offer event history, candidate offer decision semantics | offers, offer events, offer decision timestamps and audit fields | Offer management APIs, candidate offer response APIs, offer lifecycle events | Application master lifecycle except through approved events/Saga/API, interview feedback, notification persistence, analytics projections |
| Notification | Notification records, unread counts, delivery state, realtime delivery projections, email dispatch coordination, notification delivery idempotency | notifications, notification read state, email logs/intents, delivery projection state, notification cache keys | Notification list/unread/summary/mark-read/SSE APIs, notification-created events, email dispatch consumers | Source domain business decisions, application/interview/offer status ownership, AI runtime decisions, analytics ownership |
| AI Agent | HR and candidate AI chat sessions, agent runs, run events, prompt/skill/memory orchestration, embedding coordination, MCP governance, provider/model fallback, AI usage audit, AI-derived intelligence | chat sessions, session summaries, memories, agent runs/events, prompt templates, agent configs, skills, MCP policies/logs, embedding provider/model config, embeddings, AI usage logs, AI-derived candidate/job/recruiting intelligence outputs | AI chat/run APIs, streaming events, admin AI config APIs, embedding/config APIs, skill/memory/retrieval APIs, AI domain events | Candidate recruitment profile/resume/application master data, HR/candidate account state, notification delivery ownership, final analytics read models |
| Analytics | Reporting projections, funnel/time-in-stage metrics, operational dashboards, immutable analytic snapshots derived from events | analytics read models, projection checkpoints, aggregate metric tables, reporting caches | Analytics query APIs, projection consumers, backfill/replay tools | Transactional writes to source domains, direct final dependency on transitional service read APIs, ownership of status transition rules |
| Platform and Worker Runtime | Outbox/Inbox infrastructure, worker process lifecycle, idempotency envelopes, retry/dead-letter handling, dependency health conventions, shared operational adapters | Outbox/Inbox tables until assigned per service, worker checkpoints, retry/dead-letter metadata, operational heartbeat state | Worker binaries, queue consumers, publisher adapters, replay/DLQ runbooks, health/readiness interfaces | Source domain invariants, business status decisions, user-facing delivery semantics, API gateway route policy |

## Recruitment and AI Agent Boundary

Recruitment owns candidate recruitment facts and workflow state:

- Candidate recruitment profile fields required to apply for jobs.
- Resume object references, parsed resume text, resume validation, and resume
  lifecycle needed for application and matching workflows.
- Application state, application rounds, status transitions, collaboration
  workspace, and candidate-job match records used as recruitment workflow
  evidence.

AI Agent owns AI-derived intelligence and runtime artifacts:

- Chat sessions, agent run state, run traces, memory recall, prompt assembly,
  skills, MCP governance, provider/model configuration, and embedding
  lifecycle.
- AI-generated candidate/job insights, explanations, extraction outputs, and
  semantic retrieval metadata when those artifacts are not the recruitment
  system of record.

Boundary rule: AI Agent may read recruitment-owned data through approved APIs,
events, or adapters and may produce derived intelligence. It must not become
the owner of candidate profile, resume, application, or application status
truth. Recruitment may store stable workflow decisions that reference AI
outputs, but it must not embed request-time AI runtime selection, memory
ranking, provider fallback, or MCP policy decisions.

## Analytics Boundary

Analytics owns read models, projections, and metrics. Its final source is
domain events from owning contexts plus replayable projection state. Transitional
reads from `logic-grpc-service` repositories or service APIs are allowed only as
approved migration adapters with an exception record.

Analytics must not:

- Write back to recruitment, interview, offer, notification, identity, or AI
  transactional state.
- Depend on direct joins across source-domain tables as the final architecture.
- Treat a transitional service read API as the final source of truth.
- Recompute source-domain lifecycle rules independently of domain events or
  owned projection semantics.

## Dependency Direction

Allowed dependency directions during modular monolith:

```text
interfaces -> application -> domain
application -> infrastructure ports
infrastructure adapters -> external systems
gateway -> context interfaces or clients
workers -> context application ports or event handlers
projection consumers -> analytics application ports
```

Forbidden dependency directions:

- Domain packages importing gateway, handler, transport, repository, or worker
  packages.
- One context importing another context's repository implementation to write or
  mutate owned tables.
- Analytics importing source-domain repositories as its final read path.
- Gateway importing domain repositories or running business state transitions.
- AI Agent importing recruitment repositories to mutate candidate profile,
  resume, application, or status truth.

When a direct dependency remains temporarily, it must be represented as a named
compatibility adapter at the application boundary and tracked as an exception.

## Transitional Exception Record

Every allowed exception must be documented in the TASK report that introduces or
keeps it. Use this shape:

| Field | Required value |
|---|---|
| Exception ID | Stable id, for example `BDME-EX-001` |
| TASK | TASK that introduced or reviewed the exception |
| Source context | Context depending on another context |
| Target owner | Owning context of the data/API/repository |
| Dependency type | repository read, repository write, table read, table write, service API, event, cache, worker, or config |
| Reason | Why the exception is required during migration |
| Risk | Consistency, privacy, security, availability, rollback, or observability risk |
| Guardrail | Tests, static checks, adapter boundary, feature flag, idempotency, or monitoring |
| Removal plan | Target TASK or condition that removes the exception |
| Approval | Human confirmation when required by TASK scope |

Exception records are not permanent architecture. Final readiness must prove
that exceptions are removed or explicitly approved with removal plans.

## Enforcement Expectations

Later TASKs should turn this contract into increasingly automated checks:

- DDD module skeletons must follow `domain/`, `application/`,
  `infrastructure/`, `interfaces/`, and `tests/` ownership.
- Boundary checks should detect forbidden imports and direct repository access
  across contexts.
- Event and Outbox/Inbox tasks should replace cross-context writes before
  extraction depends on them.
- Service extraction TASKs should show that extracted services preserve the
  owned interfaces and do not silently borrow another context's persistence.
- Final readiness must compare remaining imports, tables, events, APIs, and
  workers against this contract and the exception register.

## Current Mapping Notes

The current repository still centralizes many contexts inside
`logic-grpc-service/service`, `logic-grpc-service/repository`, and
`logic-grpc-service/model`. That is acceptable before modularization only
because this document is a target contract. Later TASKs must move code behind
the context boundaries before physical service extraction.

Representative current mapping:

- Gateway: `web-gin-service/router`, `web-gin-service/middleware`,
  `web-gin-service/handler`, and `web-gin-service/rpc`.
- Identity: auth, RBAC, token, user, and audit services/repositories.
- Recruitment: job, candidate profile, resume, application, collaboration, and
  recruiting intelligence services/repositories.
- Interview: interview service and repository.
- Offer: offer service and repository.
- Notification: notification service, notification worker, notification
  repository, email consumer, and notification gateway handler.
- AI Agent: AI service, candidate AI service, agent run, prompt, skill, memory,
  embedding, MCP, provider/model configuration, and usage audit components.
- Analytics: analytics service/repository and future event projections.
- Platform/workers: RabbitMQ, Outbox publisher, consumers, migrations, health,
  config, logging, cache, object storage, and operational runbooks.
