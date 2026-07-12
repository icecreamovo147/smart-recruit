# Backend DDD Microservices Evolution AI Agent Gateway Cutover

Last verified: 2026-07-12

TASK-BDME-033 adds an explicit gateway routing control for AI Agent-owned gRPC APIs while preserving the current default monolith route.

## Routing Controls

`web-gin-service` supports two AI Agent route modes:

- `AI_AGENT_ROUTE_MODE=logic`: default. AI Agent-owned HTTP APIs continue to use the main `GRPC_ADDR` connection.
- `AI_AGENT_ROUTE_MODE=ai-agent`: AI Agent-owned generated gRPC clients use `AI_AGENT_GRPC_ADDR` through a separate gRPC client connection.

When `AI_AGENT_ROUTE_MODE=ai-agent`, startup fails fast if `AI_AGENT_GRPC_ADDR` is empty. Any other route mode is rejected.

The routed generated clients are:

- `AIService`
- `LlmConfigService`
- `PromptService`
- `AgentConfigService`
- `MCPService`
- `SkillService`
- `AgentSkillService`
- `RecruitingIntelligenceService`
- `EmbeddingConfigService`

The gateway keeps the same public HTTP routes and protobuf contracts. No HTTP path, protobuf message, authentication rule, authorization rule, database schema, frontend contract, dependency manifest, or default deployment traffic change is introduced by this TASK.

## Rollback

Rollback is configuration-only:

1. Set `AI_AGENT_ROUTE_MODE=logic`.
2. Leave `AI_AGENT_GRPC_ADDR` empty or unset.
3. Restart the gateway deployment.
4. Confirm gateway logs show `ai_agent_route_mode=logic` and `ai_agent_target_addr` equal to `GRPC_ADDR`.

Kubernetes and Docker defaults remain `logic`, so checked-in manifests do not route production traffic to the extracted service unless an environment explicitly overrides the values.

## Verification

Run:

```bash
cd web-gin-service
go test ./config ./rpc
go test ./...
```

The focused tests verify default logic routing, extracted service routing, missing-address fail-fast behavior, invalid-mode rejection, and readiness checks for the extracted AI Agent target.
