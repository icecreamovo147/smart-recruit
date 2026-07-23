# Backend DDD Microservices Evolution SDD

## 1. Existing Architecture Summary

The current backend has two main Go services:

- `web-gin-service`: Gin HTTP gateway with route registration in
  `router/router.go`, handlers under `handler/`, middleware under
  `middleware/`, generated gRPC clients under `recruitment/pb`, and gRPC client
  wiring in `rpc/client.go`.
- `logic-grpc-service`: core gRPC service with protobuf definitions under
  `proto/`, generated code under `recruitment/pb`, service implementations in
  `server/`, business orchestration in `service/`, persistence in
  `repository/` and `model/`, SQL migrations under `migrations/`, background
  workers, RabbitMQ, Redis, OSS, SMTP, and AI provider integration.

The gateway is already intended as a transport and policy boundary. It applies
request id propagation, access logs, panic recovery, security headers, CORS,
route-level timeouts, body limits, rate limits, quota middleware, risk blocking,
JWT authentication, current principal validation, and RBAC middleware. It calls
the logic service through generated gRPC clients and forwards request metadata.

The logic service currently assembles many repositories and services from
`main.go` and `service.NewServices`. It owns migrations, MySQL connection
pooling, RabbitMQ connection and worker startup, Redis-backed caches, object
storage, AI client initialization, email renderer/sender, default prompt and
agent seeding, legacy RBAC migration, gRPC server registration, and health
checks.

Existing production-readiness assets include Kubernetes deployments for
gateway, logic gRPC service, and worker; readiness/liveness probes; NetworkPolicy
for gRPC ingress; ConfigMaps and Secret examples; versioned migrations with
checksums and MySQL advisory lock; Outbox publishing; consumer retry/dead-letter
behavior; and 97 backend test files across logic and gateway workspaces.

## 2. Problem Analysis

The current shape has a service-oriented baseline, but the domain and runtime
boundaries are still too broad for long-term enterprise microservices.

Key problems:

- One logic service owns most business domains and infrastructure lifecycles.
- `service.Services` aggregates many unrelated domain services and workers,
  which increases coupling and makes dependency ownership hard to reason about.
- Repositories and models are shared inside one service, so domain data
  ownership is implicit rather than enforced.
- Cross-domain workflows such as application, interview, offer, notification,
  and AI intelligence are close enough that future physical extraction would be
  risky without modular boundaries first.
- Startup currently performs operational side effects such as applying
  migrations, seeding defaults, and legacy role migration. This is convenient
  but risky for multi-replica production rollouts.
- Internal gRPC auth has a production-required mode in config, but code defaults
  to optional for local friendliness. Service-to-service transport currently
  uses insecure gRPC credentials.
- Observability is mainly structured logs and request id propagation. Metrics,
  traces, SLO indicators, queue backlog monitoring, and worker semantic health
  are not yet complete.
- Worker health currently proves process liveness more than consumer or queue
  health.
- The repository has existing config safety checks, but production secret
  handling and committed local config must be treated as an early migration
  safety item.

The safest migration path is not immediate microservice extraction. It is:

1. Establish migration safety controls.
2. Modularize by DDD bounded context inside the current service.
3. Standardize domain events, Outbox/Inbox, idempotency, and observability.
4. Extract lower-risk asynchronous services.
5. Extract identity and core transactional services.
6. Split data ownership and finalize production readiness.

## 3. Proposed Design

### 3.1 Target Service Architecture

The final backend should contain these independently deployable units:

- `api-gateway`: HTTP API, auth cookie handling, route policy, rate limiting,
  quota checks, request limits, SSE transport, API docs, and protocol
  translation.
- `identity-service`: users, auth, token refresh, token invalidation, RBAC,
  scopes, service authorization, and audit logs.
- `recruitment-service`: jobs, candidate recruitment profile, resumes,
  applications, and application lifecycle state.
- `interview-service`: interview schedules, interviewer tasks, feedback, and
  interview lifecycle state.
- `offer-service`: offer creation, update, send, withdraw, candidate decision,
  and offer event history.
- `notification-service`: notification persistence, unread counts, email
  coordination, realtime delivery, and notification projections.
- `ai-agent-service`: AI chat, agent runs, prompts, skills, memory, embedding,
  MCP governance, provider fallback, and AI usage audit.
- `analytics-service`: domain-event projection read models, funnel metrics,
  time-in-stage metrics, interview/offer metrics, and reporting APIs.
- `worker-services`: async processors for outbox dispatch, notifications,
  resume parsing, email, embeddings, agent runs, projections, and backfills.

The monorepo can remain intact, but each service must have a clear build and
deployment boundary by final readiness.

### 3.2 Target DDD Package Shape

Each bounded context should converge on a local shape similar to:

```text
<service-or-domain>/
  domain/
    entity and value object definitions
    domain services
    state machines
    domain events
  application/
    use cases
    command/query handlers
    orchestration
    transaction boundaries
  infrastructure/
    repositories
    external clients
    message publishers/consumers
    persistence adapters
  interfaces/
    grpc handlers
    http adapters where applicable
    event handlers
  tests/
```

During the modular monolith phase this structure may live under
`logic-grpc-service/internal/<bounded-context>/` or another explicitly approved
path. During service extraction, the same boundaries become separate service
directories or binaries.

### 3.3 Migration Phases

Phase 0: Safety net and production baseline.

- Document current workflows, contracts, dependencies, and risks.
- Add or strengthen tests for core user workflows.
- Establish contract checks for HTTP and gRPC behavior.
- Add scope checks, reporting, and knowledge impact tracking through Harness.
- Remove or quarantine committed sensitive configuration and require external
  secret injection.
- Establish baseline observability and migration runbooks.

Phase 1: DDD modular monolith.

- Define bounded context ownership.
- Move code behind domain/application/infrastructure/interface boundaries
  inside the current deployment shape.
- Keep public behavior stable.
- Keep repository access inside owning domains or transitional adapters.
- Add boundary tests and import checks where possible.

Phase 2: Event-driven decoupling.

- Standardize domain event naming, schemas, metadata, and versioning.
- Standardize Outbox writes from transactional domains.
- Add Inbox or equivalent idempotency controls to consumers.
- Convert cross-domain writes to events and compensation workflows.
- Build read models where direct cross-domain queries would otherwise remain.

Phase 3: Service extraction.

- Extract Notification first using shadow consume and cutover.
- Extract AI Agent next because it is high-latency, externally dependent, and
  independently scalable.
- Extract Identity after auth/RBAC contracts are stable.
- Extract Recruitment, Interview, and Offer after state ownership and events are
  proven.
- Extract Analytics as an event-projection read-model service without a
  transitional dependency on transactional service read APIs.

Phase 4: Data ownership and HA.

- Assign table/schema ownership to each service.
- Remove or formally approve transitional shared database access.
- Introduce service-specific schema or physical database separation when safe.
- Add real readiness checks, worker health, metrics, traces, alerts, and load
  tests.
- Validate horizontal scaling and dependency failure drills.
- Use the confirmed initial targets: gateway normal APIs at 200 QPS, core
  writes at 50 QPS, AI/Embedding workloads at 10-20 concurrent tasks, ordinary
  API P95 below 300 ms, complex query P95 below 1 s, 99.5% monthly availability,
  RTO 30 minutes, and RPO 5 minutes.

Phase 5: Final convergence.

- Remove old adapters, direct cross-domain repository access, and obsolete
  routing paths.
- Update `.knowledge`, runbooks, deployment docs, and architecture diagrams.
- Run production readiness review, load tests, security checks, and rollback
  drills.

### 3.4 Target Ownership Matrix

| Context | Owns | Does not own |
|---|---|---|
| API Gateway | HTTP/SSE transport, cookie handling, route policy, rate limit, quota, request metadata | Domain state machines, persistence rules |
| Identity | users, auth, refresh tokens, RBAC, scopes, audit, token invalidation | recruitment lifecycle, interview lifecycle, offer lifecycle |
| Recruitment | jobs, resumes, applications, application state | offer decision internals, interview feedback internals |
| Interview | schedules, interviewer tasks, feedback | application master lifecycle except via events/API |
| Offer | offer lifecycle and events | application state mutation except via events/API |
| Notification | notification records, unread counts, delivery projections | source domain business decisions |
| AI Agent | chat, agent run, skill/prompt/memory, embedding, MCP, AI audit | HR/candidate domain state ownership |
| Analytics | event-projection reporting read models and queries | transactional writes and transitional service read API dependency |

## 4. Data Structure Changes

This draft does not implement schema changes. It defines the required migration
direction.

Expected data design changes in later TASKs:

- Add domain event metadata standards, including event id, event type, event
  version, aggregate type, aggregate id, producer, occurred time, request id,
  actor id where allowed, and payload.
- Extend or standardize existing Outbox tables to support event versioning,
  producer domain, idempotency keys, retry state, and observability.
- Add Inbox or consumer checkpoint tables for idempotent consumption by service
  and event id.
- Add event-projection read-model tables for analytics and cross-domain
  list/detail use cases that should not rely on direct joins or transitional
  service read APIs.
- Add ownership metadata or documentation for existing tables during transition.
- Split schemas or physical databases only after domain boundaries and service
  APIs are stable.

Any database schema, migration, model, or `db.sql` change is a hard stop item
and must be scoped to an implementation TASK with human confirmation.

## 5. API and Interface Changes

This draft does not change public APIs. It defines interface migration rules.

Expected interface changes in later TASKs:

- Introduce internal service contracts for Identity, Recruitment, Interview,
  Offer, Notification, AI Agent, and Analytics.
- Keep gateway HTTP behavior stable while replacing the single logic-service
  client with service-specific clients or routing adapters.
- Prefer direct gateway-to-extracted-service routing when a service is cut over.
  This migration does not require preserving the old `logic-grpc-service` gRPC
  surface as an interim backend facade.
- Keep protobuf source files and generated code synchronized in all affected
  service trees.
- Add versioning or compatibility adapters for any contract that must evolve.
- Introduce event contracts for cross-domain workflows.
- Add admin or diagnostic endpoints only when necessary for production
  operations and explicitly scoped.

Public HTTP behavior, protobuf changes, authentication behavior, and
authorization behavior require human confirmation before implementation.

## 6. Algorithm or Workflow Changes

### 6.1 Current Workflow Preservation

Existing workflows must be characterized and protected before changes:

- Auth: register, login, refresh, logout, current principal.
- Recruitment: public job list/detail, HR job management, profile, resume,
  application submission and status.
- Interview: schedule, update, cancel, feedback, interviewer task views.
- Offer: create, update, send, withdraw, accept, reject, event history.
- Notification: list, unread count, mark read, SSE stream.
- AI: candidate chat, HR chat, agent runs, skill/prompt/memory, resume
  intelligence, embedding, provider fallback.
- Analytics: dashboard and metrics queries backed by event-projection read
  models.

### 6.2 Domain Event Flow

Target cross-domain write flow:

```text
Owning application service
  -> local transaction
  -> domain state change
  -> Outbox event record
  -> Outbox dispatcher
  -> broker
  -> Inbox/idempotent consumer
  -> local projection or follow-up command
```

Consumer handlers must be idempotent. If a handler calls another service or
updates local state, the handler must record enough state to avoid duplicate
effects when RabbitMQ redelivers.

### 6.3 Saga and Compensation

For multi-step business processes that cross contexts, the design should prefer
orchestration in the owning domain or explicit process manager code over hidden
repository writes. If a later TASK identifies a process requiring compensation,
the TASK must define states, retry behavior, timeout behavior, manual recovery,
and observability.

### 6.4 Shadow and Cutover

Each extraction must use one of these patterns:

- Shadow read or shadow consume: new implementation observes data/events and
  records differences without affecting production state.
- Dual write with reconciliation: old and new paths receive writes, with
  explicit conflict detection and rollback.
- Dual read with fallback: read from new service and fall back to old path for a
  bounded transition period.
- Routed cutover: feature flag or gateway routing moves traffic by percentage,
  actor, endpoint, or environment.

Every cutover must include rollback steps and data reconciliation expectations.

## 7. Configuration Design

Configuration should converge on service-specific environment configuration
with safe defaults.

Required direction:

- Remove committed live secrets or production-like sensitive values.
- Keep example config files with placeholders only.
- Require external secret injection for MySQL, JWT, internal auth, AI provider,
  OSS, Redis, RabbitMQ, SMTP, and encryption keys.
- Make production profiles fail fast when required secrets or internal auth are
  missing.
- Separate hard dependency readiness from soft dependency degradation.
- Add service-specific runtime knobs for timeouts, retries, circuit breakers,
  rate limits, worker concurrency, queue names, event retention, and rollout
  flags only through scoped TASKs.

Do not add a new configuration framework unless explicitly approved.

## 8. Compatibility Strategy

Compatibility is maintained through staged adapters:

- Keep existing gateway routes while backend target services are introduced.
- Keep existing protobuf contracts until replacements are stable.
- Use gateway-level compatibility adapters when old public HTTP contracts need
  to route to newly extracted services.
- Avoid deleting legacy logic until shadow validation, parity tests, and
  rollback windows have passed.
- Preserve old database columns and tables during expand-contract migrations
  where schema compatibility is needed, but do not introduce an interim
  `logic-grpc-service` facade solely to preserve internal service routing.
- Document any behavior delta as a compatibility risk requiring confirmation.

The migration should avoid mixing structural refactors with product behavior
changes in the same TASK.

## 9. Error Handling and Fallback Design

Service-to-service calls:

- Use per-call deadlines.
- Classify errors into validation, authentication, authorization, not found,
  conflict, retryable dependency failure, timeout, and internal failure.
- Retry only read-only calls or explicitly idempotent commands.
- Include request id and safe actor context in logs.

Events:

- Preserve committed events in Outbox until published.
- Use consumer idempotency keys or Inbox records.
- Retry transient failures with bounded backoff.
- Dead-letter poison messages with safe diagnostics.
- Provide replay or repair runbooks.

External dependencies:

- Redis outage: documented fallback for rate limiting, token-version cache,
  notification cache, and presign cache.
- RabbitMQ outage: Outbox backlog accumulates and alerts; core transactions
  remain committed where business rules allow.
- MySQL outage: hard dependency failure for transactional services.
- OSS outage: upload/presign and resume parsing paths fail with user-safe
  messages.
- SMTP outage: email delivery retries or logs according to configuration.
- AI/Embedding provider outage: circuit breaker, fallback, quota/error metrics,
  and user-safe messaging.

## 10. Observability and Debug Output Design

Target telemetry:

- HTTP metrics: request count, latency buckets, status codes, route labels,
  request size failures, rate-limit/quota decisions.
- gRPC metrics: method count, latency, status, deadline exceeded, retries.
- Database metrics: pool stats, slow queries, migration status, error rates.
- Redis metrics: ping/read/write failures, fallback count, cache hit/miss where
  useful.
- RabbitMQ metrics: connection status, publish failures, consumer failures,
  retry count, dead-letter count, queue backlog, Outbox pending/retry age.
- Worker metrics: active workers, handled jobs, failures, processing latency,
  heartbeat/readiness.
- AI metrics: provider latency, timeout, retry, fallback, budget, circuit state,
  quota usage.
- Business metrics: application created, status transitioned, interview
  scheduled, offer sent/accepted/rejected, notification delivered, agent run
  completed/failed.

Trace and log strategy:

- Gateway request id remains mandatory.
- Trace or request id must propagate through gRPC metadata and event metadata.
- Logs must be structured and must not contain secrets, raw tokens, or
  unnecessary personal data.
- Cutover TASKs must record before/after metrics and rollback readiness.

## 11. Testing Strategy

Testing must grow with migration risk.

Baseline tests:

- Run `go test ./...` in `logic-grpc-service`.
- Run `go test ./...` in `web-gin-service`.
- Keep current unit and repository tests passing.

Contract tests:

- Verify gateway route permissions and auth behavior.
- Verify protobuf client/server compatibility.
- Verify HTTP response compatibility for core frontend flows.

Domain tests:

- Test state machines in domain/application layers.
- Test ownership boundaries and forbidden cross-domain repository access where
  static checks are feasible.
- Test idempotency keys, duplicate commands, and conflict handling.

Integration tests:

- Test MySQL migrations and model consistency.
- Test Outbox publish and consumer retry/dead-letter behavior.
- Test Redis fallback behavior.
- Test RabbitMQ failure and recovery behavior where integration dependencies
  are available.

Migration tests:

- Test shadow/dual-run parity for extracted services.
- Test rollback from new service to old implementation.
- Test expand-contract schema compatibility.
- Test data reconciliation scripts or checks.

HA and load tests:

- Validate gateway normal APIs at 200 QPS and core write operations at 50 QPS.
- Validate AI/Embedding workloads at 10-20 concurrent tasks.
- Validate ordinary API P95 latency below 300 ms and complex query P95 latency
  below 1 s.
- Validate AI workloads are asynchronous and do not block main transaction
  paths.
- Validate 99.5% monthly availability assumptions through readiness, rollout,
  and dependency failure evidence where practical in the available environment.
- Validate multiple replicas, rolling updates, worker scaling, and dependency
  failure drills.
- Record metrics evidence in TASK reports.

Security tests:

- Add secret scanning or equivalent checks.
- Validate production internal auth requirements.
- Validate RBAC and scope behavior after Identity extraction.
- Validate log/event redaction for sensitive data.

## 12. Migration Risks

- Big-bang extraction may break core workflows. Mitigation: modularize first,
  extract incrementally, and use direct gateway-to-service routing only inside
  scoped cutover TASKs with rollback plans.
- Proto drift may break gateway-service calls. Mitigation: synchronized proto
  changes, generated code updates, and contract tests.
- Schema drift may corrupt data. Mitigation: expand-contract migration,
  migration tests, model consistency checks, and rollback plans.
- Event duplication may create duplicate notifications, emails, or state
  transitions. Mitigation: Inbox/idempotency and consumer tests.
- Cross-domain reads may reintroduce coupling. Mitigation: read models,
  service APIs, and import/table ownership checks.
- Identity extraction may weaken authorization. Mitigation: RBAC matrix tests,
  token invalidation tests, and audit checks.
- Worker extraction may hide failures. Mitigation: semantic readiness,
  heartbeats, backlog metrics, and alerts.
- AI extraction may create inconsistent chat or run state. Mitigation: shadow
  runs, parity checks, durable run state, and fallback tests.
- Operational changes may exceed local development complexity. Mitigation:
  keep local scripts usable or document equivalent workflows.

## 13. Implementation Boundaries

Draft SPEC/SDD mode may only create:

- `.spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md`
- `.spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md`

Later Harness generation should scope implementation tasks narrowly.

Expected future implementation areas:

- `logic-grpc-service/service/**`
- `logic-grpc-service/repository/**`
- `logic-grpc-service/model/**`
- `logic-grpc-service/server/**`
- `logic-grpc-service/proto/**`
- `logic-grpc-service/migrations/**`
- `logic-grpc-service/mq/**`
- `logic-grpc-service/config/**`
- `web-gin-service/router/**`
- `web-gin-service/handler/**`
- `web-gin-service/middleware/**`
- `web-gin-service/rpc/**`
- `web-gin-service/proto/**`
- `deploy/**`
- `docker/**`
- `.knowledge/**`
- `docs/**`

Default hard stops for later TASKs:

- Package manifest or lockfile changes.
- New dependencies.
- Public HTTP or protobuf behavior changes.
- Database schema or migration changes.
- Authentication, authorization, permission, or security behavior changes.
- CI/CD or global deployment behavior changes.
- Deletes of legacy code paths.
- Service traffic cutovers.

## 14. Alternatives Considered

- Big-bang microservice rewrite: rejected because it creates excessive risk for
  current product workflows and makes rollback difficult.
- Keep the current two-service architecture indefinitely: rejected because it
  does not satisfy the requested microservices and DDD end state.
- Split databases first: rejected because data ownership must be proven before
  physical separation.
- Extract core Recruitment first: rejected as the first extraction because it is
  the highest-coupling transactional domain.
- Add a new infrastructure platform before modularization: rejected because the
  current stack already includes enough primitives for safe staged migration.
- Use distributed transactions for cross-service consistency: rejected as a
  default pattern; prefer Outbox/Inbox, events, and compensation.

## 15. Assumptions Requiring Confirmation

The following decisions were confirmed by the user after the initial draft:

- D-001: The final service set listed in the SPEC is the target boundary.
- D-002: The repository remains a monorepo while services become independently
  buildable and deployable.
- D-003: Go, Gin, gRPC, MySQL, Redis, RabbitMQ, and Kubernetes remain the
  baseline stack.
- D-004: Notification and AI Agent are extracted before core transactional
  services.
- D-005: Shared database use is acceptable only as a transitional state with
  explicit table ownership and removal plans.
- D-006: Existing frontend API behavior is preserved unless a later TASK scopes
  a coordinated frontend/backend contract change.
- D-007: Candidate/resume/application-related profile data belongs to
  Recruitment. AI-generated profiles, matching artifacts, embeddings, memory,
  and AI-derived intelligence belong to AI Agent.
- D-008: This migration does not require an interim backend facade preserving
  the old `logic-grpc-service` gRPC surface. Gateway can route directly to
  extracted services as each cutover TASK allows.
- D-009: Initial performance targets are gateway normal APIs at 200 QPS, core
  writes at 50 QPS, AI/Embedding at 10-20 concurrent tasks, ordinary API P95
  below 300 ms, complex query P95 below 1 s, and AI work kept asynchronous.
- D-010: Initial HA targets are 99.5% monthly availability, RTO 30 minutes, and
  RPO 5 minutes.
- D-011: Service-to-service security starts with internal TLS plus required
  `GRPC_INTERNAL_TOKEN`; mTLS is evaluated before final readiness, with no
  service mesh requirement in this SPEC.
- D-012: Observability target is Prometheus metrics, Grafana dashboards,
  OpenTelemetry traces, and structured logs.
- D-013: Retention defaults are successful Outbox/Inbox records for 30 days,
  failed/dead-letter records for 90 days, authorization audit logs for 180
  days, and desensitized AI traces for 30-90 days.
- D-014: Analytics is implemented directly to the final standard using
  domain-event projections/read models. Transitional service read APIs for
  Analytics are not part of this migration.

## 16. Open Questions

No open questions remain for the current SPEC/Harness preparation. The
retention defaults are confirmed for this feature, and the production-like
load-test and failure-drill environment is intentionally out of scope for this
draft.
