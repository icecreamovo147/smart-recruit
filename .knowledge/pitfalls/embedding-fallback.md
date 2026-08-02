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
  - smart-recruit-ai-agent-service/internal/domain/agentskill/ranking.go
  - smart-recruit-ai-agent-service/internal/infrastructure/provider/embedding.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_runtime.go
  - platform-frontend/src/views/ai/SemanticRetrievalDebugView.vue
source_refs:
  - smart-recruit-ai-agent-service/internal/domain/agentskill/ranking.go
  - smart-recruit-ai-agent-service/internal/domain/agentskill/ranking_test.go
  - smart-recruit-ai-agent-service/internal/infrastructure/provider/embedding.go
  - smart-recruit-ai-agent-service/internal/infrastructure/provider/embedding_test.go
  - smart-recruit-ai-agent-service/internal/infrastructure/provider/doc.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_runtime.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_runtime_test.go
  - platform-frontend/src/views/ai/SemanticRetrievalDebugView.vue
last_verified: 2026-07-28
review_after: 2026-10-21
---

# Embedding Fallback Pitfall

Embedding degradation is candidate-aware, not an unconditional pool-wide switch. Version and section searches compute cosine only when the query and stored vectors are both non-empty and have exactly equal dimensions. Each compatible candidate uses `hybrid`; each candidate without a compatible stored vector uses `vector_score=0` and `relevance_mode=lexical_metadata`. Consequently, one result pool may legitimately contain both modes.

If some stored vectors are empty or dimension-incompatible while others remain compatible, valid candidates keep their vector scores, `EmbeddingAvailable` remains true, and diagnostics report that some version/section embeddings were ignored. Only when no compatible vector remains does the entire pool use lexical+metadata with `EmbeddingAvailable=false` and an all-fallback reason. Missing provider/model config, provider errors, an empty query vector, or no ready current-model/text-hash rows also causes explicit whole-pool fallback. Do not diagnose one corrupt stored dimension as proof that every candidate globally degraded.

The 0.15 relevance gate still applies in either mode, and priority boost is bounded and applied only after the gate; fallback must not admit an unrelated high-priority Package.

Fallback never widens scope. Version candidates remain restricted to the active release's exact version IDs, and section candidates remain restricted to the already selected versions. It also does not bypass Package/hash integrity, required capabilities, composition, risk confirmation, output contract, or token budgets. When the entire embedding service is absent, the HR runtime uses the same local ranker rather than reverting to v1 selection.

## Verification

Verified against Package v2 ranking constants/tests, non-empty equal-dimension cosine checks, mixed-mode and partial/all version/section fallback tests, scope/hash tests, local HR runtime fallback tests, and the Semantic Retrieval debug surface on 2026-07-28.
