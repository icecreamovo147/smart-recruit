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
  - retrieval
  - debug
applies_to:
  - smart-recruit-ai-agent-service/internal/application/service/capability_service.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go
  - platform-frontend/src/views/ai/SemanticRetrievalDebugView.vue
source_refs:
  - smart-recruit-ai-agent-service/internal/application/service/capability_service.go
  - smart-recruit-ai-agent-service/internal/domain/policy/capability.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go
  - platform-frontend/src/views/ai/SemanticRetrievalDebugView.vue
last_verified: 2026-07-23
review_after: 2026-10-21
---

# Debug Agent Skill and Memory Retrieval

Check embedding runtime state, Agent Skill eligibility/current version/status, memory/session-summary repositories, context budget limits, and the Semantic Retrieval debug page. Run focused AI Agent tests around capability policy, selection, embedding fallback, and context assembly.

## Verification

Verified against capability policy/service, native AI gRPC entrypoints, and `SemanticRetrievalDebugView` on 2026-07-23.
