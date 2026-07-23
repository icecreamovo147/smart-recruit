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
  - smart-recruit-ai-agent-service/internal/application/service/capability_service.go
  - platform-frontend/src/views/ai/SemanticRetrievalDebugView.vue
source_refs:
  - smart-recruit-ai-agent-service/internal/application/service/capability_service.go
  - smart-recruit-ai-agent-service/internal/infrastructure/provider/doc.go
  - smart-recruit-ai-agent-service/internal/domain/policy/capability.go
  - platform-frontend/src/views/ai/SemanticRetrievalDebugView.vue
last_verified: 2026-07-23
review_after: 2026-10-21
---

# Embedding Fallback Pitfall

Embedding outages, disabled provider/model config, dimension mismatches, or credential failures must degrade explicitly. Retrieval should fall back to rule/lexical behavior with visible status instead of silently losing context. `ResolveEmbeddingRuntime` marks unavailable/invalid embedding configuration with `FallbackUsed` and a concrete `FallbackReason` rather than pretending vectors remain available.

## Verification

Verified against embedding runtime resolution in capability policy/service and the Semantic Retrieval debug surface on 2026-07-23.
