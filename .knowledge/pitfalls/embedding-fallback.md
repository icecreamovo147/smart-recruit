---
schema_version: 1
id: embedding-fallback
title: Embedding fallback pitfall
kind: pitfall
status: active
owners:
  - agent-platform
tags:
  - embedding
  - fallback
  - retrieval
applies_to:
  - smart-recruit-ai-agent-service/internal/legacydomain/service/embedding_*.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/agent_skill_*.go
  - hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue
source_refs:
  - smart-recruit-ai-agent-service/internal/legacydomain/service/embedding_service.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/embedding_provider_factory.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/agent_skill_selector.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/agent_skill_service.go
  - smart-recruit-ai-agent-service/internal/domain/policy/capability.go
  - hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue
last_verified: 2026-07-14
review_after: 2026-10-14
---

# Embedding Fallback Pitfall

Embedding outages, disabled provider/model config, dimension mismatches, or credential failures must degrade explicitly. Retrieval should fall back to rule/lexical behavior with visible status instead of silently losing context.

## Verification

Verified against current repository files on 2026-07-14.
