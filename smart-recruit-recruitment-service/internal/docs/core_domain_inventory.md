# Recruitment Core Domain Inventory

TASK-015 establishes the Recruitment DDD skeleton and records the current core
domain contract. TASK-029 later localizes the remaining legacy implementation
paths under `internal/legacydomain` without changing protobuf contracts,
database schema, or recruitment API behavior.

## Current Runtime Boundary

- Service root: `smart-recruit-recruitment-service`.
- Active runtime entrypoint: `cmd/recruitment-service/main.go`.
- Runtime registration: `internal/runtime/runtime.go` registers `JobService`,
  `AdminService` subsets, `CandidateService`, `ApplicationService`, and
  `CollaborationService`.
- Current implementation source after TASK-029: service-local
  `internal/legacydomain/service`, `repository`, `model`, and `ai`
  compatibility packages provide active job, taxonomy, candidate, resume,
  application, collaboration, usage-stats, authz/scope, AI helper, and outbox
  behavior. `smart-recruit-commons/oss` remains a shared infrastructure
  bridge/commons candidate.
- Platform behavior retained by runtime: Nacos discovery, gRPC internal auth,
  health, metrics, trace, MySQL, optional Redis, and OSS configuration.

## Service Surface Inventory

| Surface | Runtime provider | Current responsibility |
| --- | --- | --- |
| Job | local legacy `JobService` | Create/update/online/offline jobs, HR job listing, public listing, job detail, department/location validation, scope filtering. |
| Job taxonomy | local legacy `JobTaxonomyService` | Department, location, department-location configuration, job option queries. |
| Recruitment admin | local legacy `AdminService` subset | Invite-code and third-party usage log RPCs remain registered through Recruitment runtime, not Identity runtime. |
| Usage stats | local legacy `UsageStatsService` | Usage stats and trends guarded by service authorizer. |
| Candidate | local legacy `CandidateService` | Candidate profile, resume read/update, resume upload presign/confirm, OSS interactions, usage logs and outbox side effects. |
| Application | local legacy `ApplicationService` | Apply job, candidate application list, HR application list, status mutation, status transition listing, outbox/notification side effects. |
| Collaboration | local legacy `CollaborationService` | Candidate workspace aggregate, notes, tags, follow-up tasks, timeline, and cross-context interview/offer/resume reads. |

## Core Domain Areas

### Jobs

- Owner table: `jobs`.
- Key rules: department/location validation, online/offline transitions, HR owner
  scope, public visibility, list/filter/pagination compatibility.
- Current dependencies: `JobRepo`, `AuthzRepo`, `JobTaxonomyService`, optional
  job cache, scope evaluator.

### Candidate Profile And Resume

- Owner tables: `candidate_profiles`, `resumes`, `resume_parse_runs`,
  `resume_profiles`, `resume_educations`, `resume_experiences`,
  `resume_projects`, `resume_skills`.
- Key rules: actor match for candidate self-service, profile completeness,
  resume ownership, presigned upload, confirm upload, OSS object keys, usage log
  and event/outbox side effects.
- Transitional readers: AI Agent reads resume/profile tables for parsing,
  matching, and intelligence.

### Applications

- Owner tables: `applications`, `application_status_transitions`.
- Key rules: application round creation, complete profile/resume preconditions,
  online job precondition, status keys, allowed transitions, reason requirements,
  terminal states, current-round constraints, HR/candidate list visibility, and
  transition audit.
- Transitional readers: Interview, Offer, Analytics, and AI Agent read
  applications through declared access.

### Collaboration

- Owner tables: `candidate_notes`, `candidate_tags`,
  `candidate_tag_assignments`, `follow_up_tasks`.
- Key rules: staff scope checks, workspace aggregation, notes, tags, task
  status, timeline composition, sanitized resume/profile/job context.
- Transitional reads: interview schedules and offer state are read to compose
  workspace/timeline but remain owned by Interview and Offer services.

### Taxonomy

- Owner tables: `departments`, `job_locations`, `department_locations`.
- Key rules: active/inactive state, department-location relation validation,
  job option compatibility, admin mutation behavior.

### Usage Stats

- Active provider: shared `UsageStatsService`.
- Current storage: `third_party_usage_logs` remains platform-owned; Recruitment
  runtime exposes read APIs for usage stats/trends through the registered admin
  surface.
- Later migration must decide whether usage stats remains a platform read model
  or moves to an analytics/query context.

## Event And External Dependency Inventory

- OSS: resume upload/presign/confirm currently uses shared OSS storage.
- Outbox: candidate/application flows publish notifications and async events
  through platform-owned `event_outbox`.
- Notifications: application status changes and candidate actions can emit
  notification/outbox side effects; Notification service owns notification
  persistence.
- Authz/data scope: Recruitment depends on Identity-owned RBAC/data-scope state
  through shared `AuthzRepo` and scope evaluator today.
- Interview/Offer reads: Collaboration and lifecycle flows read interview/offer
  data as transitional cross-context reads.

## Table Ownership

Recruitment owns read/write access for:

- `jobs`
- `candidate_profiles`
- `resumes`
- `resume_parse_runs`
- `resume_profiles`
- `resume_educations`
- `resume_experiences`
- `resume_projects`
- `resume_skills`
- `applications`
- `application_status_transitions`
- `departments`
- `job_locations`
- `department_locations`
- `candidate_notes`
- `candidate_tags`
- `candidate_tag_assignments`
- `follow_up_tasks`

Declared non-owner reads relevant to Recruitment:

- `interview_schedules` from Interview.
- Offer state through Offer repositories/runtime dependency where collaboration
  or lifecycle views require it.
- Identity-owned RBAC/data-scope data through current authorizer/scope evaluator.

Platform interaction:

- `event_outbox` writes are allowed through the platform event contract.
- `event_inbox` read/write is declared for Recruitment for future consumers.
- `third_party_usage_logs` remains platform-owned.

## Existing Test Anchors

- `cmd/recruitment-service/main_test.go` verifies discovery and Nacos fallback.
- `internal/runtime/runtime_test.go` verifies service registration and required
  dependencies.
- Shared domain tests currently cover application lifecycle, job, collaboration,
  resume, outbox/inbox, RBAC/scope, and status behavior.

## Migration Notes For Later TASKs

- Do not change protobuf, public HTTP behavior, schema, status keys, candidate
  labels, HR labels, or permission/data-scope behavior in incidental migration
  work.
- Move application status rules into pure domain/application tests before
  switching runtime.
- Keep OSS, outbox, notification, Identity authz, Interview, Offer, AI Agent,
  and Analytics dependencies behind local application ports.
- Preserve current cross-surface workflow behavior for HR, candidate, and
  interviewer users.
