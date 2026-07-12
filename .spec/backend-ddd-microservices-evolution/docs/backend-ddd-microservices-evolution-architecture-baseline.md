# Backend DDD Microservices Evolution Architecture Baseline

Date: 2026-07-11
Feature: `backend-ddd-microservices-evolution`
Task: `TASK-BDME-001`

## 1. Purpose

This inventory captures the current backend architecture before DDD module
movement or service extraction. It is intentionally read-only with respect to
business code: the goal is to map the existing gateway, logic service, workers,
runtime dependencies, candidate bounded contexts, cross-domain access risks,
and startup side effects.

## 2. Current Runtime Shape

The backend currently runs as a Gin HTTP gateway plus one central Go gRPC logic
service:

- `web-gin-service` owns HTTP routing, middleware, cookie/JWT handling,
  route-level RBAC, body limits, rate limits, quota checks, risk blocking,
  health endpoints, Swagger, SSE transport, and gRPC client calls.
  Source anchors: `web-gin-service/router/router.go:27`,
  `web-gin-service/router/router.go:81`, `web-gin-service/router/router.go:130`,
  `web-gin-service/router/router.go:150`.
- `logic-grpc-service` owns MySQL connection setup, migrations, repository
  construction, service construction, RabbitMQ, Redis-backed caches, OSS, SMTP,
  AI client initialization, worker startup, gRPC service registration, and
  health checks. Source anchors: `logic-grpc-service/main.go:83`,
  `logic-grpc-service/main.go:98`, `logic-grpc-service/main.go:242`,
  `logic-grpc-service/main.go:311`, `logic-grpc-service/main.go:368`.
- The gateway connects to the logic service through one gRPC connection and
  exposes generated clients for all current backend service contracts.
  Source anchors: `web-gin-service/rpc/client.go:47`,
  `web-gin-service/rpc/client.go:75`, `web-gin-service/rpc/client.go:129`.
- The same logic service binary can also run workers with `--worker-only`;
  Kubernetes includes a separate `logic-worker` deployment using that flag.
  Source anchors: `logic-grpc-service/main.go:42`,
  `logic-grpc-service/main.go:337`, `deploy/k8s/worker-deployment.yaml:20`.

## 3. Runtime Dependencies

| Dependency | Current owner and usage | Source anchors | Notes |
|---|---|---|---|
| MySQL | `logic-grpc-service` opens the GORM connection, sets pool parameters, runs migrations, and constructs all repositories. | `logic-grpc-service/main.go:83`, `logic-grpc-service/main.go:97`, `logic-grpc-service/main.go:132` | Hard dependency for logic readiness. |
| Redis | Logic uses Redis for health, OSS presign cache, notification cache, job cache, and token-version sync. Gateway uses Redis for token-version validation, rate limits, AI quotas, resume quotas, risk blocks, and readiness. | `logic-grpc-service/main.go:193`, `logic-grpc-service/main.go:205`, `logic-grpc-service/service/admin_service.go:72`, `web-gin-service/router/router.go:84`, `web-gin-service/router/router.go:141`, `web-gin-service/middleware/jwt.go:155` | Redis is hard for configured health in logic; some gateway middleware has resilient behavior but auth revocation semantics can degrade. |
| RabbitMQ | Logic creates the MQ connection, declares notification/resume/email/embedding/agent-run queues, starts consumers, and runs keepalive. | `logic-grpc-service/main.go:155`, `logic-grpc-service/main.go:314`, `logic-grpc-service/mq/rabbitmq.go:155`, `logic-grpc-service/mq/consumer.go:14` | Logic health treats RabbitMQ as a soft dependency unless the connection is present and closed. |
| Outbox | Domain services write `event_outbox`; `OutboxPublisher` polls pending rows and publishes to RabbitMQ. | `logic-grpc-service/service/outbox_publisher.go:30`, `logic-grpc-service/service/outbox_publisher.go:58`, `logic-grpc-service/service/outbox_publisher.go:116`, `logic-grpc-service/model/model.go:594` | Important bridge for event-driven decoupling; envelope is not yet a full domain event standard. |
| OSS/object storage | Logic initializes `oss.Storage`; candidate resume flows and AI tooling use it for presign, verification, download, and analysis inputs. | `logic-grpc-service/main.go:177`, `logic-grpc-service/service/candidate_service.go:31`, `logic-grpc-service/service/resume_parse_consumer.go:23`, `logic-grpc-service/service/services.go:93` | Candidate resume and AI resume intelligence depend on this boundary. |
| SMTP/email | Logic initializes sender/renderer and starts `EmailConsumer` for email outbox events. | `logic-grpc-service/main.go:224`, `logic-grpc-service/service/email_consumer.go:40`, `logic-grpc-service/service/email_consumer.go:64` | SMTP can be configured as required; email delivery should remain asynchronous. |
| AI provider | Logic initializes an AI client from DB model config or environment config; runtime services consume it. | `logic-grpc-service/main.go:217`, `logic-grpc-service/main.go:470`, `logic-grpc-service/service/services.go:94`, `logic-grpc-service/service/ai_service.go:102` | External latency and provider failure make AI Agent an extraction candidate. |
| Embedding provider | Logic owns provider/model config, embedding provider factory, embedding service, backfill service, and queue consumer. | `logic-grpc-service/service/services.go:87`, `logic-grpc-service/service/services.go:147`, `logic-grpc-service/service/embedding_event_consumer.go:25` | Fallback behavior is explicit and must stay observable. |
| MCP external tools | Logic owns server registration, tool listing, policy evaluation, calls, and audit logs. | `logic-grpc-service/service/mcp_service.go:32`, `logic-grpc-service/service/mcp_policy.go:1`, `logic-grpc-service/service/mcp_log_api.go:1` | Security-sensitive AI Agent subdomain. |
| HTTP clients/frontends | Three frontends call the gateway; gateway route behavior must remain stable during migration. | `web-gin-service/router/router.go:158`, `web-gin-service/router/router.go:191`, `web-gin-service/router/router.go:219` | Frontend changes are out of scope for this task. |

## 4. Gateway Surface Inventory

The gateway groups routes by public, candidate, staff/HR, and admin scopes.
It applies a shared `/api/v1` group with general rate limiting and then adds
JWT/current-principal/RBAC middleware on protected subgroups.

Important route clusters:

- Public/auth: register, invite validation, login, logout, refresh, current
  principal, public job list/detail. Source anchors:
  `web-gin-service/router/router.go:158`, `web-gin-service/router/router.go:165`.
- Candidate: profile, resume presign/confirm, applications, interviews,
  offers, notifications, candidate AI sessions/chat. Source anchors:
  `web-gin-service/router/router.go:191`, `web-gin-service/router/router.go:195`,
  `web-gin-service/router/router.go:204`, `web-gin-service/router/router.go:210`.
- Staff/HR recruitment: jobs, application status, resume intelligence,
  interviews, offers, AI, notifications, dashboard/analytics, collaboration.
  Source anchors: `web-gin-service/router/router.go:222`,
  `web-gin-service/router/router.go:229`, `web-gin-service/router/router.go:239`,
  `web-gin-service/router/router.go:253`, `web-gin-service/router/router.go:262`,
  `web-gin-service/router/router.go:298`, `web-gin-service/router/router.go:318`.
- Admin/configuration: invite codes, departments, locations, usage logs, RBAC,
  staff users, LLM/model/prompt/agent/MCP/SKILL/Agent Skill/embedding admin.
  Source anchors: `web-gin-service/router/router.go:342`,
  `web-gin-service/router/router.go:377`, `web-gin-service/router/router.go:388`,
  `web-gin-service/router/router.go:417`, `web-gin-service/router/router.go:442`,
  `web-gin-service/router/router.go:455`.

Gateway policy dependencies include request IDs, access logs, recovery,
security headers, CSP/CORS, timeouts, max body sizes, resilient rate limits,
AI daily quota, resume quota, and risk block checks. Source anchors:
`web-gin-service/router/router.go:81`, `web-gin-service/router/router.go:130`,
`web-gin-service/router/router.go:141`, `web-gin-service/router/router.go:151`.

## 5. Logic Service Inventory

The logic service currently registers all protobuf service implementations on
one `server.Server`, backed by one `service.Services` aggregate:

- Registered gRPC contracts: Auth, Job, Candidate, Application, AI,
  Notification, Interview, Offer, Admin, Collaboration, LLM Config, Prompt,
  Agent Config, MCP, Skill, Agent Skill, Recruiting Intelligence, Embedding
  Config, and gRPC Health. Source anchors: `logic-grpc-service/main.go:380`,
  `logic-grpc-service/main.go:397`.
- `server.Server` embeds all generated unimplemented service interfaces and
  forwards calls to the aggregate service fields. Source anchors:
  `logic-grpc-service/server/server.go:14`, `logic-grpc-service/server/server.go:41`.
- `service.Services` aggregates user-facing domain services, AI/configuration
  services, analytics, and background workers in one struct. Source anchors:
  `logic-grpc-service/service/services.go:29`, `logic-grpc-service/service/services.go:70`.

Key business and infrastructure areas:

| Current area | Main services | Main repositories/models | Current responsibility |
|---|---|---|---|
| Identity and Access | `AuthService`, `AdminService`, `ServiceAuthorizer`, `scopeEvaluator` | `UserRepo`, `RefreshTokenRepo`, `AuthzRepo`, `InviteCodeRepo`; `User`, `RefreshToken`, `Role`, `Permission`, `UserRole`, `UserDataScope`, `AuthorizationAuditLog` | Registration, login, refresh, current principal, RBAC, role assignment, data scopes, token invalidation, audit. |
| Recruitment jobs | `JobService`, `JobTaxonomyService` | `JobRepo`, `DepartmentRepo`, `JobLocationRepo`, `DepartmentLocationRepo`; `Job`, `Department`, `JobLocation`, `DepartmentLocation` | Job publishing, list/detail, taxonomy, scope checks, job cache. |
| Candidate/recruitment profile/resume/application | `CandidateService`, `ApplicationService` | `ProfileRepo`, `ResumeRepo`, `ApplicationRepo`; `CandidateProfile`, `Resume`, `Application`, `ApplicationStatusTransition` | Candidate profile/resume, apply, application list/status transitions, outbox side effects. |
| Interview | `InterviewService` | `InterviewRepo`, `ApplicationRepo`, `UserRepo`; `InterviewSchedule`, `InterviewFeedback` | Scheduling, update/cancel, feedback, interviewer/candidate lists, application status side effects. |
| Offer | `OfferService` | `OfferRepo`, `ApplicationRepo`; `Offer`, `OfferEvent` | Offer creation/update/send/withdraw/accept/reject, offer events, application status side effects. |
| Notification/email | `NotificationService`, `NotificationConsumer`, `EmailConsumer`, `OutboxPublisher` | `NotificationRepo`, `OutboxRepo`, `EmailLogRepo`; `Notification`, `EventOutbox`, `EmailLog` | Notification reads/unread/mark-read/SSE cache, async notification creation, email delivery. |
| AI Agent and candidate AI | `AIService`, `CandidateAIService`, `AgentRunConsumer`, `AgentContextBuilder` | `ChatRepo`, `SessionSummaryRepo`, `ToolTraceRepo`, `AgentRunRepo`, `AgentRunEventRepo`, `MemoryRepo`; AI chat/run/memory/trace models | HR/candidate chat, durable agent runs, tool execution, memory/context, outbox-backed run dispatch. |
| AI configuration and governance | `LlmConfigService`, `PromptService`, `AgentConfigService`, `MCPService`, `SkillService`, `AgentSkillService`, `EmbeddingConfigService`, `EmbeddingService` | provider/model/prompt/agent/MCP/SKILL/embedding repositories and models | Runtime configuration, prompt versions, capability binding, MCP policy/audit, skill registry, Agent Skill ranking, embedding config. |
| Resume intelligence and matching | `ResumeParseConsumer`, `ResumeProfileService`, `CandidateMatchService`, `RecruitingIntelligenceService` | `ResumeRepo`, `ResumeProfileRepo`, `CandidateMatchRepo`; resume profile and match models | Async text extraction, structured profile parse, candidate/job match evaluation, HR intelligence API. |
| Collaboration | `CollaborationService` | `CollaborationRepo` plus application/profile/job/user/interview/offer/resume repositories; collaboration models | Candidate workspace, notes, tags, follow-up tasks, timeline composition. |
| Analytics | `AnalyticsService`, `UsageStatsService` | `AnalyticsRepo`, `UsageStatsRepo`, `AuthzRepo`; reporting and usage models | Dashboard, funnel, time-in-stage, interview/offer metrics, auth audit queries, AI usage stats. |
| Platform/workers | `OutboxPublisher`, RabbitMQ consumers, migration runner, health server | `OutboxRepo`, MQ connection, migration files, `db.sql` | Migration, event dispatch, async processing, readiness/liveness. |

## 6. Bounded-Context Candidate Mapping

| Target context | Current packages and files | Current tables/models | Current gateway handlers/routes |
|---|---|---|---|
| API Gateway | `web-gin-service/router/**`, `handler/**`, `middleware/**`, `rpc/**`, `pkg/authz/**` | No owned domain tables | All `/api/v1` routes; transport, cookie, RBAC, limits, SSE, gateway-to-gRPC calls. |
| Identity and Access | `logic-grpc-service/service/auth_service.go`, `admin_service.go`, `authorizer.go`, `audit.go`; `repository/user_repo.go`, `refresh_token_repo.go`, `authz_repo.go`; `pkg/authz/**`, `pkg/jwt/**` | `users`, `refresh_tokens`, `roles`, `permissions`, `role_permissions`, `user_roles`, `user_data_scopes`, `authorization_audit_logs`, `invite_codes` | `handler/auth.go`, `handler/hr/admin.go`; `/auth/**`, `/hr/admin/roles`, `/permissions`, `/users/**`, `/admin/auth-audit-logs`. |
| Recruitment | `job_service.go`, `job_taxonomy_service.go`, `candidate_service.go`, `application_service.go`, `transition_validator.go`; job/profile/resume/application repositories | `jobs`, `departments`, `job_locations`, `department_locations`, `candidate_profiles`, `resumes`, `applications`, `application_status_transitions` | public jobs, candidate profile/resume/applications, HR jobs/applications/resume intelligence. |
| Interview | `interview_service.go`; `repository/interview_repo.go` | `interview_schedules`, `interview_feedback` | `handler/hr/interview.go`, `handler/candidate/interview.go`; HR and candidate/interviewer interview routes. |
| Offer | `offer_service.go`; `repository/offer_repo.go` | `offers`, `offer_events` | `handler/hr/offer.go`, `handler/candidate/offer.go`; HR/candidate offer routes. |
| Notification | `notification_service.go`, `notification_consumer.go`, `notification_worker.go`, `outbox_publisher.go`, `email_consumer.go`; notification/outbox/email repositories | `notifications`, `event_outbox`, `email_logs` | `handler/notification.go`; notification list/summary/mark-read/SSE routes for candidate and staff. |
| AI Agent | `ai_service.go`, `candidate_ai_service.go`, `agent_*`, `embedding_*`, `mcp_*`, `skill_*`, `prompt_service.go`, `llm_config_service.go`, `resume_profile_*`, `candidate_match_*`, `recruiting_intelligence_service.go`; `ai/**`, `resumeparser/**`, `oss/**` | `ai_chat_sessions`, `ai_chat_history`, `ai_session_summaries`, `ai_tool_traces`, `agent_runs`, `agent_run_events`, `agent_run_steps`, `ai_memories`, `ai_embeddings`, `llm_*`, `embedding_*`, `prompt_*`, `agent_*`, `mcp_*`, `ai_skill*`, `agent_skill*`, `resume_profile*`, `candidate_match*` | HR/candidate AI routes, admin AI config routes, HR recruiting intelligence routes. |
| Analytics | `analytics_service.go`, `usage_stats_service.go`; `repository/analytics_repo.go`, `usage_stats_repo.go` | Reads recruitment/notification/status tables today; writes/reads usage stats and audit-derived metrics | `handler/hr/analytics.go`, dashboard and usage stats routes. |
| Platform/workers | `main.go`, `migration/**`, `mq/**`, `server/health.go`, worker consumers, deployment manifests | `event_outbox`, migration metadata, queue-backed worker state | Health/ready endpoints, Kubernetes deployments/services, Docker Compose infrastructure. |

## 7. Cross-Domain Access and Coupling Risks

The following are current-code observations, not defects introduced by this
task. They are the main migration risks to resolve in later TASKs.

| Risk | Current source evidence | Migration note |
|---|---|---|
| Central service aggregate hides domain ownership. | `service.Services` contains every domain, config service, and worker in one struct at `logic-grpc-service/service/services.go:29`; `NewServices` constructs many repositories from the same DB at `logic-grpc-service/service/services.go:83`. | Later DDD module skeletons should isolate domain/application/infrastructure/interface packages before extraction. |
| Recruitment, Interview, Offer, Notification, and Analytics share repositories and transactional state. | `ApplicationService` owns applications but directly uses interview and notification repositories at `logic-grpc-service/service/application_service.go:22`; `InterviewService` uses `ApplicationRepo` and `NotificationRepo` at `logic-grpc-service/service/interview_service.go:19`; `OfferService` uses `ApplicationRepo` and `NotificationRepo` at `logic-grpc-service/service/offer_service.go:20`. | Cross-domain writes should move toward domain events, Outbox/Inbox, and idempotent consumers. |
| Interview and Offer mutate application status inside their own transactions. | Interview status updates appear around `logic-grpc-service/service/interview_service.go:250`, `logic-grpc-service/service/interview_service.go:579`, `logic-grpc-service/service/interview_service.go:1085`; offer status updates appear around `logic-grpc-service/service/offer_service.go:185` and later send/decision flows. | This is a high-value target for Saga/process-manager or event-driven decoupling. |
| Collaboration composes many domain repositories directly. | `NewServices` constructs `CollaborationService` with applications, profiles, jobs, users, interviews, offers, and resumes at `logic-grpc-service/service/services.go:201`. | Keep as a read/composition boundary or convert to projections before service extraction. |
| Analytics reads transactional state rather than final event projections. | `AnalyticsService` combines analytics repository and authz repository at `logic-grpc-service/service/analytics_service.go:19`; current SPEC requires final Analytics to use event projections. | Later Analytics TASKs should avoid transitional service read APIs. |
| AI Agent reaches into recruitment data for tools, analysis, resume intelligence, and candidate matching. | `NewAIService` receives application/job/resume repositories and tool executor dependencies at `logic-grpc-service/service/ai_service.go:102`; `RecruitingIntelligenceService` composes application/job/resume/profile/match services at `logic-grpc-service/service/services.go:178`. | AI-generated artifacts should become AI-owned and synchronize with recruitment through events or APIs. |
| Identity is used both at gateway policy level and logic-side authorization level. | Gateway RBAC is declared in `web-gin-service/router/router.go`; logic service uses `ServiceAuthorizer` and `AuthzRepo` across many services, for example `logic-grpc-service/service/analytics_service.go:25` and `logic-grpc-service/service/job_service.go:20`. | Identity extraction must preserve route RBAC, service-side authorization, token invalidation, data scopes, and audit. |
| Generated protobuf contract is a shared public surface. | Both services use generated clients/servers from `recruitment/pb`; gateway client construction is in `web-gin-service/rpc/client.go:47`; logic registration is in `logic-grpc-service/main.go:380`. | Proto changes require synchronized source/generated code and human confirmation. |
| Worker and request-serving runtime share one binary and service aggregate. | Background workers start from `main.go:314`; worker-only mode runs the same binary at `main.go:337`; Kubernetes `logic-worker` uses the logic image. | Worker decomposition can be staged before full service extraction. |

## 8. Startup-Time Operational Side Effects

Startup currently does more than serving requests:

- Loads config, validates internal gRPC token, and initializes logger before
  dependencies. Source anchors: `logic-grpc-service/main.go:70`,
  `logic-grpc-service/main.go:76`.
- Opens MySQL and applies pending migrations automatically on startup.
  Source anchors: `logic-grpc-service/main.go:83`,
  `logic-grpc-service/main.go:127`.
- Supports migration CLI flags for baseline/status/down before service start.
  Source anchors: `logic-grpc-service/main.go:42`,
  `logic-grpc-service/main.go:103`.
- Initializes RabbitMQ and continues with a reconnect-capable worker posture
  when it is unavailable. Source anchors: `logic-grpc-service/main.go:155`,
  `logic-grpc-service/main.go:170`.
- Initializes OSS, Redis caches, AI client, SMTP sender, and email renderer.
  Source anchors: `logic-grpc-service/main.go:177`,
  `logic-grpc-service/main.go:193`, `logic-grpc-service/main.go:217`,
  `logic-grpc-service/main.go:224`.
- Seeds default prompt templates and agent configurations from hardcoded
  defaults. Source anchors: `logic-grpc-service/main.go:256`,
  `logic-grpc-service/main.go:264`.
- Optionally bootstraps an initial admin from `INITIAL_ADMIN_USERNAME` and
  performs legacy RBAC migration for users without role rows. Source anchors:
  `logic-grpc-service/main.go:271`, `logic-grpc-service/main.go:303`.
- Starts outbox publisher and all consumers in the same process unless
  disabled by environment. Source anchors: `logic-grpc-service/main.go:314`,
  `logic-grpc-service/main.go:337`.

Migration risk: multi-replica production rollouts can amplify startup side
effects. Later TASKs should separate migration/seed/admin bootstrap controls
from ordinary request-serving startup and define idempotency/locking evidence.

## 9. Current Deployment and HA Baseline

- Docker Compose provides MySQL, Redis, RabbitMQ, logic service, gateway, and
  frontend containers. Source anchors: `docker/docker-compose.yml:2`,
  `docker/docker-compose.yml:23`, `docker/docker-compose.yml:41`,
  `docker/docker-compose.yml:63`, `docker/docker-compose.yml:119`.
- Kubernetes manifests define two gateway replicas, two logic-service replicas,
  one logic-worker replica, services, config maps, secret examples, and a
  NetworkPolicy that limits logic gRPC ingress to gateway and worker pods.
  Source anchors: `deploy/k8s/web-deployment.yaml:9`,
  `deploy/k8s/logic-deployment.yaml:9`, `deploy/k8s/worker-deployment.yaml:9`,
  `deploy/k8s/network-policy.yaml:1`.
- Logic readiness checks MySQL and configured Redis; RabbitMQ is considered a
  soft dependency unless the connection is present but closed. Source anchor:
  `logic-grpc-service/server/health.go:21`.
- Gateway readiness calls logic gRPC health and Redis ping. Source anchors:
  `web-gin-service/router/router.go:84`, `web-gin-service/rpc/client.go:161`.

Residual risk: readiness proves process/dependency availability more than
semantic worker health, queue backlog, consumer lag, or domain-specific
degradation behavior.

## 10. Not-Found Notes

- No physical microservice directories for `identity-service`,
  `notification-service`, `ai-agent-service`, `analytics-service`, or core
  recruitment/interview/offer services were found. Current extraction state is
  still the gateway plus central logic service plus worker-only mode.
- No final Analytics event-projection service was found. Current analytics is
  still implemented inside `logic-grpc-service`.
- No Inbox/idempotency storage table dedicated to consumer idempotency was
  found in the model/table inventory. Current idempotency is partial and
  flow-specific.
- No production-grade tracing/metrics implementation was identified in this
  TASK's source scan beyond structured logs, request IDs, health probes, and
  selected runtime debug metadata. Later observability TASKs should confirm and
  extend this.

## 11. Knowledge Impact Review

Reviewed active knowledge documents routed by `.knowledge/manifest.yaml` and
`task-scope.json` for this inventory. No active knowledge file was changed by
this TASK. The reviewed documents are consistent with the source-level
baseline: gateway owns transport/policy, logic owns business/persistence/AI,
current service boundaries are broad, and later migration must preserve
contracts, RBAC, schema alignment, notification/outbox behavior, and sensitive
data handling.

Result: `none`.
Coverage gap: `false` for this inventory task, because the missing future
service boundaries are explicitly planned by the active SPEC/SDD/TASKS.
