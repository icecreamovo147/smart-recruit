# Recruitment Gateway Cutover

## Scope

TASK-BDME-039 adds gateway routing controls for Recruitment job, candidate profile/resume, and application lifecycle APIs. It preserves public HTTP behavior and existing protobuf contracts.

## Routing Modes

Default rollback-safe mode:

```text
RECRUITMENT_ROUTE_MODE=logic
RECRUITMENT_GRPC_ADDR=
```

Cutover mode:

```text
RECRUITMENT_ROUTE_MODE=recruitment
RECRUITMENT_GRPC_ADDR=dns:///recruitment-service:50051
```

When `RECRUITMENT_ROUTE_MODE=recruitment`, the gateway fails fast if `RECRUITMENT_GRPC_ADDR` is empty.

## Routed Clients

The gateway routes these generated clients to the Recruitment target in cutover mode:

- `JobService`
- `CandidateService`
- `ApplicationService`

All other generated clients remain on their existing targets. Identity, Notification, and AI Agent route switches continue to control only their own service surfaces.

## Rollback

Set:

```text
RECRUITMENT_ROUTE_MODE=logic
RECRUITMENT_GRPC_ADDR=
```

Then restart or roll the gateway. No database schema, protobuf, frontend, or public HTTP contract change is required for rollback.

## Verification

Run:

```bash
cd web-gin-service && go test ./config ./rpc
cd web-gin-service && go test ./...
cd logic-grpc-service && go test ./...
```

Harness verification for TASK-BDME-039 also runs the feature scope check, `agent-check.sh`, and knowledge validation.
