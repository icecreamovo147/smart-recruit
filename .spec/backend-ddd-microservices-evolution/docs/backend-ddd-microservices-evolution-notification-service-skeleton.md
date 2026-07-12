# Backend DDD Microservices Evolution Notification Service Skeleton

Last verified: 2026-07-11

TASK-BDME-027 adds a compile-safe Notification service binary skeleton without production traffic cutover.

## What Exists

- Command entrypoint: `logic-grpc-service/cmd/notification-service/main.go`.
- Runtime descriptor: `logic-grpc-service/internal/notification/runtime/skeleton.go`.
- Runtime tests: `logic-grpc-service/internal/notification/runtime/skeleton_test.go`.

The skeleton uses the service binary registry entry for `notification-service` and validates:

- role is `service`;
- cutover mode is `none`;
- `TrafficEnabled` is `false`;
- no network listener is bound;
- notification consumers are not started;
- gateway traffic is not received.

## Local Commands

```bash
cd logic-grpc-service
go run ./cmd/notification-service --describe
go run ./cmd/notification-service --check
go test ./internal/notification/runtime ./cmd/notification-service
```

Running the binary without flags exits with code `2` and explains that it is intentionally unrouted.

## Compatibility

TASK-BDME-028 later extracted the current monolith Notification runtime composition into `logic-grpc-service/service/notification_runtime.go`. The skeleton command remains unrouted and still does not start the runtime by default.

This skeleton TASK does not change:

- `logic-grpc-service/main.go`;
- existing notification handlers or workers;
- gateway routing;
- protobuf or HTTP contracts;
- deployment manifests;
- Dockerfiles;
- notification table ownership or schema.

## Future Cutover Requirements

A later Notification cutover TASK must connect `cmd/notification-service` to the extracted runtime with explicit shadow or dual-run wiring, readiness behavior, metrics, rollback evidence, and gateway or worker routing changes before the skeleton can receive production traffic.
