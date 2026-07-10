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
  - logic-grpc-service/service/agent_skill_selector.go
  - logic-grpc-service/service/agent_skill_service.go
  - logic-grpc-service/service/agent_context.go
  - logic-grpc-service/service/embedding_service.go
  - logic-grpc-service/repository/memory_repo.go
source_refs:
  - logic-grpc-service/service/agent_skill_selector.go
  - logic-grpc-service/service/agent_skill_service.go
  - logic-grpc-service/service/agent_context.go
  - logic-grpc-service/service/embedding_service.go
  - .spec/skill-memory-ranking/skill-memory-ranking-SDD.md
last_verified: 2026-07-10
review_after: 2026-10-08
---

# Semantic Retrieval for Agent Skills and Memories

Semantic retrieval is used as an input to Agent Skill ranking and memory recall. It is not the only ranking signal. Skill selection still filters by agent type, manual invocation, runtime capability, and request limits before ranking candidates. Memory recall scopes are derived from the current HR/session/application/job context and then ranked and trimmed by count and character budget.

`EmbeddingService` owns embedding object storage, provider selection, fallback status, vector validation, and search metadata. When embedding calls fail or are unavailable, callers must handle the unavailable state explicitly instead of pretending semantic scoring succeeded.

Embedding provider and model configuration is managed separately by `EmbeddingConfigService`. Configuration changes can rebuild the embedding provider, alter the default model, or trigger backfill. Retrieval callers should treat embedding availability, vector dimension, model name, and fallback reason as observable runtime state rather than static assumptions.

## Agent Skill Retrieval

- Manual `agent_skill_ids` are handled before automatic selection and do not receive automatic backfill.
- Automatic candidates are filtered by agent type, seen IDs, required capabilities, and max selection count.
- Semantic scores can enrich ranking, but rules and metadata still participate in final pool selection.
- Debug retrieval exposes selected skills, memories, confidence, provider, model, dimension, candidate count, and latency through the current debug API.

## Memory Retrieval

- Context builder reads recent messages, summaries, prompt template, and scoped memory candidates.
- Memory candidates are ranked, capped by configured count, and trimmed by total character budget.
- Debug memory ranking should remain request-local; shared mutable debug state is a concurrency risk.

## Impact Guidance

- If `EmbeddingService.Search`, Agent Skill ranking, memory ranking, or debug retrieval changes, run knowledge impact detection for Skill and Memory routes.
- If embedding provider/model admin behavior changes, review `ai-configuration-governance` and `debug-ai-configuration`.
- If embedding fallback behavior changes, also review `pitfalls/embedding-fallback.md` after TASK-004 exists.
- If protobuf debug response fields change, treat it as a public contract change.

## Verification

This document was verified from current selector, debug, context builder, embedding service, and the active Skill/Memory ranking SDD on 2026-07-10.
