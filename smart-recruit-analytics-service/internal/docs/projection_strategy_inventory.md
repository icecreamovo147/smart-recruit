# Analytics Projection Strategy Inventory

TASK-019 establishes the Analytics DDD skeleton and records the current
reporting/projection contract. It does not change runtime wiring, protobuf
contracts, database schema, reporting API behavior, or shared implementation
paths.

## Current Runtime Boundary

- Service root: `smart-recruit-analytics-service`.
- Active runtime entrypoint: `cmd/analytics-service/main.go`.
- Runtime registration: `internal/runtime/runtime.go` registers the
  Analytics-owned AdminService reporting subset:
  - `GetDashboardReport`
  - `GetFunnelReport`
  - `GetTimeInStageReport`
  - `GetInterviewOfferMetrics`
- Current implementation source: shared `smart-recruit-domain-go/service`
  `AnalyticsService` and shared `repository.AnalyticsRepo`.
- Runtime platform behavior retained: Nacos discovery/config, gRPC internal
  auth, optional TLS, health, metrics, trace, MySQL, optional Redis, and
  structured logging.

## Reporting API Inventory

| API | Current responsibility | Current data source |
| --- | --- | --- |
| `GetDashboardReport` | Dashboard KPI, stage distribution, trends, unread staff notifications. | Transitional read queries over jobs, applications, interviews, offers, notifications, RBAC/data scopes. |
| `GetFunnelReport` | Funnel counts and conversion rates by scoped job/date filter. | Transitional read queries over applications and jobs. |
| `GetTimeInStageReport` | Average duration in pipeline stages. | Transitional read queries over application status transitions and applications. |
| `GetInterviewOfferMetrics` | Interview and offer metrics by scoped job/date filter. | Transitional read queries over interview schedules, offers, applications, and jobs. |

## Projection Ownership

Analytics owns read/write access for:

- `analytics_projection_events`
- `analytics_projection_checkpoints`

Projection target:

- Domain events should be stored idempotently by event id.
- Checkpoints should track replay/projection cursors by projection name.
- Projection writes must not mutate transactional business state owned by
  Recruitment, Interview, Offer, Identity, Notification, or AI Agent contexts.

## Transitional Read Debt

Current reporting still reads source-domain tables through shared SQL queries.
This is allowed only as migration debt until projection-backed read models are
fully implemented.

Known transitional read dependencies:

- Recruitment: `jobs`, `applications`, `application_status_transitions`.
- Interview: `interview_schedules`.
- Offer: offer state tables.
- Notification: notifications unread-count read.
- Identity: RBAC, permissions, data scopes for analytics access filtering.

These reads must remain read-only. Analytics must not write Recruitment,
Interview, Offer, Notification, Identity, or AI Agent transactional state.

## Projection Event Inputs

Future projection ingestion should consume stable domain-event envelopes for:

- Application created and status changed.
- Interview scheduled, completed, cancelled, or rescheduled.
- Offer created, sent, accepted, rejected, or withdrawn.
- Notification read-state changes only when dashboard unread counts move to an
  Analytics-owned read model.

TASK-019 does not create new event schemas, projection tables, or protobuf APIs.

## Existing Test Anchors

- `cmd/analytics-service/main_test.go` validates discovery/config fallback.
- `internal/runtime/runtime_test.go` validates reporting runtime registration
  and projection descriptor behavior.
- Shared tests currently cover `AnalyticsService`, `AnalyticsRepo`, and
  `AnalyticsProjectionRepo` behavior.

## Migration Notes For Later TASKs

- Keep dashboard/funnel/time-in-stage/interview-offer report semantics stable.
- Keep permission and data-scope filtering aligned with Identity-owned RBAC.
- Move query orchestration into Analytics local application services before
  switching runtime wiring.
- Introduce projection-backed reads only after local projection adapters and
  replay/backfill behavior are covered by tests.
- Do not add schema, protobuf, or new snapshot APIs without a separate scoped
  confirmation.
