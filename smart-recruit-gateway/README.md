# smart-recruit-gateway

Independent HTTP gateway source root for Smart Recruit.

## Responsibility

- Own the public HTTP entry point and preserve existing frontend API behavior.
- Route requests to backend services through gRPC using Nacos discovery when enabled.
- Own the migrated Gateway HTTP router, handlers, middleware, gRPC clients, Redis integration and observability in this source root.

## Startup

This root contains the Gateway entry point and all HTTP-facing packages:

```bash
go run ./cmd/gateway
```

The entry point uses local `config`, `router`, `rpc`, `handler`, `middleware` and `pkg` packages. Protobuf contracts come from `smart-recruit-proto`; runtime foundations come from `smart-recruit-platform-go`.

## Nacos Runtime

`internal/runtime` includes Nacos config/discovery helpers. Local development can enable explicit static fallback; non-local environments fail fast when `NACOS_ADDR` is missing.

## Route Modes

Gateway route modes cover `identity`, `recruitment`, `interview`, `offer`, `notification`, `ai-agent`, and `analytics`. Setting a service to its own mode requires a matching target address and makes that target part of the gateway readiness plan.
