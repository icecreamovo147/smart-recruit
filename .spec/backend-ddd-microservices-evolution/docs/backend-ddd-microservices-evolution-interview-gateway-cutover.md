# Interview Gateway Cutover

## Scope

TASK-BDME-041 adds gateway routing controls for Interview scheduling, interviewer task views, and feedback APIs. It preserves public HTTP behavior and existing protobuf contracts.

## Routing Modes

Default rollback-safe mode:

```text
INTERVIEW_ROUTE_MODE=logic
INTERVIEW_GRPC_ADDR=
```

Cutover mode:

```text
INTERVIEW_ROUTE_MODE=interview
INTERVIEW_GRPC_ADDR=dns:///interview-service:50051
```

When `INTERVIEW_ROUTE_MODE=interview`, the gateway fails fast if `INTERVIEW_GRPC_ADDR` is empty.

## Routed Clients

The gateway routes this generated client to the Interview target in cutover mode:

- `InterviewService`

All other generated clients remain on their existing targets. Recruitment, Identity, Notification, and AI Agent route switches continue to control only their own service surfaces.

## Rollback

Set:

```text
INTERVIEW_ROUTE_MODE=logic
INTERVIEW_GRPC_ADDR=
```

Then restart or roll the gateway. No database schema, protobuf, frontend, or public HTTP contract change is required for rollback.

## Verification

Run:

```bash
cd web-gin-service && go test ./config ./rpc
cd web-gin-service && go test ./...
cd logic-grpc-service && go test ./...
```

Harness verification for TASK-BDME-041 also runs the feature scope check, `agent-check.sh`, and knowledge validation.
