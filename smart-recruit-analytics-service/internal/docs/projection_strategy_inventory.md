# Analytics Projection Strategy Inventory

TASK-019 established the Analytics DDD skeleton and reporting/projection
contract. TASK-020 localized the reporting/projection application layer.
TASK-021 switches runtime reporting to the service-local implementation while
preserving protobuf contracts, database schema, and reporting API behavior.

## Current Runtime Boundary

- Service root: `smart-recruit-analytics-service`.
- Active runtime entrypoint: `cmd/analytics-service/main.go`.
- Runtime registration: `internal/runtime/runtime.go` registers the
  Analytics-owned AdminService reporting subset:
  - `GetDashboardReport`
  - `GetFunnelReport`
  - `GetTimeInStageReport`
  - `GetInterviewOfferMetrics`
- Current implementation source: service-local `internal/application/service`
  reporting orchestration, `internal/infrastructure/persistence` GORM reporting
  queries, and `internal/interfaces/grpc` proto mapping.
- Remaining compatibility debt: Identity-owned permission/data-scope reads use
  service-local SQL in `internal/infrastructure/client.AuthzAdapter` until
  Identity exposes a service query port or signed scope snapshot contract.
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

## Local Reporting Application Boundary

TASK-020 localizes reporting/projection application contracts under
`internal/application` and `internal/domain` without switching runtime wiring.

- `application/service.ReportingService` owns report query orchestration,
  permission checks, scope lookup, fail-soft dashboard secondary reads, and
  legacy-compatible result shaping.
- `domain/policy` owns funnel ordering, stage labels, conversion rates,
  time-in-stage hour conversion, interview pass rate, offer acceptance rate,
  actor mismatch rules, and projection strategy validation.
- `domain/repository.ReportingRepository` is the local read port for dashboard,
  funnel, time-in-stage, interview, offer, trend, stage distribution, and unread
  notification metrics.
- `application/service.ProjectionService` writes only Analytics projection
  events/checkpoints through `domain/repository.ProjectionStore`.

The local application boundary does not write Recruitment, Interview, Offer,
Notification, Identity, or AI Agent transactional state. Any future adapter that
needs writes outside Analytics-owned projection tables is a Hard Stop.

## Transitional Read Debt

Current reporting still reads source-domain tables through service-local SQL
queries. This is allowed only as migration debt until projection-backed read
models are fully implemented.

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

TASK-021 does not create new event schemas, projection tables, or protobuf APIs.

## Existing Test Anchors

- `cmd/analytics-service/main_test.go` validates discovery/config fallback.
- `internal/runtime/runtime_test.go` validates reporting runtime registration
  and projection descriptor behavior.
- `internal/application/service/reporting_service_test.go` covers local
  reporting/projection application orchestration.
- `internal/interfaces/grpc/reporting_api_test.go` covers AdminService reporting
  response compatibility.
- `internal/infrastructure/projection/gorm_store_test.go` covers local
  projection event/checkpoint persistence.

## Migration Notes For Later TASKs

- Keep dashboard/funnel/time-in-stage/interview-offer report semantics stable.
- Keep permission and data-scope filtering aligned with Identity-owned RBAC.
- Replace transitional source-domain reads with projection-backed read models
  only after replay/backfill behavior is covered by tests.
- Replace local RBAC compatibility reads with Identity-owned query ports when
  that cross-service contract is available.
- Do not add schema, protobuf, or new snapshot APIs without a separate scoped
  confirmation.
