---
schema_version: 1
id: debug-agent-retrieval
title: Debug Agent Skill and memory retrieval
kind: runbook
status: active
owners:
  - agent-platform
tags:
  - agent
  - skill
  - memory
  - embedding
applies_to:
  - logic-grpc-service/service/agent_skill_service.go
  - logic-grpc-service/service/agent_skill_selector.go
  - logic-grpc-service/service/agent_context.go
  - logic-grpc-service/service/embedding_service.go
  - hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue
source_refs:
  - logic-grpc-service/service/agent_skill_service.go
  - logic-grpc-service/service/agent_skill_selector.go
  - logic-grpc-service/service/agent_context.go
  - hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue
last_verified: 2026-07-10
review_after: 2026-10-08
---

# Debug Agent Skill and Memory Retrieval

Use this runbook when Agent Skill selection, semantic retrieval, memory recall, embedding fallback, or HR semantic debug output looks wrong.

## Diagnostic Path

1. Confirm the request path: manual Skill selection, automatic Skill selection, semantic debug, or normal AI chat.
2. Check whether an embedding provider is available in the debug response. If unavailable, expect fallback behavior rather than vector-backed ranking.
3. Compare selected Skill metadata against agent type, required capabilities, manual invocation flag, enabled state, and priority.
4. Compare memory output against recall scope, max memory count, and character budget.
5. Review pool confidence and ranking fields as diagnostic output, not as an authorization or product decision.
6. If debug output changed because protobuf fields changed, route the change through public-contract review.

## Useful Code Anchors

- `agent_skill_selector.go`: request-time Skill filtering, manual selection, ranking, and max pool behavior.
- `agent_skill_service.go`: semantic debug API and response assembly.
- `agent_context.go`: recent messages, summary, prompt template, memories, and budget.
- `embedding_service.go`: provider state, unavailable fallback, search metadata, and vector validation.

## Expected Fallback Clues

- Provider reported as unavailable.
- Fallback reason explains that embedding retrieval is unavailable.
- Skill selection may still show rule-based or metadata-driven results.
- Memory recall may still return scoped candidates, but without successful vector scoring.

## Safety

Do not paste live candidate data, credentials, or raw production logs into knowledge docs. Use sanitized examples in reports.
