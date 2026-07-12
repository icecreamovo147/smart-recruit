# smart-recruit-platform-go

Shared Go runtime platform for extracted Smart Recruit services.

## Responsibility

- Provide service bootstrap, configuration, metadata, gRPC server/client helpers, and internal auth primitives.
- Host Nacos discovery/config, logging, metrics, tracing, health, Redis prefix, service config, and gRPC runtime helpers.
- Keep platform concerns outside domain service packages.

## Packages

- `config`: loads and validates bootstrap runtime settings from environment variables.
- `servicemeta`: defines service identity and common gRPC metadata headers.
- `grpcx`: builds baseline gRPC server/client options, including internal-token propagation.
- `nacos`: provides Nacos OpenAPI based config/discovery adapters with local static fallback and non-local fail-fast behavior.
- `observability`: initializes Prometheus metrics, OpenTelemetry tracing, and health/readiness helpers.
- `logger`, `metadata`, `server`, `serviceconfig`: shared runtime support used by the service source roots.

## Startup

This root is a library module and does not run a server directly. Service modules import it as runtime capabilities are implemented.
