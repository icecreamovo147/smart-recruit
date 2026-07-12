# smart-recruit-gateway

Independent HTTP gateway source root for Smart Recruit.

## Responsibility

- Own the public HTTP entry point and preserve existing frontend API behavior.
- Route requests to backend services through gRPC using Nacos discovery when enabled.
- Keep static fallback and route-mode rollback to the legacy `web-gin-service`/logic runtime during migration.

## Startup

Later TASKs add `cmd/gateway/main.go`, config, Dockerfile, and route-mode wiring. Until then, this root is a scaffolded Go module.

## Monolith Relationship

`web-gin-service` remains the active gateway fallback until scoped gateway cutover TASKs provide compatibility and rollback evidence.
