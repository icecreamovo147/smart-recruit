# smart-recruit-identity-service

Independent Identity service source root.

## Responsibility

- Own authentication, RBAC, principal context, token lifecycle, and security audit behavior.
- Expose gRPC service endpoints for gateway identity traffic.
- Keep auth semantics compatible with existing public HTTP behavior.

## Startup

This root now contains an explicit Identity gRPC runtime:

```bash
go run ./cmd/identity-service --check
go run ./cmd/identity-service --serve --addr :50061
```

The runtime uses local Identity domain/application/infrastructure/interface components against the shared MySQL schema, registers `AuthService` plus the Identity-owned `AdminService` subset, and initializes Nacos discovery/config, gRPC health, metrics, trace, and structured logs.
