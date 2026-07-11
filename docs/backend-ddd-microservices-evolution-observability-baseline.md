# Backend Observability Baseline Design

本文档记录 `backend-ddd-microservices-evolution` 的 TASK-BDME-006 观测性基线。它定义迁移期间需要保持一致的 telemetry contract，不改变当前运行时代码。

## Telemetry Scope

所有后端服务、worker、事件消费者和未来拆分服务应至少记录以下维度：

| Path | Required Telemetry | Required Labels / Fields |
| --- | --- | --- |
| HTTP gateway | request count, latency buckets, status code count, timeout count, panic/recovery count, rate-limit/quota/risk-block decisions | `service`, `route`, `method`, `status`, `request_id`, safe actor id/account type when available |
| gRPC client/server | method count, latency, status, deadline exceeded, retry count, internal-auth failures | `service`, `grpc_method`, `grpc_code`, `request_id`, peer service |
| Database/MySQL | pool stats, query errors, slow query count/latency, migration status, transaction retry/failure count | `service`, `db_role`, `operation`, `table` when safe, no raw SQL values |
| Redis | ping failures, read/write failures, cache hit/miss where useful, fallback count, token-version lookup failures | `service`, `operation`, `cache_name`, `fallback` |
| RabbitMQ | connection state, publish failures, consumer failures, retry count, dead-letter count, queue backlog, delivery age | `service`, `exchange`, `queue`, `routing_key`, `consumer`, `event_type` |
| Workers | active workers, handled jobs, failed jobs, processing latency, retry attempts, graceful shutdown state | `service`, `worker`, `job_type`, `event_type`, `attempt` |
| Outbox/Inbox | pending count, retrying count, failed/dead-letter count, max age, publish/consume latency, idempotent replay count | `aggregate_type`, `event_type`, `event_version`, `producer`, `consumer`, `request_id` |
| AI/Embedding | provider latency, timeout, retry, fallback, budget, circuit state, token/context usage, agent run status transition | `provider`, `model`, `agent_type`, `run_status`, `error_type`, no raw prompt or resume text |
| OSS/SMTP | operation latency/errors, retry/fallback count, provider outage classification | `provider`, `operation`, `error_type`, no object keys containing personal data unless hashed/redacted |
| Business workflow | application created, status transitioned, interview scheduled/cancelled/completed, offer sent/accepted/rejected, notification emitted | `domain`, `event_type`, `aggregate_id`, `request_id`, safe actor id |

## Request And Trace Propagation

The gateway remains the entry point for request identity.

- `RequestID` middleware must create or preserve a request id for every HTTP request.
- Gateway logs, auth audit entries, and gRPC calls must include `request_id`.
- gRPC metadata must carry request id and safe actor context into logic service calls.
- Domain events and Outbox records must carry request id, event id, event version, aggregate type, aggregate id, producer, occurred time, and actor id only when allowed.
- Workers and consumers must log and emit metrics with the event/request id they received.
- Future OpenTelemetry traces should map the existing request id to trace/span correlation rather than replacing it abruptly.

## Logging Contract

Logs must be structured and queryable.

Required fields:

- `service`
- `environment`
- `request_id` or `trace_id`
- `actor_user_id` or safe actor reference when available
- `route` or `grpc_method` where applicable
- `domain`, `aggregate_type`, `aggregate_id`, or `event_type` for domain/event logs
- `error_type`, `retryable`, and `attempt` for failures

Do not log:

- JWTs, refresh tokens, internal gRPC tokens, API keys, access-key secrets, database passwords, Redis passwords, RabbitMQ credentials, SMTP credentials, encryption keys
- raw prompts, raw chat transcripts, raw resume text, full candidate documents, raw provider payloads, or dead-letter payloads containing unnecessary personal data
- signed OSS URLs or object keys that expose personal identifiers

Allowed sensitive references:

- hashed token ids
- redacted provider/model names
- candidate/application/job ids when needed for debugging and permitted by data policy
- summarized AI error type, provider status, and token/cost counters

## Cutover Diagnostics

Every future cutover TASK must record:

- pre-change and post-change HTTP/gRPC success rate and latency
- queue backlog, retry, dead-letter, and max event age
- DB and Redis error/fallback deltas
- AI provider timeout/fallback/circuit metrics for AI-facing changes
- rollback readiness and last known rollback point
- residual risk and known blind spots

## Current Gaps

- Metrics and traces are not yet standardized in code; current coverage is mainly structured logs, request id propagation, health checks, and targeted tests.
- Outbox/Inbox metrics and event-age dashboards are design requirements for later implementation tasks.
- AI prompt/resume redaction should be validated by later log/event tests when telemetry sinks are introduced.
- Worker semantic health currently needs stronger dependency-aware checks in later readiness tasks.
