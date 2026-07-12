# Recruitment API Extraction

## Scope

TASK-BDME-038 extracts the Recruitment-owned generated gRPC service surface into `internal/recruitment/runtime` without gateway cutover.

The extracted runtime can explicitly register:

- `JobService`
- `CandidateService`
- `ApplicationService`

The runtime uses thin adapters under `internal/recruitment/interfaces` that forward to the current monolith services. No protobuf, HTTP, database schema, or gateway route contract changes are introduced.

## Extracted APIs

`JobService`:

- `CreateJob`
- `UpdateJob`
- `OfflineJob`
- `OnlineJob`
- `ListHRJobs`
- `ListPublicJobs`
- `GetJobDetail`
- `ListJobOptions`
- `ListDepartmentLocations`

`CandidateService`:

- `GetProfile`
- `UpdateProfile`
- `GetResume`
- `PresignResumeUpload`
- `ConfirmResumeUpload`

`ApplicationService`:

- `ApplyJob`
- `ListMyApplications`
- `ListJobApplications`
- `UpdateApplicationStatus`
- `ListApplicationStatusTransitions`

## Runtime Behavior

`runtime.New` requires explicit dependencies for job, job taxonomy, candidate profile/resume, and application lifecycle APIs. `Runtime.RegisterGRPC` registers only the Recruitment-owned generated services listed above.

The `cmd/recruitment-service` command remains fail-closed by default. It supports descriptor validation and description, but it does not bind a listener or receive gateway traffic in this TASK.

## Compatibility

Current production traffic still reaches the monolith `logic-grpc-service`. Existing job, candidate profile, resume, application status, notification side effects, outbox events, and Analytics projection behavior remain on the current service path until a later gateway cutover TASK.

## Verification

Run:

```bash
cd logic-grpc-service && go test ./internal/recruitment/interfaces ./internal/recruitment/runtime ./cmd/recruitment-service
cd logic-grpc-service && go run ./cmd/recruitment-service --check && go run ./cmd/recruitment-service --describe
cd logic-grpc-service && go test ./...
```

Harness verification for TASK-BDME-038 also runs the feature scope check, `agent-check.sh`, and knowledge validation.
