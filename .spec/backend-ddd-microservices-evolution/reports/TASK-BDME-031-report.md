# TASK-BDME-031 Report - AI Agent Runtime Extraction

## Summary

Extracted the current AI Agent runtime composition into `service.AIAgentRuntime` while preserving monolith behavior. The runtime groups HR AI service, candidate AI service, provider fallback/config surface, embedding service, embedding workload consumer, durable agent-run consumer, runtime policy, and runtime name. Background startup now invokes this runtime instead of starting embedding and agent-run consumers directly.

## Modified Files

- `logic-grpc-service/service/ai_agent_runtime.go`: added AI Agent runtime composition and worker startup orchestration.
- `logic-grpc-service/service/ai_agent_runtime_test.go`: covered runtime wiring and nil runtime/MQ failure paths.
- `logic-grpc-service/service/services.go`: constructs `AIAgentRuntime` and exposes legacy fields from the runtime for compatibility.
- `logic-grpc-service/main.go`: starts AI Agent runtime in the existing background worker block.
- `docs/backend-ddd-microservices-evolution-ai-agent-runtime-extraction.md`: documented runtime boundary, compatibility guarantees, verification, and future work.
- `docs/backend-ddd-microservices-evolution-ai-agent-service-skeleton.md`: linked the skeleton doc to the runtime extraction.
- `.knowledge/architecture/agent-runtime.md`: documented `service.AIAgentRuntime`.
- `.knowledge/architecture/service-boundaries.md`: documented the transitional monolith AI Agent runtime boundary.
- `.knowledge/runbooks/service-binary-convention.md`: updated runtime extraction guidance.
- `.knowledge/manifest.yaml`: routed AI Agent runtime files and docs to relevant knowledge.
- `.spec/backend-ddd-microservices-evolution/pipeline-state.json`: recorded TASK progress and completion.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-031-report.md`: this report.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-031-evidence.json`: machine-readable evidence.

## Scope

Scope check passed. All changed files are allowed by TASK-BDME-031 scope. No frontend, dependency manifest, `go.mod`, `go.sum`, database schema, protobuf, gateway route, public API, auth policy, deployment traffic, SPEC/SDD/TASK, or Harness contract files were modified.

Human confirmation was required by the TASK and is recorded as satisfied by the user's global gate override.

## SPEC / SDD / Acceptance

- SPEC comparison: aligned with FR-002 and FR-021..FR-025 by moving AI Agent runtime composition behind an extraction boundary without changing public behavior.
- SDD comparison: aligned with target service architecture and shadow/cutover strategy by keeping runtime extraction inside the monolith and avoiding gateway/deployment cutover.
- Acceptance comparison: implemented the scoped runtime extraction, preserved existing behavior, and recorded scope, tests, risks, and knowledge impact.

## Checks

- `git diff --name-only && git ls-files --others --exclude-standard`: passed; listed tracked and untracked TASK files.
- `cd logic-grpc-service && go test ./service -run 'TestNewAIAgentRuntime|TestAIAgentRuntime'`: passed.
- `node .knowledge/scripts/validate-knowledge.mjs --root .`: passed.
- `cd logic-grpc-service && go test ./...`: passed.
- `TASK_BASE_TREE=f52cf3bbd702367ee154ea947a03f010cb326122 bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-031`: passed.
- `bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh`: passed.
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree f52cf3bbd702367ee154ea947a03f010cb326122 --json`: passed; update required, no coverage gaps.

## Self-Review

Verdict: 通过.

Findings: none. Worker startup order remains equivalent: Notification runtime, resume parse consumer, AI Agent embedding consumer, AI Agent agent-run consumer, then RabbitMQ keepalive. Public contracts, MQ routing keys, schemas, dependencies, and deployment traffic are unchanged.

## Risks

- Future extraction must still split prompt/skill/memory/embedding/MCP ownership more deeply and add dependency degradation, readiness, metrics, rollback evidence, and gateway controls before production routing.
- `service.AIAgentRuntime` is a composition boundary only; it does not make `cmd/ai-agent-service` serve traffic.

## Next TASK

TASK-BDME-032 can start after this TASK is committed.
