# Interview Service Contract Inventory

## 1. Purpose

This inventory tracks the Interview migration baseline for `.spec/microservice-ddd-evolution`.

It records the current public contract, runtime wiring, shared-domain dependencies, table access boundary, event side effects, and test coverage as Interview business behavior moves into local DDD layers.

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

TASK-006 created package boundaries and recorded the legacy contract baseline.

TASK-007 moved Interview domain/application behavior into local packages:

- `internal/domain/model`: Interview schedule, feedback, status, application status, terminal-status guard, and domain validation.
- `internal/domain/service`: schedule/cancel/feedback lifecycle transition policy.
- `internal/domain/repository`: Interview repository port and transactional writer port.
- `internal/application/command`: schedule, update, cancel, batch cancel, and feedback command DTOs.
- `internal/application/query`: listing/detail query DTOs for future interface wiring.
- `internal/application/port`: authorizer, application snapshot, application lifecycle, outbox, and clock ports.
- `internal/application/service`: Interview use-case orchestration for schedule/update/cancel/batch cancel/feedback and feedback retrieval.

TASK-008 wires the active runtime path to local infrastructure, application, and gRPC adapters:

- `internal/infrastructure/persistence`: shared repository backed Interview repository adapter and transaction context bridge.
- `internal/infrastructure/client`: application snapshot/lifecycle, authorizer, and staff directory adapters.
- `internal/infrastructure/mq`: legacy-compatible event_outbox publisher adapter.
- `internal/interfaces/mapper`: proto mapping helpers.
- `internal/interfaces/grpc`: local `pb.InterviewServiceServer` implementation.
- `cmd/interview-service/main.go`: constructs local dependencies instead of shared `service.NewServices`.

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

- TASK-007 does not change protobuf request/response types, rpc names, route mode, gateway behavior, or runtime registration behavior.
- Existing gRPC messages and response codes are mapped by `internal/interfaces/grpc` for the active runtime path.
- TASK-008 does not change protobuf request/response types, rpc names, route mode, gateway behavior, or runtime registration behavior.

## 4. Current Compatibility Dependency Baseline

Current direct imports in `cmd/interview-service/main.go` after TASK-029:

- `smart-recruit-interview-service/internal/legacydomain/repository`
- local `internal/application/service`
- local `internal/infrastructure/client`
- local `internal/infrastructure/mq`
- local `internal/infrastructure/persistence`
- local `internal/interfaces/grpc`
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

- `buildInterviewServer` constructs only Interview-required repositories/adapters and returns local `interfaces/grpc.Server`.
- The active gRPC implementation no longer uses `smart-recruit-commons/service.InterviewService`.

Owner-local legacy repositories constructed as temporary infrastructure adapters:

- `repository.NewUserRepo`
- `repository.NewJobRepo`
- `repository.NewApplicationRepo`
- `repository.NewInterviewRepo`
- `repository.NewOutboxRepo`
- `repository.NewAuthzRepo`

Migration implication:

- TASK-007 moved Interview domain model, lifecycle policy, feedback rules, repository ports, command/query DTOs, and application ports into local packages.
- TASK-008 replaces active runtime wiring with local `interfaces/grpc` backed by local application services, and removes active direct dependency on shared `service.InterviewService`.
- Remaining shared `model`/`repository` imports are infrastructure-only transitional debt until shared cleanup.

## 5. Current Behavior Dependency Map

### Schedule interview

- Protobuf: `ScheduleInterviewRequest` and `ScheduleInterviewResponse`.
- Legacy method: `service.InterviewService.ScheduleInterview`.
- Local application method: `internal/application/service.InterviewService.ScheduleInterview`.
- Key dependencies: actor metadata match, `interview.schedule` permission, application/job scope, `ApplicationRepo.GetDetail`, `JobRepo.GetByID`, `JobRepo.BelongsToHR`, `InterviewRepo.GetMaxRoundNo`, `InterviewRepo.CreateWithTx`, `RecruitmentLifecycleProcessManager.ApplyTransitionTx`, and outbox notification/email writes.
- Local ports: `Authorizer`, `ApplicationSnapshotReader`, `InterviewRepository`, `ApplicationLifecycle`, and `OutboxPublisher`.
- Side effects: creates `interview_schedules`, may transition application status to `interview_pending`, writes notification/email outbox events for interviewer and candidate.

### Update interview

- Protobuf: `UpdateInterviewRequest` and `CommonResponse`.
- Legacy method: `service.InterviewService.UpdateInterview`.
- Local application method: `internal/application/service.InterviewService.UpdateInterview`.
- Key dependencies: actor metadata match, schedule permission/scope, `InterviewRepo.GetModelByID`, RFC3339 scheduled time parsing, `InterviewRepo.UpdateWithTx`, `ApplicationRepo.GetDetail`, and outbox notification/email writes.
- Local ports: `Authorizer`, `ApplicationSnapshotReader`, `InterviewRepository`, and `OutboxPublisher`.
- Side effects: updates `interview_schedules`, writes notification/email outbox events for interviewer and candidate.

### Cancel interview

- Protobuf: `CancelInterviewRequest` and `CommonResponse`.
- Legacy method: `service.InterviewService.CancelInterview`.
- Local application method: `internal/application/service.InterviewService.CancelInterview`.
- Key dependencies: actor metadata match, schedule permission/scope, `InterviewRepo.GetModelByID`, `InterviewRepo.UpdateWithTx`, `ApplicationRepo.GetDetail`, `RecruitmentLifecycleProcessManager.ApplyTransitionTx`, and outbox notification/email writes.
- Local ports: `Authorizer`, `ApplicationSnapshotReader`, `InterviewRepository`, `ApplicationLifecycle`, and `OutboxPublisher`.
- Side effects: marks one `interview_schedules` row as `cancelled`, may transition application status to `interview_cancelled`, writes notification/email outbox events when the lifecycle transition succeeds.

### Batch cancel interviews

- Protobuf: `BatchCancelInterviewsRequest` and `BatchCancelInterviewsResponse`.
- Legacy method: `service.InterviewService.BatchCancelInterviews`.
- Local application method: `internal/application/service.InterviewService.BatchCancelInterviews`.
- Key dependencies: actor metadata match, schedule permission/scope, `ApplicationRepo.GetDetail`, `InterviewRepo.ListByApplication`, `InterviewRepo.CancelPendingByApplication`, `RecruitmentLifecycleProcessManager.ApplyTransitionTx`, and outbox notification/email writes.
- Local ports: `Authorizer`, `ApplicationSnapshotReader`, `InterviewRepository`, `ApplicationLifecycle`, and `OutboxPublisher`.
- Side effects: marks active pending/scheduled interviews for one application as `cancelled`, may transition application status to `interview_cancelled`, and writes a single notification/email event group based on the first cancelled interview.

### Feedback submission and retrieval

- Protobuf: `SubmitFeedbackRequest`, `GetFeedbackRequest`, `CommonResponse`, and `GetFeedbackResponse`.
- Legacy methods: `service.InterviewService.SubmitFeedback` and `service.InterviewService.GetFeedback`.
- Local application methods: `internal/application/service.InterviewService.SubmitFeedback` and `GetFeedback`.
- Key dependencies: actor metadata match, `interview.feedback.submit` permission, assigned-interviewer check, application id validation, terminal application status guard, duplicate feedback guard, recommendation/score validation, `InterviewRepo.CreateFeedbackWithTx`, `InterviewRepo.UpdateWithTx`, and `RecruitmentLifecycleProcessManager.ApplyTransitionTx`.
- Local ports: `Authorizer`, `InterviewRepository`, `ApplicationLifecycle`, and `Clock`.
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

No TASK-007 change modifies these ownership rules.

## 7. Current Test Coverage

Existing Interview service tests:

- `cmd/interview-service/main_test.go`
- `internal/runtime/runtime_test.go`
- `internal/domain/model/interview_test.go`
- `internal/domain/model/feedback_test.go`
- `internal/application/service/interview_service_test.go`
- `internal/infrastructure/mq/outbox_publisher_test.go`
- `internal/interfaces/grpc/interview_server_test.go`

Current test focus:

- CLI/discovery runtime behavior.
- Runtime requires an Interview dependency.
- Runtime registers `InterviewService` with gRPC.
- Domain schedule defaults, cancellation guard, feedback recommendation and score validation.
- Application schedule/update/cancel/batch cancel/feedback state transitions and notification/outbox publication semantics.
- Outbox legacy envelope, idempotency key, routing metadata, and interview email payload compatibility.
- gRPC legacy response message and error-code compatibility for schedule, batch cancel, feedback, and get interview.

Legacy shared tests still covering Interview behavior:

- `smart-recruit-commons/service/interview_service_test.go`
- `smart-recruit-commons/repository/interview_repo_test.go`

Current local test gaps:

- No local application listing query tests yet.
- No local DB-backed persistence adapter tests yet; adapters rely on shared repository tests plus service-level compile/tests.

Expected next tests:

- Later cleanup tasks should add DB-backed persistence tests if adapter logic grows beyond mapping/transaction bridging.

## 8. Out-of-Scope Confirmation

TASK-008 intentionally does not:

- Change protobuf or generated Go contracts.
- Change database schema, migrations, or table ownership.
- Change gateway routing or public HTTP behavior.
- Delete or deprecate shared legacy implementation.
