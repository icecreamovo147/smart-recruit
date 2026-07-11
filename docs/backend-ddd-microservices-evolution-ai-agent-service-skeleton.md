# Backend DDD Microservices Evolution AI Agent Service Skeleton

Last verified: 2026-07-12

TASK-BDME-030 adds a compile-safe AI Agent service binary skeleton without production traffic cutover.

## What Exists

- Command entrypoint: `logic-grpc-service/cmd/ai-agent-service/main.go`.
- Runtime descriptor: `logic-grpc-service/internal/aiagent/runtime/skeleton.go`.
- Runtime tests: `logic-grpc-service/internal/aiagent/runtime/skeleton_test.go`.

The skeleton uses the service binary registry entry for `ai-agent-service` and validates:

- role is `service`;
- cutover mode is `none`;
- `TrafficEnabled` is `false`;
- no network listener is bound;
- AI chat, agent-run, embedding, MCP, and memory runtime workers are not started;
- gateway traffic is not received.

## Local Commands

```bash
cd logic-grpc-service
go run ./cmd/ai-agent-service --describe
go run ./cmd/ai-agent-service --check
go test ./internal/aiagent/runtime ./cmd/ai-agent-service
```

Running the binary without flags exits with code `2` and explains that it is intentionally unrouted.

## Compatibility

This skeleton TASK does not change:

- `logic-grpc-service/main.go`;
- AI chat, agent-run, embedding, MCP, Skill, memory, or provider runtime behavior;
- gateway routing;
- protobuf or HTTP contracts;
- deployment manifests;
- Dockerfiles;
- AI tables, ownership, or schema.

## Future Cutover Requirements

A later AI Agent runtime extraction TASK must connect the runtime behind this binary with explicit shadow or dual-run wiring, dependency degradation behavior, readiness checks, metrics, rollback evidence, and gateway routing controls before the skeleton can receive production traffic.
