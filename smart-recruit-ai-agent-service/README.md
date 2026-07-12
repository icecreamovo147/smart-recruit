# smart-recruit-ai-agent-service

Independent AI Agent service source root.

## Responsibility

- Own AI chat, candidate assistant flows, provider/config access, embedding runtime coordination, and durable agent-run orchestration.
- Keep provider secrets out of Git and runtime logs.
- Move long-running work to worker/event paths where appropriate.

## Startup

This module now contains an independently buildable AI Agent gRPC runtime:

```bash
GOWORK=off go test ./...
GOWORK=off go run ./cmd/ai-agent-service --check
GOWORK=off go run ./cmd/ai-agent-service --serve --addr :50066
```

At runtime it reuses the shared Smart Recruit MySQL schema, registers AI-owned gRPC services for AI, Prompt, AgentConfig, MCP, Skill, AgentSkill, RecruitingIntelligence, and EmbeddingConfig, exposes gRPC health, starts metrics and tracing, registers the `ai-agent` instance through Nacos discovery when configured, and starts RabbitMQ-controlled embedding and agent-run workers.

## Monolith Relationship

AI Agent traffic remains on `logic-grpc-service` until scoped extraction and gateway cutover TASKs pass validation and preserve rollback.
