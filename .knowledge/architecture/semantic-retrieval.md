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
  - smart-recruit-ai-agent-service/internal/domain/agentskill/ranking.go
  - smart-recruit-ai-agent-service/internal/application/memory/service.go
  - smart-recruit-ai-agent-service/internal/infrastructure/provider/embedding.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_memory.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_memory_runtime.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_runtime.go
  - platform-frontend/src/views/ai/SemanticRetrievalDebugView.vue
source_refs:
  - smart-recruit-ai-agent-service/internal/domain/agentskill/ranking.go
  - smart-recruit-ai-agent-service/internal/domain/agentskill/ranking_test.go
  - smart-recruit-ai-agent-service/internal/domain/memory/ranking.go
  - smart-recruit-ai-agent-service/internal/application/memory/service.go
  - smart-recruit-ai-agent-service/internal/infrastructure/provider/embedding.go
  - smart-recruit-ai-agent-service/internal/infrastructure/provider/embedding_test.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/config_store.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/config_store_agent_skill_embedding_test.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_runtime.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_runtime_test.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_memory_runtime.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/mcp_skill_services.go
  - platform-frontend/src/views/ai/SemanticRetrievalDebugView.vue
  - platform-frontend/src/api/memory.ts
  - smart-recruit-commons/config/config.example.yaml
last_verified: 2026-07-28
review_after: 2026-10-14
---

# Semantic Retrieval for Agent Skills and Memories

Semantic retrieval lives in AI Agent service. Agent Skill Package v2 and AI Memory recall both combine eligibility filters, lexical/vector signals, metadata boosts, business-boost gating, deterministic ordering, and context limits. Embedding failures degrade through explicit lexical+metadata fallback rather than silently returning empty context.

## Agent Skill Package v2 hierarchy

Agent Skill retrieval has two explicit levels:

1. `SearchAgentSkillVersions` ranks only immutable versions allowed by the active capability release. The catalog document uses manifest identity/description, agent type, category/scenario, risk, trigger keywords, semantic tags, required capabilities, and a bounded Core summary.
2. After composition selects exact Primary/Supporting versions, `SearchAgentSkillSections` ranks only sections whose `skill_version_id` is in that selected set. Section text uses key/title/description, trigger terms, semantic tags, planner intents, and content.

Both levels validate embedding object type, model, scope, text hash, and ready vector state. The runtime independently recompiles the persisted Package and checks compiled/section hashes before ranking or injection; a vector hit cannot authorize stale or altered content.

The shared ranker evaluates mode per candidate. Cosine is computed only when both the query vector and stored vector are non-empty and have exactly the same dimension. A candidate with a compatible vector uses vector 0.70, lexical 0.20, and metadata 0.10 in `hybrid` mode; a candidate without one uses lexical 0.70 and metadata 0.30 with `RelevanceMode=lexical_metadata` and zero vector score. A single version/section pool may therefore contain both modes. A 0.15 relevance gate runs before the bounded priority boost (maximum 0.10), so unrelated high-priority Packages/sections are excluded. Ties are deterministic by object identity.

When some stored vectors are empty or dimension-incompatible but at least one candidate has a compatible vector, the search keeps `EmbeddingAvailable=true`, preserves valid hybrid candidates, degrades only affected candidates, and reports a partial fallback reason. When no candidate has a compatible vector, the whole pool ranks lexical+metadata with `EmbeddingAvailable=false` and an all-fallback reason. Missing runner/config, provider failure, an empty query vector, or no ready model/hash-matching rows also produces an explicit whole-pool fallback. A dimension error in one stored embedding must not force otherwise valid candidates out of hybrid mode.

Runtime auto-selection applies the same local fallback if the embedding service itself is absent. The fallback reason remains visible in semantic diagnostics. Section loading consumes the remaining Skill budget in rank order and drops a whole section when it does not fit; it never truncates the immutable section.

## Memory pool (live)

The platform **Semantic Retrieval Debug** page runs real recall against both Skill and Memory pools. Memory results show scope, type, vector score, final rank score, importance, and relevance mode. A **Memory 校正** drawer supports list/create/revoke for platform debugging via gateway Memory APIs.

Runtime recall (`native_memory_runtime.go`) builds owner-scoped scopes, calls `embedding.SemanticMemoryScores` when a ready provider exists, and passes vector scores into `memory.Service.Recall`. When embeddings are unavailable, ranking uses lexical+metadata only (`RelevanceMode=fallback`).

## Ranking config (consumed)

Memory ranking reads `RankingConfigFromService(cfg.Ranking)` with defaults: vector 0.6, lexical 0.3, metadata 0.1, `relevance_gate` 0.15, `business_boost_max` 1.5. Business boost (importance/confidence) applies only when relevance ≥ gate. Zero-score candidates are dropped before truncation.

## ai_memory embeddings

`EmbeddingService.UpsertMemoryEmbedding` writes vectors for active memories on create/update. Backfill supports `ai_memory` entity type. Cleanup invalidates embeddings when memories are archived or purged. Semantic debug reports `memory_pool_confidence` (high/medium/low/none) based on top-gap heuristics shared with Skill pool.

## Verification

Verified against the Package v2 ranker and tests, per-candidate vector compatibility and partial/all fallback tests, version/section embedding persistence, hash- and release-scoped provider searches, HR two-level runtime selection/budget tests, memory ranking/runtime wiring, SemanticRetrievalDebugView, and embedding config defaults on 2026-07-28.
