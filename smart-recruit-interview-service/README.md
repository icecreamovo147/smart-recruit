# smart-recruit-interview-service

Independent Interview service source root.

## Responsibility

- Own interview scheduling, interviewer tasks, interview feedback, and interview-specific runtime behavior.
- Coordinate application lifecycle effects through Recruitment APIs or events.
- Preserve interviewer and HR API compatibility through the gateway.

## Startup

Later TASKs add service runtime, config, health, metrics, tracing, and Docker support. Until then, this root is a scaffolded Go module.

## Monolith Relationship

Interview traffic remains on `logic-grpc-service` until the Interview service and gateway cutover TASKs pass validation with rollback evidence.
