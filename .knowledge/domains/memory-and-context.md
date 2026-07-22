---
schema_version: 1
id: memory-and-context
title: Memory and Agent context domain
kind: domain
status: active
owners:
  - agent-platform
tags:
  - memory
  - context
  - agent
applies_to:
  - smart-recruit-ai-agent-service/internal/application/memory/service.go
  - smart-recruit-ai-agent-service/internal/application/contextbudget/controller.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_memory_runtime.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/hr_context_budget.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_memory.go
  - smart-recruit-ai-agent-service/cmd/ai-agent-service/main.go
  - smart-recruit-proto/proto/recruitment.proto
  - smart-recruit-gateway/handler/hr/memory.go
  - smart-recruit-gateway/handler/candidate/memory.go
  - smart-recruit-commons/migrations/000083_ai_memories_owner_lifecycle.sql
source_refs:
  - smart-recruit-ai-agent-service/internal/application/memory/service.go
  - smart-recruit-ai-agent-service/internal/application/memory/config.go
  - smart-recruit-ai-agent-service/internal/domain/memory/ranking.go
  - smart-recruit-ai-agent-service/internal/domain/memory/pii.go
  - smart-recruit-ai-agent-service/internal/application/contextbudget/controller.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_memory_runtime.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/hr_context_budget.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_memory.go
  - smart-recruit-ai-agent-service/cmd/ai-agent-service/main.go
  - smart-recruit-proto/proto/recruitment.proto
  - smart-recruit-gateway/handler/hr/memory.go
  - smart-recruit-gateway/handler/candidate/memory.go
  - smart-recruit-commons/migrations/000083_ai_memories_owner_lifecycle.sql
  - smart-recruit-commons/config/config.example.yaml
last_verified: 2026-07-22
review_after: 2026-10-14
---

# Memory and Agent Context Domain

Agent context assembly combines recent messages, rolling summaries, long-term memories, Prompt/Agent/Skill governance, Tool schemas, and business records under a shared short-term context (STM) budget. HR and candidate chat both share the same `contextbudget` controller; memories are injected as a dedicated system section (`[Long-term memories]`) and metered separately in `ContextUsageInfo.memory_tokens`.

## Long-term memory (LTM)

**Owner model** (`000083`): each row has `owner_role` (1=candidate, 2=HR) and `owner_id`. Scopes are role-specific — HR uses `hr`, `application`, `job`, `candidate`; candidates use `user`, `application`, `job`. Recall and CRUD always filter by `(owner_role, owner_id)` so cross-owner isolation is enforced at persistence and service layers.

**Write path**: `memory.Service.Write` validates scope, classifies PII, rejects high-PII content unless `ConfirmHighPII`, deduplicates by normalized `content_hash`, and optionally upserts `ai_memory` embeddings. Post-turn extraction (`WriteFromExtractor`) runs asynchronously for HR and candidate chats when `write_enabled` is true.

**Recall path**: loads active, non-expired memories for owner+scopes, ranks with hybrid/lexical scoring (`RankingConfig` from service config), filters high-PII from injectables, truncates by `max_memories` / `max_memory_chars`, and formats inject text. Semantic vector scores are merged when embedding runtime is available; otherwise relevance mode falls back to lexical+metadata (`fallback`).

**Inject path**: HR injects via prompt variable `memory_section` and context-budget memory block; candidate injects via system prompt append. High-PII memories are never injected even if stored with explicit confirmation.

**Lifecycle**: statuses `active` → `archived` (TTL expiry) or `revoked` (soft delete). Background cleanup loop in `main.go` archives expired rows and purges revoked rows older than `revoked_retention`, invalidating embeddings.

## Feature flags (`agent.memory`)

| Flag | Default | Effect |
|------|---------|--------|
| `enabled` | true | Master switch for recall/inject/cleanup |
| `write_enabled` | true | Persist writes; false = dry-run |
| `inject_enabled` | true | Include recalled text in prompts |

Related limits: `max_memories` (10), `max_memory_chars` (1500). Cleanup: `cleanup_interval` 15m, `cleanup_timeout` 5m, `revoked_retention` 720h.

## PII policy

High PII (phone, email, ID card, salary, bank card patterns) is rejected on create unless confirmed. Stored high-PII rows are excluded from inject via `FilterInjectables`. Classification runs at write time and is persisted in `pii_level`.

## STM context budget (shared with memories)

For a configured model, `W` = context window, `O` = max output reservation, `S` = safety margin (`clamp(5% of W, 256, 2048)`), `B` = `W - O - S`. Assembly preserves fixed system/current-call content, prefers newest history, optionally applies rolling summary, and injects memory section when it fits. **Under tight budget, the memory section is dropped before history trimming** so fixed content never exceeds `B`. Unknown window falls back to summary + recent 20 messages.

HR chat persists `stage=post_turn` context snapshots including `memory_applied` and `memory_tokens` breakdown. Model-switch preview uses `stage=model_preview` without provider calls.

## Verification

Verified against memory domain/service/persistence tests, context-budget memory-drop tests, gateway Memory RPC handlers, migration `000083`, proto Memory RPCs, main wiring (`memoryService` + `runMemoryCleanupLoop`), and frontend typechecks on 2026-07-22.
