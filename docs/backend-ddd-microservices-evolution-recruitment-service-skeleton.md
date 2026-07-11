# Recruitment Service Skeleton

## Scope

TASK-BDME-037 creates a compile-safe Recruitment service binary skeleton without production recruitment traffic cutover.

The skeleton follows the service binary convention already registered as `recruitment-service`:

- command: `logic-grpc-service/cmd/recruitment-service`
- image convention: `recruitment/recruitment-service`
- config prefix: `RECRUITMENT_`
- health convention: gRPC health before traffic routing
- cutover state: no traffic until a later Recruitment cutover TASK

## Runtime Behavior

`cmd/recruitment-service` supports:

- `--check`: validates the skeleton descriptor and exits 0.
- `--describe`: prints the registered service unit, cutover mode, startup mode, owned API surface, and safety notes.

Running without flags exits non-zero and prints that the skeleton is intentionally unrouted.

Default execution does not:

- bind a network listener;
- register generated gRPC services;
- start recruitment lifecycle workers or consumers;
- mutate jobs, candidates, resumes, applications, statuses, notifications, or Analytics projections;
- receive gateway traffic.

## Owned API Surface

The descriptor documents the future Recruitment-owned generated service surface:

- `JobService`: job create/update, online/offline, HR list, public list, and detail.
- `CandidateService`: profile, resume, presign upload, and confirm upload.
- `ApplicationService`: apply, candidate application list, job application list, status update, and status transition list.

This TASK does not register those generated services. Runtime registration and API extraction must happen in later scoped TASKs.

## Compatibility

Existing job, candidate profile, resume, and application lifecycle behavior remains in the current monolith paths. The gateway continues to route Recruitment-related HTTP requests to the existing logic service.

This skeleton TASK does not change:

- `logic-grpc-service/main.go`;
- existing job, candidate, resume, application, interview, offer, collaboration, notification, or Analytics behavior;
- gateway routing;
- protobuf or HTTP contracts;
- deployment manifests;
- Dockerfiles;
- recruitment table ownership or schema.

## Verification

Run:

```bash
cd logic-grpc-service && go run ./cmd/recruitment-service --check && go run ./cmd/recruitment-service --describe
cd logic-grpc-service && go test ./internal/recruitment/runtime ./cmd/recruitment-service ./internal/platform/servicebinary
cd logic-grpc-service && go test ./...
```

Harness verification for TASK-BDME-037 also runs the feature scope check, `agent-check.sh`, and knowledge validation.
