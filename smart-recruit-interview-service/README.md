# smart-recruit-interview-service

Independent Interview service source root.

## Responsibility

- Own interview scheduling, interviewer tasks, interview feedback, and interview-specific runtime behavior.
- Coordinate application lifecycle effects through Recruitment APIs or events.
- Preserve interviewer and HR API compatibility through the gateway.

## Startup

This module now contains an independently buildable Interview gRPC runtime:

```bash
GOWORK=off go test ./...
GOWORK=off go run ./cmd/interview-service --check
GOWORK=off go run ./cmd/interview-service --serve --addr :50063
```

At runtime it reuses the shared Smart Recruit MySQL schema through existing repositories, registers `InterviewService`, exposes gRPC health, starts the shared metrics endpoint, initializes tracing, and registers the `interview` instance through Nacos discovery when configured.

Gateway traffic targets this service directly through discovery or `INTERVIEW_GRPC_ADDR`.
