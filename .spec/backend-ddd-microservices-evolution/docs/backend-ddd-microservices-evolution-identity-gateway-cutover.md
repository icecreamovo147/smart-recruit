# Identity Gateway Cutover

## Scope

TASK-BDME-036 adds gateway routing controls for Identity authentication and authorization integration. It preserves public HTTP behavior and existing protobuf contracts.

## Routing Modes

Default rollback-safe mode:

```text
IDENTITY_ROUTE_MODE=logic
IDENTITY_GRPC_ADDR=
```

Cutover mode:

```text
IDENTITY_ROUTE_MODE=identity
IDENTITY_GRPC_ADDR=dns:///identity-service:50051
```

When `IDENTITY_ROUTE_MODE=identity`, the gateway fails fast if `IDENTITY_GRPC_ADDR` is empty.

## Routed Clients

`AuthService` is routed to the Identity target in cutover mode:

- register
- login
- refresh token
- revoke refresh token
- current principal
- update email
- authorization audit write

`AdminService` uses a hybrid gateway client in cutover mode. These Identity-owned methods route to the Identity target:

- `ListRoles`
- `ListPermissions`
- `GetUserRoles`
- `AssignUserRole`
- `RevokeUserRole`
- `AssignDataScope`
- `RevokeDataScope`
- `ListStaffUsers`
- `CreateStaffUser`
- `QueryAuthAuditLogs`

All other `AdminService` methods stay on the logic target, including invite-code, usage-log, analytics dashboard, department, location, and department-location configuration calls.

## Rollback

Set:

```text
IDENTITY_ROUTE_MODE=logic
IDENTITY_GRPC_ADDR=
```

Then restart or roll the gateway. No database schema, protobuf, cookie, frontend, or public HTTP contract change is required for rollback.

## Verification

Run:

```bash
cd web-gin-service && go test ./config ./rpc
cd web-gin-service && go test ./...
cd logic-grpc-service && go test ./...
```

Harness verification for TASK-BDME-036 also runs the feature scope check, `agent-check.sh`, and knowledge validation.
