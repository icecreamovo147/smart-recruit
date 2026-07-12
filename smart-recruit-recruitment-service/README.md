# smart-recruit-recruitment-service

Independent Recruitment service source root.

## Responsibility

- Own jobs, candidates, applications, and recruitment lifecycle behavior.
- Expose gRPC service endpoints for gateway recruitment traffic after scoped cutover.
- Coordinate cross-domain changes through gRPC or events rather than direct cross-service table writes.

## Startup

Later TASKs add service runtime, config, health, metrics, tracing, and Docker support. Until then, this root is a scaffolded Go module.

## Monolith Relationship

Recruitment traffic remains on `logic-grpc-service` until extraction and route-mode cutover evidence is complete.
