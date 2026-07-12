# smart-recruit-ai-agent-service

Independent AI Agent service source root.

## Responsibility

- Own AI chat, candidate assistant flows, provider/config access, embedding runtime coordination, and durable agent-run orchestration.
- Keep provider secrets out of Git and runtime logs.
- Move long-running work to worker/event paths where appropriate.

## Startup

Later TASKs add service runtime, config, health, metrics, tracing, and Docker support. Until then, this root is a scaffolded Go module.

## Monolith Relationship

AI Agent traffic remains on `logic-grpc-service` until scoped extraction and gateway cutover TASKs pass validation and preserve rollback.
