---
schema_version: 1
id: semantic-retrieval
title: Semantic retrieval for Agent Skills and memories
kind: architecture
status: active
owners:
  - agent-platform
tags:
  - agent
  - skill
  - memory
  - embedding
applies_to:
  - smart-recruit-ai-agent-service/internal/application/service/capability_service.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_store.go
  - platform-frontend/src/views/ai/SemanticRetrievalDebugView.vue
source_refs:
  - smart-recruit-ai-agent-service/internal/domain/policy/capability.go
  - smart-recruit-ai-agent-service/internal/application/service/capability_service.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go
  - smart-recruit-ai-agent-service/internal/infrastructure/provider/doc.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_store.go
  - platform-frontend/src/views/ai/SemanticRetrievalDebugView.vue
last_verified: 2026-07-20
review_after: 2026-10-14
---

# Semantic Retrieval for Agent Skills and Memories

Semantic retrieval is AI Agent service behavior. Agent Skill and memory retrieval combine eligibility, lexical/vector signals, metadata boosts, context limits, and embedding runtime availability. Embedding failures must degrade through explicit fallback rather than silently returning empty context.

## Verification

Verified against current repository files on 2026-07-14.
