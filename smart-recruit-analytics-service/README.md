# smart-recruit-analytics-service

Independent Analytics service source root.

## Responsibility

- Own reporting, projections, read models, and analytics query/runtime behavior.
- Consume domain-event envelopes for projection updates.
- Avoid writing transactional domain state owned by request services.

## Startup

```bash
GOWORK=off go test ./...
GOWORK=off go run ./cmd/analytics-service --check
GOWORK=off go run ./cmd/analytics-service --serve
```

The service registers the Analytics reporting subset of `AdminService` and uses the shared MySQL instance for Analytics-owned projection/read-model queries. It loads bootstrap config from environment and Nacos Config, registers discovery in Nacos, and exposes gRPC health plus metrics/trace/logging wiring.

Gateway traffic targets this service directly through discovery or `ANALYTICS_GRPC_ADDR`.
