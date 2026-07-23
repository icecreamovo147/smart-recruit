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
  - smart-recruit-ai-agent-service/internal/application/memory/service.go
  - smart-recruit-ai-agent-service/internal/infrastructure/provider/embedding.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_memory.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_memory_runtime.go
  - platform-frontend/src/views/ai/SemanticRetrievalDebugView.vue
source_refs:
  - smart-recruit-ai-agent-service/internal/domain/memory/ranking.go
  - smart-recruit-ai-agent-service/internal/application/service/capability_service.go
  - smart-recruit-ai-agent-service/internal/application/memory/service.go
  - smart-recruit-ai-agent-service/internal/infrastructure/provider/embedding.go
  - smart-recruit-ai-agent-service/internal/infrastructure/provider/embedding_test.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_memory_runtime.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/mcp_skill_services.go
  - platform-frontend/src/views/ai/SemanticRetrievalDebugView.vue
  - platform-frontend/src/api/memory.ts
  - smart-recruit-commons/config/config.example.yaml
last_verified: 2026-07-22
review_after: 2026-10-14
---

# Semantic Retrieval for Agent Skills and Memories

Semantic retrieval lives in AI Agent service. Agent Skill and AI Memory recall both combine eligibility filters, lexical/vector signals, metadata scope boosts, business boost gating, and context limits. Embedding failures degrade through explicit lexical+metadata fallback rather than silently returning empty context.

## Memory pool (live)

The platform **Semantic Retrieval Debug** page runs real recall against both Skill and Memory pools. Memory results show scope, type, vector score, final rank score, importance, and relevance mode. A **Memory 校正** drawer supports list/create/revoke for platform debugging via gateway Memory APIs.

Runtime recall (`native_memory_runtime.go`) builds owner-scoped scopes, calls `embedding.SemanticMemoryScores` when a ready provider exists, and passes vector scores into `memory.Service.Recall`. When embeddings are unavailable, ranking uses lexical+metadata only (`RelevanceMode=fallback`).

## Ranking config (consumed)

Memory ranking reads `RankingConfigFromService(cfg.Ranking)` with defaults: vector 0.6, lexical 0.3, metadata 0.1, `relevance_gate` 0.15, `business_boost_max` 1.5. Business boost (importance/confidence) applies only when relevance ≥ gate. Zero-score candidates are dropped before truncation.

## ai_memory embeddings

`EmbeddingService.UpsertMemoryEmbedding` writes vectors for active memories on create/update. Backfill supports `ai_memory` entity type. Cleanup invalidates embeddings when memories are archived or purged. Semantic debug reports `memory_pool_confidence` (high/medium/low/none) based on top-gap heuristics shared with Skill pool.

## Verification

Verified against embedding provider tests, memory ranking/domain tests, native memory runtime wiring, SemanticRetrievalDebugView Memory pool UI, and config defaults on 2026-07-22.
