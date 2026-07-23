# Interview Service Skeleton And API Extraction

## Scope

TASK-BDME-040 creates a compile-safe Interview service binary skeleton and extracts the Interview-owned generated gRPC service surface into `internal/interview/runtime` without gateway cutover.

The runtime can explicitly register:

- `InterviewService`

The runtime uses a thin adapter under `internal/interview/interfaces` that forwards to the current monolith `InterviewService`. No protobuf, HTTP, database schema, or gateway route contract changes are introduced.

## Extracted APIs

`InterviewService`:

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

## Runtime Behavior

`cmd/interview-service` supports:

- `--check`: validates the descriptor and exits 0.
- `--describe`: prints the registered service unit, cutover mode, startup mode, extracted API surface, and safety notes.

Default command execution exits non-zero and does not bind a listener, start workers, register services, mutate interview/application lifecycle state, or receive gateway traffic.

`runtime.New` requires an explicit Interview API dependency. `Runtime.RegisterGRPC` registers only InterviewService.

## Compatibility

Current production traffic still reaches the monolith `logic-grpc-service`. Existing interview scheduling, updates, cancellations, feedback, notification side effects, and application lifecycle transitions remain on the current service path until a later gateway cutover TASK.

## Verification

Run:

```bash
cd logic-grpc-service && go test ./internal/interview/interfaces ./internal/interview/runtime ./cmd/interview-service
cd logic-grpc-service && go run ./cmd/interview-service --check && go run ./cmd/interview-service --describe
cd logic-grpc-service && go test ./...
```

Harness verification for TASK-BDME-040 also runs the feature scope check, `agent-check.sh`, and knowledge validation.
