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
  - smart-recruit-ai-agent-service/internal/legacydomain/service/agent_skill_*.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/agent_context.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/embedding_*.go
  - hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue
source_refs:
  - smart-recruit-ai-agent-service/internal/legacydomain/service/agent_skill_service.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/agent_skill_selector.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/agent_context.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/embedding_service.go
  - smart-recruit-ai-agent-service/internal/domain/policy/capability.go
  - hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue
last_verified: 2026-07-14
review_after: 2026-10-14
---

# Debug Agent Skill and Memory Retrieval

Check embedding runtime state, Agent Skill eligibility/current version/status, memory/session-summary repositories, context budget limits, and the Semantic Retrieval debug page. Run focused AI Agent tests around capability policy, selection, embedding fallback, and context assembly.

## Verification

Verified against current repository files on 2026-07-14.
