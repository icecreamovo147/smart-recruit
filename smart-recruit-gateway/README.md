# smart-recruit-gateway

Independent HTTP gateway source root for Smart Recruit.

## Responsibility

- Own the public HTTP entry point and preserve existing frontend API behavior.
- Route requests to backend services through gRPC using Nacos discovery when enabled.
- Keep static fallback and route-mode rollback to the legacy `web-gin-service`/logic runtime during migration.

## Startup

This root now contains a compatibility gateway entry point:

```bash
go run ./cmd/gateway
```

The entry point reuses the current `web-gin-service` router/config/rpc packages during migration to preserve HTTP behavior, while the module also imports the unified `smart-recruit-proto` and `smart-recruit-platform-go` foundations for later Nacos discovery/config cutover.

## Monolith Relationship

`web-gin-service` remains the active gateway fallback until scoped gateway cutover TASKs provide compatibility and rollback evidence.
