# smart-recruit-identity-service

Independent Identity service source root.

## Responsibility

- Own authentication, RBAC, principal context, token lifecycle, and security audit behavior.
- Expose gRPC service endpoints for gateway identity traffic after scoped cutover.
- Keep auth semantics compatible with the current monolith.

## Startup

This root now contains an explicit Identity gRPC runtime:

```bash
go run ./cmd/identity-service --check
go run ./cmd/identity-service --serve --addr :50061
```

The runtime reuses the existing repository/service implementation against the shared MySQL schema, registers `AuthService` plus the Identity-owned `AdminService` subset, and initializes Nacos discovery/config, gRPC health, metrics, trace, and structured logs. Gateway traffic still remains on `logic` until the scoped cutover TASK records rollback evidence.

## Monolith Relationship

Identity traffic remains on `logic-grpc-service` until the Identity extraction and gateway cutover TASKs pass validation and record rollback evidence.
