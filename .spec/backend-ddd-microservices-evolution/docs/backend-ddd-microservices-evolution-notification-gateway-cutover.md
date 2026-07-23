# Backend DDD Microservices Evolution Notification Gateway Cutover

Last verified: 2026-07-12

TASK-BDME-029 adds an explicit gateway routing control for Notification gRPC APIs while preserving the current default monolith route.

## Routing Controls

`web-gin-service` supports two Notification route modes:

- `NOTIFICATION_ROUTE_MODE=logic`: default. Notification HTTP APIs continue to use the main `GRPC_ADDR` connection.
- `NOTIFICATION_ROUTE_MODE=notification`: Notification HTTP APIs use `NOTIFICATION_GRPC_ADDR` through a separate gRPC client connection.

When `NOTIFICATION_ROUTE_MODE=notification`, startup fails fast if `NOTIFICATION_GRPC_ADDR` is empty. Any other route mode is rejected.

The gateway still exposes the same public HTTP routes and uses the same protobuf-generated `NotificationService` client contract. No HTTP path, protobuf message, authentication rule, authorization rule, database schema, frontend contract, or dependency manifest changes are introduced by this TASK.

## Realtime Delivery

The SSE stream endpoints continue to subscribe to the shared Redis notification channel:

```text
notification:{account_type}:{user_id}
```

This keeps realtime delivery compatible with the extracted Notification runtime because both the current monolith path and the extracted service path publish through the same notification cache/channel contract.

## Rollback

Rollback is configuration-only:

1. Set `NOTIFICATION_ROUTE_MODE=logic`.
2. Leave `NOTIFICATION_GRPC_ADDR` empty or unset.
3. Restart the gateway deployment.
4. Confirm gateway logs show `notification_route_mode=logic` and `notification_target_addr` equal to `GRPC_ADDR`.

Kubernetes and Docker defaults remain `logic`, so checked-in manifests do not route production traffic to the extracted service unless an environment explicitly overrides the values.

## Verification

Run:

```bash
cd web-gin-service
go test ./config ./rpc
go test ./...
```

The focused tests verify default logic routing, extracted service routing, missing-address fail-fast behavior, and invalid-mode rejection.
