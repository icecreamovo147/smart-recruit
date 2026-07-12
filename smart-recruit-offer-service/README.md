# smart-recruit-offer-service

Independent Offer service source root.

## Responsibility

- Own offer creation, updates, send/revoke behavior, candidate response handling, and offer lifecycle state.
- Coordinate application lifecycle changes through gRPC or events.
- Avoid direct writes to Recruitment-owned tables.

## Startup

Later TASKs add service runtime, config, health, metrics, tracing, and Docker support. Until then, this root is a scaffolded Go module.

## Monolith Relationship

Offer traffic remains on `logic-grpc-service` until the Offer extraction and gateway cutover TASKs provide compatibility and rollback evidence.
