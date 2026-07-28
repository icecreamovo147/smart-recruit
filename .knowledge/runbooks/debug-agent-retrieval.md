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
  - smart-recruit-ai-agent-service/internal/domain/agentskill/ranking.go
  - smart-recruit-ai-agent-service/internal/infrastructure/provider/embedding.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_runtime.go
  - platform-frontend/src/views/ai/SemanticRetrievalDebugView.vue
source_refs:
  - smart-recruit-ai-agent-service/internal/domain/agentskill/ranking.go
  - smart-recruit-ai-agent-service/internal/infrastructure/provider/embedding.go
  - smart-recruit-ai-agent-service/internal/infrastructure/provider/embedding_test.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_runtime.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_runtime_test.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/platform_ai_control_plane.go
  - platform-frontend/src/views/ai/SemanticRetrievalDebugView.vue
last_verified: 2026-07-28
review_after: 2026-10-21
---

# Debug Agent Skill and Memory Retrieval

Diagnose Agent Skill Package v2 in this order:

1. Confirm `AGENT_FEATURE_SKILL_PACKAGE_V2=true`; when false, `skill_v2_disabled` evidence is expected and no v1 fallback exists.
2. Resolve the purchaser's exact capability release and inspect `configuration_refs.agent_skill_version_ids`, snapshot hash, `skill_runtime_policy`, and evaluation hashes. Empty version IDs mean no Package.
3. Recompile the listed manifest/Core/sections and compare `compiled_hash`, section hashes, and token estimates. Do not use `current_version_id` as a runtime substitute.
4. Check Agent type, required-capability eligibility, risk/activation policy, and the published `agent_type + scenario` composition group. At runtime, compare normalized agent type/scenario: automatic selection should record and skip an incompatible Supporting before trying the next candidate; a manual mismatch must fail closed with version-specific evidence.
5. Inspect version ranking first, then section ranking scoped to selected versions. Compare vector/lexical/metadata/relevance/boost/final score, per-candidate relevance mode, fallback reason, and deterministic pool rank. Cosine requires non-empty equal-dimension query/stored vectors. A partial dimension failure should leave compatible candidates in `hybrid` and only affected candidates in `lexical_metadata`; an all-fallback reason is expected only when no compatible vector remains.
6. Inspect runtime evidence for `core_included`, `section_included`, relevance, integrity, composition, confirmation, or budget drop reasons. Core/section bodies are intentionally absent.
7. Compare the effective Skill budget with release `max_skill_tokens`, `max_input_ratio`, model input budget, and the fixed two-Skill ceiling.

The Semantic Retrieval debug page can inspect the global version/Memory pools, but a real HR run additionally enforces the exact release allowlist, Agent/capability eligibility, composition, risk confirmation, and context budget. Reproduce runtime issues with `native_agent_skill_runtime_test.go`; use provider tests for embedding-scope/hash/fallback issues.

## Verification

Verified against Package v2 ranker/provider searches, mixed-mode and partial/all vector fallback tests, exact-release runtime selection, normalized composition mismatch handling, budget evidence, capability snapshot policy, focused runtime/provider tests, and `SemanticRetrievalDebugView` on 2026-07-28.
