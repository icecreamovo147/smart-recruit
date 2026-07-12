# Identity API Extraction

## Scope

TASK-BDME-035 extracts the existing Identity auth, RBAC, scope, staff identity, and security-audit API surface into the `identity-service` runtime without gateway cutover.

This TASK does not change public HTTP behavior, protobuf contracts, database schema, cookie handling, refresh-token semantics, token-version invalidation semantics, or deployment traffic routing.

## Extracted Runtime Surface

The Identity runtime registers these generated gRPC services when `cmd/identity-service --serve` is used:

- `AuthService`
- `AdminService` Identity-owned subset

`AuthService` methods:

- `Register`
- `Login`
- `RefreshToken`
- `RevokeRefreshToken`
- `RecordAuthDecision`
- `GetPrincipal`
- `UpdateEmail`

`AdminService` Identity-owned methods:

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

Non-Identity `AdminService` methods, including invite-code, usage-log, department, location, and department-location configuration APIs, remain on the monolith target for this TASK.

## Runtime Startup

Default execution remains fail-closed and unrouted:

```bash
cd logic-grpc-service && go run ./cmd/identity-service
```

Descriptor validation:

```bash
cd logic-grpc-service && go run ./cmd/identity-service --check
```

Descriptor inspection:

```bash
cd logic-grpc-service && go run ./cmd/identity-service --describe
```

Explicit local runtime startup:

```bash
cd logic-grpc-service && go run ./cmd/identity-service --serve --addr :50061
```

`--serve` loads the existing logic service config, validates the internal gRPC token policy, connects to MySQL and optional Redis, registers the Identity runtime gRPC services, and exposes gRPC health. It does not run migrations, seed RBAC data, start workers, or receive gateway traffic.

## Compatibility

The existing monolith `logic-grpc-service` still registers the current `AuthService` and `AdminService` implementations. `web-gin-service` continues to route authentication, authorization, and admin traffic to the configured logic service until a later Identity gateway cutover TASK changes routing with rollback controls.

## Verification

Run:

```bash
cd logic-grpc-service && go test ./internal/identity/interfaces ./internal/identity/runtime ./cmd/identity-service
cd logic-grpc-service && go test ./...
```

Harness verification for TASK-BDME-035 also runs the feature scope check, `agent-check.sh`, and knowledge validation.
