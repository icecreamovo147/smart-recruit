# smart-recruit-identity-service

Independent Identity service source root.

## Responsibility

- Own authentication, RBAC, principal context, token lifecycle, and security audit behavior.
- Expose gRPC service endpoints for gateway identity traffic after scoped cutover.
- Keep auth semantics compatible with the current monolith.

## Startup

Later TASKs add service runtime, config, health, metrics, tracing, and Docker support. Until then, this root is a scaffolded Go module.

## Monolith Relationship

Identity traffic remains on `logic-grpc-service` until the Identity extraction and gateway cutover TASKs pass validation and record rollback evidence.
