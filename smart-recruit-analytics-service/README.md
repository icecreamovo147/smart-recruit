# smart-recruit-analytics-service

Independent Analytics service source root.

## Responsibility

- Own reporting, projections, read models, and analytics query/runtime behavior.
- Consume domain-event envelopes for projection updates.
- Avoid writing transactional domain state owned by request services.

## Startup

Later TASKs add service runtime, config, health, metrics, tracing, and Docker support. Until then, this root is a scaffolded Go module.

## Monolith Relationship

Analytics traffic remains on `logic-grpc-service` until Analytics extraction and gateway route-mode validation is complete.
