# Backend DDD Microservices Evolution AI Agent Runtime Extraction

Last verified: 2026-07-12

TASK-BDME-031 extracts the current AI Agent runtime composition while preserving existing monolith behavior and avoiding production traffic cutover.

## Runtime Boundary

`logic-grpc-service/service/ai_agent_runtime.go` now owns the composition of:

- `AIService` for HR chat, durable agent runs, tool execution, runtime policy, trace recording, provider fallback, and agent-run event dispatch;
- `CandidateAIService` for candidate assistant chat behavior;
- `LlmConfigService` as the model/provider fallback configuration surface used by runtime extraction paths;
- `EmbeddingService` as the runtime dependency used by context assembly and semantic ranking;
- `EmbeddingConsumer` for asynchronous embedding workload execution through Inbox idempotency;
- `AgentRunConsumer` for durable agent-run execution through Inbox idempotency.

`service.NewServices` creates this runtime and exposes the same fields as before for compatibility. `logic-grpc-service/main.go` starts the AI Agent runtime after the Notification runtime and resume parse consumer in the existing background-worker block.

## Compatibility Guarantees

This TASK does not change:

- protobuf or HTTP contracts;
- gateway routing;
- table schemas or migrations;
- Dockerfiles or Kubernetes routed deployments;
- RabbitMQ queue names, routing keys, or message payload shapes;
- frontend behavior;
- AI runtime provider/model selection, prompt templates, memory ranking, embedding behavior, MCP governance, or Skill selection.

Sensitive prompt diagnostics are logged as character count plus SHA-256 fingerprint only. Resume parsing and AI trace surfaces continue to use existing safe previews, truncation, and PII masking helpers rather than logging raw resume or prompt text.

Normal monolith worker startup remains behaviorally equivalent:

1. Start Notification runtime outbox dispatcher, notification consumer, and email consumer.
2. Start resume parse consumer.
3. Start AI Agent runtime embedding consumer.
4. Start AI Agent runtime durable agent-run consumer.
5. Keep RabbitMQ keepalive in the existing worker block.

## Verification

Run:

```bash
cd logic-grpc-service
go test ./service -run 'TestNewAIAgentRuntime|TestAIAgentRuntime'
go test ./...
```

The focused tests verify that the runtime wires AI Agent service, candidate AI service, provider fallback/config surface, embedding service, runtime name, and nil runtime/MQ failure paths.

## Future Work

- A later AI Agent knowledge runtime extraction TASK can move prompt, skill, memory, embedding, MCP governance, and AI-derived intelligence ownership behind narrower runtime boundaries.
- A gateway cutover TASK must still own any route changes and rollback evidence.
- Extracted deployment manifests must still provide dependency degradation checks, metrics, and traffic controls before the AI Agent service receives production traffic.
