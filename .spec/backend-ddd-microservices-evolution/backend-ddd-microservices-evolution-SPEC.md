# Backend DDD Microservices Evolution SPEC

## 1. Background

Smart Recruit currently uses three Vue frontends, a Gin HTTP gateway in
`web-gin-service`, and a Go gRPC business service in `logic-grpc-service`.
The gateway owns HTTP routing, middleware, JWT/cookie handling, route-level
RBAC, rate limits, quotas, request size limits, SSE endpoints, Swagger, and
gRPC client calls. The logic service owns business services, repositories,
GORM models, migrations, AI orchestration, background workers, RabbitMQ,
Outbox publishing, Redis-backed caches, object storage, email, and protobuf
service implementations.

This structure is a useful service-oriented baseline, but the backend is not
yet a long-term enterprise-grade microservices and DDD architecture. The main
risks are concentrated around a large central logic service, broad service
aggregation, shared database ownership, mixed cross-domain state changes,
startup-time operational side effects, incomplete production governance, and
limited observability beyond structured logs.

The requested feature is a gradual architecture migration, not a big-bang
rewrite. The end state must be a standard, maintainable, production-capable
microservices plus DDD backend with decoupled business domains, measurable
concurrency support, and high availability. The migration itself must be
stable, reviewable, testable, and rollback-capable at every step.

## 2. Goals

- Evolve the backend to a microservices plus DDD architecture with explicit
  bounded contexts and clear service ownership.
- Keep the current user-facing behavior stable during migration.
- Establish a migration safety net before large structural changes.
- First modularize current backend code by DDD domain inside the existing
  deployment shape, then split services incrementally.
- Ensure business domains are sufficiently decoupled: no direct cross-domain
  repository or table access after service extraction.
- Move cross-domain write collaboration to domain events, Outbox/Inbox,
  idempotent consumers, and explicit Saga or compensation workflows where
  needed.
- Support horizontal scaling and high availability for gateway, business
  services, and workers.
- Improve production readiness: configuration safety, secret handling,
  health checks, graceful shutdown, metrics, traces, logs, alerts, runbooks,
  and rollback paths.
- Define acceptance criteria that can be verified by automated tests,
  integration checks, scope checks, deployment checks, and documented manual
  validation where needed.

## 3. Non-Goals

- Do not perform a big-bang rewrite of all backend services.
- Do not require all services to be physically split in the first implementation
  phase.
- Do not change product behavior unless a later TASK explicitly authorizes and
  tests that behavior change.
- Do not remove existing public HTTP or gRPC APIs without a confirmed
  deprecation and compatibility plan.
- Do not split databases before service and domain ownership boundaries are
  proven by tests and runtime validation.
- Do not introduce distributed transactions as the default cross-service
  consistency mechanism.
- Do not add new infrastructure dependencies unless a later TASK explicitly
  scopes and receives human confirmation for the dependency.
- Do not modify frontend product behavior except when required to preserve
  compatibility with backend service extraction.

## 4. User-Facing Behavior

- Candidate, HR, and interviewer workflows must remain behaviorally compatible
  throughout migration.
- Login, refresh, logout, route permission checks, and current principal
  behavior must remain stable unless a confirmed Identity migration TASK changes
  the contract.
- Public job browsing, candidate profile, resume upload, application,
  interview, offer, notification, and AI workflows must keep the same visible
  semantics during staged migration.
- During shadow or dual-run phases, only the approved production path may affect
  user-visible state unless a cutover TASK explicitly changes the active path.
- When a new service is introduced, rollback must allow traffic to return to
  the previous implementation without corrupting business data.
- Failure of non-critical asynchronous services such as notification or AI
  background processing must degrade gracefully and must not break core
  transactional flows unless the affected feature is explicitly hard dependent.

## 5. Functional Requirements

- FR-001: The feature must define target bounded contexts at minimum for
  Identity and Access, Recruitment, Interview, Offer, Notification, AI Agent,
  Analytics, and platform/workers.
- FR-002: The final architecture must expose independently deployable backend
  service units for API gateway, identity, recruitment, interview, offer,
  notification, AI agent, analytics, and worker workloads, unless a later
  approved SPEC revision narrows the target.
- FR-003: Each bounded context must have explicit ownership of domain model,
  application use cases, infrastructure adapters, interfaces, tests, and
  runtime configuration.
- FR-004: Gateway code must remain a transport and policy boundary; it must not
  own core business state machines or persistence rules.
- FR-005: Domain invariants and state transitions must be owned by domain or
  application services in the owning context.
- FR-006: Cross-domain write operations must be replaced by domain events,
  Outbox/Inbox, idempotent consumers, or explicit Saga/compensation workflows.
- FR-007: Cross-domain read needs must be satisfied through owned APIs, read
  models, projections, or explicitly approved compatibility adapters; new
  direct cross-domain joins are forbidden.
- FR-008: Each extracted service must have an explicit data ownership model.
  Shared database use may exist only as a transitional phase with documented
  table ownership and a migration plan.
- FR-009: Existing protobuf and HTTP contracts must remain synchronized between
  gateway and backend services throughout migration.
- FR-010: Every public API or protobuf change must be treated as a compatibility
  contract change and require human confirmation before implementation.
- FR-011: All asynchronous consumers must be idempotent and must record enough
  evidence to diagnose retries, duplicates, dead-letter cases, and poison
  messages.
- FR-012: Existing Outbox behavior must be standardized across domains before
  service extraction depends on it.
- FR-013: Services must propagate request id, authenticated actor identity, and
  account type where relevant.
- FR-014: Authentication, authorization, token invalidation, RBAC, scopes, and
  audit logging must become Identity-owned capabilities that other services
  consume through explicit interfaces.
- FR-015: Recruitment must own jobs, candidate recruitment profile data needed
  by recruitment workflows, resumes, applications, and application lifecycle
  state.
- FR-016: Interview must own interview schedules, interviewer tasks, feedback,
  and interview-specific state.
- FR-017: Offer must own offer lifecycle, offer events, and candidate offer
  decision state.
- FR-018: Notification must own notification persistence, unread counts, email
  dispatch coordination, and realtime delivery contracts.
- FR-019: AI Agent must own chat sessions, agent runs, prompt/skill/memory
  orchestration, embedding coordination, MCP governance, and AI provider
  fallback behavior.
- FR-020: Analytics must own reporting read models built from domain-event
  projections and must not write back into transactional domain state or depend
  on transitional service read APIs as its final data source.
- FR-021: Migration must include a documented shadow, dual-read, dual-write, or
  cutover strategy for every high-risk extraction.
- FR-022: Migration must include rollback plans for service cutovers, schema
  changes, event consumer changes, and API routing changes.
- FR-023: Feature flags or equivalent controlled routing must guard high-risk
  behavior changes and traffic cutovers.
- FR-024: A final architecture readiness review must prove that forbidden
  cross-domain repository imports, table writes, and state ownership violations
  are absent or documented as approved exceptions.
- FR-025: During service extraction, the gateway may call extracted services
  directly; this feature does not require preserving the old
  `logic-grpc-service` gRPC surface as an interim backend facade.

## 6. Non-Functional Requirements

- NFR-001: The migration must be incremental. Each TASK must be independently
  reviewable, scope-checked, testable, and rollback-aware.
- NFR-002: The system must support horizontal scaling of gateway and core
  backend services by avoiding required local in-memory state in request paths.
- NFR-003: Worker workloads must be independently scalable from request-serving
  workloads.
- NFR-004: Each service must expose liveness and readiness checks that reflect
  its true hard dependencies.
- NFR-005: Soft dependency degradation must be explicit. Examples include
  RabbitMQ backlog accumulation, AI provider fallback, SMTP unavailability, and
  Redis cache degradation.
- NFR-006: Each service must expose or record metrics sufficient to validate
  throughput, latency, errors, saturation, queue backlog, retry rates, and
  external dependency failures.
- NFR-007: Distributed tracing or trace-compatible request correlation must
  connect gateway requests, gRPC calls, workers, events, and downstream
  dependency calls.
- NFR-008: Production logs must be structured and must avoid secrets, tokens,
  raw credentials, and unnecessary candidate personal data.
- NFR-009: The final system must validate at least the following initial
  medium-production targets: gateway normal APIs at 200 QPS, core write
  operations at 50 QPS, AI/Embedding workloads at 10-20 concurrent tasks,
  ordinary API P95 latency below 300 ms, complex query P95 latency below 1 s,
  and AI work kept asynchronous so it does not block the main transaction path.
- NFR-010: The final system must target 99.5% monthly availability, RTO of 30
  minutes, and RPO of 5 minutes unless a later confirmed production SLO changes
  those values.
- NFR-011: The final architecture must include load test evidence for agreed
  concurrency targets and regression evidence for core workflows.
- NFR-012: The migration must not reduce existing test coverage for touched
  services and must add tests for newly introduced domain boundaries and
  integration contracts.
- NFR-013: All production configuration must be environment-specific and must
  not require committed secrets or local credential files.

## 7. Compatibility Requirements

- CR-001: Existing frontends must continue using compatible HTTP behavior unless
  a later confirmed TASK changes an API and updates affected frontend clients.
- CR-002: Existing protobuf definitions and generated code in both
  `logic-grpc-service` and `web-gin-service` must stay synchronized for any
  proto transition period.
- CR-003: Existing database migrations and `db.sql` baseline must remain aligned
  whenever schema changes occur.
- CR-004: Current CLI migration flags in `logic-grpc-service` must remain usable
  until a confirmed operational migration TASK replaces them.
- CR-005: Existing local development scripts should remain usable, or any
  replacement must provide equivalent documented local workflows.
- CR-006: Current RBAC permission semantics and route protection must remain
  compatible during Identity extraction.
- CR-007: Existing background workflows for notification, resume parsing,
  email, embedding, and agent runs must keep compatible event semantics during
  migration.
- CR-008: Existing tests under `logic-grpc-service` and `web-gin-service` must
  remain valid or be intentionally updated within the relevant TASK scope.

## 8. Observability and Debug Requirements

- ODR-001: The feature must define standard service telemetry for HTTP, gRPC,
  database, Redis, RabbitMQ, OSS, SMTP, AI provider, embedding provider, and
  worker execution paths.
- ODR-002: Each service must include request id or trace id in logs and must
  propagate that identifier across gateway, gRPC, event, and worker boundaries.
- ODR-003: Business events must include safe correlation identifiers such as
  event id, aggregate type, aggregate id, actor id where allowed, and request id
  where available.
- ODR-004: Outbox and Inbox must expose pending, retrying, failed, dead-letter,
  and publish/consume latency indicators.
- ODR-005: AI-related services must expose provider latency, timeout, retry,
  fallback, circuit breaker, budget, and quota indicators.
- ODR-006: Readiness and liveness failures must be diagnosable from logs and
  metrics without exposing secrets or candidate-sensitive payloads.
- ODR-007: Migration and cutover TASK reports must include validation evidence,
  observed metrics, rollback status, and known residual risks.

## 9. Error Handling and Fallback Requirements

- EFR-001: Cross-service calls must use timeouts and must classify retryable
  versus non-retryable failures.
- EFR-002: Write operations must not be retried unless idempotency is explicit.
- EFR-003: Event consumers must be safe under duplicate delivery, delayed
  delivery, out-of-order arrival where possible, and poison message handling.
- EFR-004: Queue or broker outage must not silently lose committed domain
  events; events must remain recoverable from durable storage.
- EFR-005: Redis outage must degrade cache, token-version cache, quota, or rate
  limiting behavior according to documented service-specific rules.
- EFR-006: AI provider, embedding provider, OSS, and SMTP outages must have
  explicit user-safe error messages, retry policy, fallback policy, and
  observability.
- EFR-007: Cutover failures must have rollback steps that restore traffic to
  the previous implementation and document any required data reconciliation.

## 10. Security and Safety Requirements

- SSR-001: No secrets, tokens, private keys, live credentials, or production-like
  sensitive values may be committed as part of this migration.
- SSR-002: Secret removal and rotation must be treated as an early safety
  requirement before production readiness.
- SSR-003: Internal service authentication must be required in production
  environments.
- SSR-004: The first production-hardening target is internal TLS plus required
  `GRPC_INTERNAL_TOKEN` validation. mTLS must be evaluated before final
  production readiness, but this SPEC does not require introducing a service
  mesh.
- SSR-005: RBAC, data scopes, token invalidation, and authorization audit
  behavior must remain correct across service boundaries.
- SSR-006: Services must enforce ownership checks for candidate, HR, and
  interviewer data instead of trusting only gateway checks.
- SSR-007: Logs, metrics, traces, events, and dead-letter payloads must not
  expose raw secrets or unnecessary personal data.
- SSR-008: Changes to authentication, authorization, security policy, public
  API behavior, database schema, dependencies, global config, or CI/CD require
  human confirmation before implementation.

## 11. Acceptance Criteria

- AC-001: SPEC, SDD, TASKS, acceptance files, scope rules, and scripts exist for
  this feature before implementation starts.
- AC-002: A migration safety net exists before service extraction begins,
  including core workflow tests, contract checks, scope checks, and baseline
  observability checks.
- AC-003: Current backend behavior remains compatible for registration/login,
  job management, candidate application, interview management, offer lifecycle,
  notification, and AI workflows during migration.
- AC-004: DDD module boundaries are documented and enforced inside the current
  backend before physical service extraction starts.
- AC-005: The codebase has explicit bounded-context ownership for Identity,
  Recruitment, Interview, Offer, Notification, AI Agent, Analytics, and
  platform/worker concerns.
- AC-006: At final readiness, gateway code contains no core domain state
  machines or persistence rules.
- AC-007: At final readiness, business services do not directly import or call
  repositories owned by another bounded context, except documented and approved
  transitional adapters.
- AC-008: At final readiness, cross-domain writes use domain events,
  Outbox/Inbox, idempotent consumers, or approved Saga/compensation flows.
- AC-009: At final readiness, each extracted service has clear build, run,
  health, config, logging, metrics, tracing, and deployment behavior.
- AC-010: At final readiness, data ownership is documented per service, and any
  remaining shared database tables have approved exception records and removal
  plans.
- AC-011: Notification and AI Agent are extractable or extracted first through
  shadow, dual-run, or equivalent low-risk cutover patterns.
- AC-012: Identity extraction preserves login, token refresh, token invalidation,
  permissions, scopes, and audit behavior.
- AC-013: Recruitment, Interview, and Offer extraction preserves all current
  lifecycle transitions and user-visible state.
- AC-014: Analytics reads are served from owned event-projection read models
  and do not call transactional service read APIs as a transition strategy or
  mutate transactional domain state.
- AC-015: All asynchronous consumers are idempotent and expose retry/dead-letter
  evidence.
- AC-016: Load tests pass the confirmed initial targets: 200 QPS for gateway
  normal APIs, 50 QPS for core writes, 10-20 concurrent AI/Embedding tasks,
  ordinary API P95 below 300 ms, complex query P95 below 1 s, and no AI
  workload blocking of main transaction paths.
- AC-017: HA validation proves multiple gateway and service replicas can run,
  roll, and recover without data corruption for core flows.
- AC-018: Documented simulations cover Redis, RabbitMQ, MySQL, OSS, SMTP, and
  AI/embedding provider degradation paths. Selecting a realistic load-test or
  failure-drill deployment environment is deferred outside this SPEC draft.
- AC-019: Security checks prove no committed live secrets, production gRPC
  internal auth is required, and sensitive logs/events are redacted.
- AC-020: `go test ./...` succeeds in both backend service workspaces, or any
  skipped/inapplicable tests are documented with approved reasons.
- AC-021: TASK reports and evidence prove scope compliance, SPEC/SDD alignment,
  acceptance alignment, validation command results, and knowledge impact.
- AC-022: Final architecture documentation, runbooks, deployment manifests, and
  `.knowledge` entries are updated or explicitly marked with tracked debt.

## 12. Out of Scope

- Rebuilding frontend applications into a new frontend architecture.
- Replacing Go, Gin, gRPC, MySQL, Redis, RabbitMQ, or Kubernetes without a later
  approved architecture decision.
- Introducing a service mesh, Kafka, distributed transaction coordinator, or
  new API gateway product as a hidden dependency of this SPEC.
- Product feature redesign, UI redesign, or workflow semantics changes.
- Migrating historical production data outside of schema ownership and
  cutover tasks approved under this feature.
- Defining exact production SLO numbers without user confirmation.

## 13. Assumptions Requiring Confirmation

The following decisions were confirmed by the user after the initial draft:

- D-001: The final service boundary includes API gateway, Identity,
  Recruitment, Interview, Offer, Notification, AI Agent, Analytics, and worker
  deployment units.
- D-002: The repository remains a monorepo while services become independently
  buildable and deployable.
- D-003: Go, Gin, gRPC, MySQL, Redis, RabbitMQ, and Kubernetes remain the
  baseline stack for this migration.
- D-004: Physical database splitting happens after code and service boundaries
  are proven. Interim shared database use is acceptable only with explicit
  ownership rules and removal plans.
- D-005: Existing frontend API behavior is preserved unless a later TASK
  explicitly scopes a coordinated frontend/backend contract change.
- D-006: Notification and AI Agent are the first extraction candidates before
  core Recruitment, Interview, and Offer service extraction.
- D-007: Candidate/resume/application-related profile data belongs to
  Recruitment. AI-generated profiles, matching artifacts, embeddings, memory,
  and AI-derived intelligence belong to AI Agent and synchronize through events
  or service APIs.
- D-008: During extraction, this feature does not require an interim
  `logic-grpc-service` facade. Gateway can call extracted services directly
  when a TASK explicitly performs that routing change.
- D-009: Initial performance targets are gateway normal APIs at 200 QPS, core
  writes at 50 QPS, AI/Embedding at 10-20 concurrent tasks, ordinary API P95
  below 300 ms, complex query P95 below 1 s, and AI workloads kept off the main
  transaction path.
- D-010: Initial HA targets are 99.5% monthly availability, RTO 30 minutes, and
  RPO 5 minutes.
- D-011: Internal service security starts with internal TLS plus required
  `GRPC_INTERNAL_TOKEN`; mTLS is evaluated before final readiness, and service
  mesh adoption is not required by this SPEC.
- D-012: Observability target is Prometheus metrics, Grafana dashboards,
  OpenTelemetry traces, and structured logs.
- D-013: Retention defaults are successful Outbox/Inbox records for 30 days,
  failed/dead-letter records for 90 days, authorization audit logs for 180
  days, and desensitized AI traces for 30-90 days.
- D-014: Analytics is implemented directly to the final standard using
  domain-event projections/read models. Transitional service read APIs for
  Analytics are not part of this migration.

## 14. Open Questions

No open questions remain for the current SPEC/Harness preparation. The
production-like load-test and failure-drill environment is intentionally out of
scope for this draft and must be handled by a later confirmed feature or TASK if
needed.
