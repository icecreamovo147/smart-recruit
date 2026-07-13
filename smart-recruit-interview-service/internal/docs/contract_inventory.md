# Interview Service Contract Inventory

## 1. Purpose

This inventory tracks the Interview migration baseline for `.spec/microservice-ddd-evolution`.

It records the current public contract, runtime wiring, shared-domain dependencies, table access boundary, event side effects, and test coverage before Interview business behavior is moved into local DDD layers.

This document does not change protobuf, runtime registration, persistence behavior, schema, gateway routing, or user-visible behavior.

## 2. Current Local DDD Layers

The Interview service now has the target DDD package shape under `internal/`:

```text
internal/
  domain/
    model/
    repository/
    service/
    event/
  application/
    command/
    query/
    port/
    service/
  infrastructure/
    persistence/
    client/
    mq/
  interfaces/
    grpc/
    mapper/
  runtime/
```

TASK-006 only creates package boundaries and records the legacy contract baseline. Interview lifecycle rules remain in `smart-recruit-domain-go/service.InterviewService` until TASK-007 and TASK-008 migrate behavior and runtime wiring.

## 3. Protobuf and Runtime Contract

Current runtime registration:

- `smart-recruit-interview-service/internal/runtime.Runtime.RegisterGRPC`
- `pb.RegisterInterviewServiceServer`
- Service descriptor: `pb.InterviewService_ServiceDesc.ServiceName`

Current `pb.InterviewServiceServer` methods exposed by runtime:

- `ScheduleInterview`
- `UpdateInterview`
- `CancelInterview`
- `GetInterview`
- `ListInterviewers`
- `ListApplicationInterviews`
- `ListMyInterviews`
- `ListCandidateInterviews`
- `SubmitFeedback`
- `GetFeedback`
- `BatchCancelInterviews`

Compatibility rule:

- TASK-006 does not change protobuf request/response types, rpc names, route mode, gateway behavior, or runtime registration behavior.
- Existing messages, response codes, actor metadata validation, permission checks, and candidate-facing note filtering remain owned by the legacy implementation for this TASK.

## 4. Current Shared Dependency Baseline

Current direct imports in `cmd/interview-service/main.go`:

- `smart-recruit-domain-go/repository`
- `smart-recruit-domain-go/service`
- `smart-recruit-proto/recruitment/pb`
- `smart-recruit-platform-go/config`
- `smart-recruit-platform-go/logger`
- `smart-recruit-platform-go/nacos`
- `smart-recruit-platform-go/observability`
- `smart-recruit-platform-go/server`
- `smart-recruit-platform-go/serviceconfig`

Current direct import in runtime:

- `smart-recruit-proto/recruitment/pb`

Current local service construction:

- `buildDomainServices` constructs the shared `service.Services` graph and passes `services.Interview` into the Interview runtime.
- The active gRPC implementation is therefore still `smart-recruit-domain-go/service.InterviewService`.

Shared repositories constructed for the active runtime path:

- `repository.NewUserRepo`
- `repository.NewRefreshTokenRepo`
- `repository.NewJobRepo`
- `repository.NewProfileRepo`
- `repository.NewResumeRepo`
- `repository.NewApplicationRepo`
- `repository.NewInterviewRepo`
- `repository.NewOfferRepo`
- `repository.NewNotificationRepo`
- `repository.NewOutboxRepo`
- `repository.NewAuthzRepo`
- plus shared AI/chat/session/memory/invite/department/location/usage/email repositories required by `service.NewServices`.

Migration implication:

- TASK-007 should move Interview domain model, lifecycle policy, feedback rules, repository ports, command/query DTOs, and application ports into local packages.
- TASK-008 should replace active runtime wiring with a local `interfaces/grpc` adapter backed by local application services, and remove the active direct dependency on shared `service.InterviewService`.
- Remaining cross-context reads and lifecycle writes must be represented as explicit application ports or transitional infrastructure adapters.

## 5. Current Behavior Dependency Map

### Schedule interview

- Protobuf: `ScheduleInterviewRequest` and `ScheduleInterviewResponse`.
- Legacy method: `service.InterviewService.ScheduleInterview`.
- Key dependencies: actor metadata match, `interview.schedule` permission, application/job scope, `ApplicationRepo.GetDetail`, `JobRepo.GetByID`, `JobRepo.BelongsToHR`, `InterviewRepo.GetMaxRoundNo`, `InterviewRepo.CreateWithTx`, `RecruitmentLifecycleProcessManager.ApplyTransitionTx`, and outbox notification/email writes.
- Side effects: creates `interview_schedules`, may transition application status to `interview_pending`, writes notification/email outbox events for interviewer and candidate.

### Update interview

- Protobuf: `UpdateInterviewRequest` and `CommonResponse`.
- Legacy method: `service.InterviewService.UpdateInterview`.
- Key dependencies: actor metadata match, schedule permission/scope, `InterviewRepo.GetModelByID`, RFC3339 scheduled time parsing, `InterviewRepo.UpdateWithTx`, `ApplicationRepo.GetDetail`, and outbox notification/email writes.
- Side effects: updates `interview_schedules`, writes notification/email outbox events for interviewer and candidate.

### Cancel interview

- Protobuf: `CancelInterviewRequest` and `CommonResponse`.
- Legacy method: `service.InterviewService.CancelInterview`.
- Key dependencies: actor metadata match, schedule permission/scope, `InterviewRepo.GetModelByID`, `InterviewRepo.UpdateWithTx`, `ApplicationRepo.GetDetail`, `RecruitmentLifecycleProcessManager.ApplyTransitionTx`, and outbox notification/email writes.
- Side effects: marks one `interview_schedules` row as `cancelled`, may transition application status to `interview_cancelled`, writes notification/email outbox events when the lifecycle transition succeeds.

### Batch cancel interviews

- Protobuf: `BatchCancelInterviewsRequest` and `BatchCancelInterviewsResponse`.
- Legacy method: `service.InterviewService.BatchCancelInterviews`.
- Key dependencies: actor metadata match, schedule permission/scope, `ApplicationRepo.GetDetail`, `InterviewRepo.ListByApplication`, `InterviewRepo.CancelPendingByApplication`, `RecruitmentLifecycleProcessManager.ApplyTransitionTx`, and outbox notification/email writes.
- Side effects: marks active pending/scheduled interviews for one application as `cancelled`, may transition application status to `interview_cancelled`, and writes a single notification/email event group based on the first cancelled interview.

### Feedback submission and retrieval

- Protobuf: `SubmitFeedbackRequest`, `GetFeedbackRequest`, `CommonResponse`, and `GetFeedbackResponse`.
- Legacy methods: `service.InterviewService.SubmitFeedback` and `service.InterviewService.GetFeedback`.
- Key dependencies: actor metadata match, `interview.feedback.submit` permission, assigned-interviewer check, application id validation, terminal application status guard, duplicate feedback guard, recommendation/score validation, `InterviewRepo.CreateFeedbackWithTx`, `InterviewRepo.UpdateWithTx`, and `RecruitmentLifecycleProcessManager.ApplyTransitionTx`.
- Side effects: creates `interview_feedback`, may mark scheduled interview as `completed`, and may transition application status from `interview_pending` to `interviewing`.

### Listing and detail reads

- Protobuf: `GetInterviewRequest`, `ListInterviewersRequest`, `ListApplicationInterviewsRequest`, `ListMyInterviewsRequest`, and `ListCandidateInterviewsRequest`.
- Legacy methods: `GetInterview`, `ListInterviewers`, `ListApplicationInterviews`, `ListMyInterviews`, and `ListCandidateInterviews`.
- Key dependencies: actor metadata match, `interview.read` or `interview.schedule` permission, assigned-interviewer or job scope checks, `UserRepo.ListStaffByRole`, joined `InterviewRepo` read methods, feedback batch lookup, application detail lookup, and optional OSS resume URL signing.
- Candidate-facing behavior: candidate interview reads filter `internal_note` and omit cancelled interviews from candidate lists.

## 6. Table Access Boundary

Source: `smart-recruit-deploy/mysql-table-ownership.json`.

Interview owner tables:

- `interview_schedules`
- `interview_feedback`

Allowed transitional read:

- `applications` (owner: recruitment)

Current transitional direct read not yet reflected for Interview owner access:

- `jobs`
- `users`
- `candidate_profiles`
- `resumes`
- `authz` tables through permission/scope repositories

Current transitional write debt:

- Application lifecycle status changes are still written through the shared `RecruitmentLifecycleProcessManager` inside Interview transactions. This must become a Recruitment owner API/event/application port before shared cleanup.

Allowed shared/platform write:

- `event_outbox` (owner: platform)

No TASK-006 change modifies these ownership rules.

## 7. Current Test Coverage

Existing Interview service tests:

- `cmd/interview-service/main_test.go`
- `internal/runtime/runtime_test.go`

Current test focus:

- CLI/discovery runtime behavior.
- Runtime requires an Interview dependency.
- Runtime registers `InterviewService` with gRPC.

Legacy shared tests still covering Interview behavior:

- `smart-recruit-domain-go/service/interview_service_test.go`
- `smart-recruit-domain-go/repository/interview_repo_test.go`

Current local test gaps:

- No local Interview domain model/state tests yet.
- No local application command/query tests yet.
- No local persistence, mapper, or gRPC adapter tests yet.
- No local contract compatibility tests beyond runtime registration yet.

Expected next tests:

- TASK-007: local domain/application unit tests for schedule, update, cancel, batch cancel, feedback submission, feedback retrieval, and listing rules.
- TASK-008: local infrastructure/interface/runtime tests for repository adapters, outbox/client adapters, gRPC compatibility, and runtime dependency replacement.

## 8. Out-of-Scope Confirmation

TASK-006 intentionally does not:

- Move Interview business rules from `smart-recruit-domain-go`.
- Replace active runtime wiring away from shared `service.InterviewService`.
- Change protobuf or generated Go contracts.
- Change database schema, migrations, or table ownership.
- Change gateway routing or public HTTP behavior.
- Delete or deprecate shared legacy implementation.
- Start using the new skeleton packages from runtime.
