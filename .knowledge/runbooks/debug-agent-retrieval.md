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
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/mcp_skill_services.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/semantic_debug_scope.go
  - platform-frontend/src/views/ai/SemanticRetrievalDebugView.vue
source_refs:
  - smart-recruit-ai-agent-service/internal/domain/agentskill/ranking.go
  - smart-recruit-ai-agent-service/internal/infrastructure/provider/embedding.go
  - smart-recruit-ai-agent-service/internal/infrastructure/provider/embedding_test.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_runtime.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_runtime_test.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/mcp_skill_services.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_memory_handlers_test.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/semantic_debug_scope.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/platform_ai_control_plane_test.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/platform_ai_control_plane.go
  - platform-frontend/src/views/ai/SemanticRetrievalDebugView.vue
last_verified: 2026-07-30
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

The Semantic Retrieval debug page does not query a global Skill version pool. With the embedding runtime bound, blank or `hr_recruiting_agent` resolves the current published `ai.chat` release for audience `tenant_hr`; `candidate_assistant` resolves audience `candidate`. Only that release's exact `agent_skill_version_ids` are passed to the provider search.

Interpret scope edge cases as follows:

- An empty release allowlist returns zero Skill candidates with `published capability release contains no Agent Skill versions`; it never falls back to the global catalog. A valid owner-scoped Memory search still runs.
- An unsupported `agent_type`, missing published release, or release resolution failure produces no Skill candidates and reports `runtime Agent Skill release scope is unavailable`; the service still attempts the independent Memory search when its embedding runtime is available.
- If the store cannot provide the release-scope resolver, the request returns unavailable instead of performing an unscoped Skill query. If no embedding runtime is bound, the persistence fallback reports that the embedding query runner is unavailable.
- Memory results are a separate owner-scoped pool. HR requests require a valid tenant/HR owner and use HR plus requested application/job scopes; candidate owners use user plus requested application/job scopes. Owner/tenant validation happens before retrieval, so a forged or incomplete owner cannot widen the Memory pool.

A real runtime still adds capability eligibility, Primary/Supporting composition, risk confirmation, whole-section injection, and context-budget decisions that the debug result does not simulate. Reproduce runtime issues with `native_agent_skill_runtime_test.go`; use control-plane/provider tests for release resolution, embedding scope/hash, empty-allowlist, and fallback behavior.

## Verification

Verified against the semantic debug release-scope resolver, gRPC owner/scope validation and fallback branches, release-scoped provider searches, empty-allowlist tests, Package v2 ranker mixed-mode and partial/all vector fallback tests, exact-release runtime selection, normalized composition mismatch handling, budget evidence, capability snapshot policy, focused runtime/provider tests, and `SemanticRetrievalDebugView` on 2026-07-30.
