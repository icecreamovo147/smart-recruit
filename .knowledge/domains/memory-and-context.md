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
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/hr_context_budget.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_store.go
  - smart-recruit-proto/proto/recruitment.proto
  - smart-recruit-gateway/handler/hr/ai.go
  - hr-frontend/src/components/chat/ChatComposer.vue
  - hr-frontend/src/utils/contextUsage.ts
source_refs:
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/hr_context_budget.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/hr_context_budget_test.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/context_usage_test.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_store.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_session_summary.go
  - smart-recruit-commons/migrations/000038_persist_chat_context_usage.sql
  - smart-recruit-commons/migrations/000045_add_ai_memory_importance.sql
  - smart-recruit-proto/proto/recruitment.proto
  - smart-recruit-gateway/handler/hr/ai.go
  - hr-frontend/src/components/chat/ChatComposer.vue
  - hr-frontend/src/utils/contextUsage.ts
last_verified: 2026-07-18
review_after: 2026-10-14
---

# Memory and Agent Context Domain

Agent context assembly combines recent messages, summaries, memories, selected skills, tool traces, and business records under configured limits. Keep candidate/staff data boundaries, prompt size limits, memory importance, context usage persistence, and fallback behavior intact.

HR chat reports the current effective provider input, not cumulative session billing. Provider-reported prompt usage from the latest model call is authoritative when available; otherwise the runtime retains a conservative estimate of the exact message/tool envelope sent to the provider. The estimate counts non-ASCII runes conservatively and includes tool schemas and protocol framing. Context snapshots are persisted on assistant messages and as the session's latest snapshot; cumulative billing usage remains audit data.

For a configured model, context governance uses `W` (context window), `O` (maximum output reservation), `S` (safety margin), and `B` (available input budget): `S = clamp(5% of W, 256, 2048)` and `B = W - O - S`. The normal target is 75% of `B`. Assembly preserves fixed system/current-call content, prefers the newest history that fits, restores chronological order, and can apply a rolling summary for covered older messages. At 60% of `B` it requests summary refresh asynchronously; an already over-budget envelope attempts synchronous refresh and then trims history. Summary failure degrades to safe trimming. If fixed content alone exceeds `B`, or `W/O/S` is invalid, the provider is not called and transports expose `AI_CONTEXT_BUDGET_EXCEEDED` or `AI_CONTEXT_CONFIGURATION_INVALID`.

When the model context window is unknown, the runtime does not invent a window or ratio. It applies an existing summary when present and retains at most the most recent 20 history messages plus the current fixed envelope. The HR composer therefore displays `current effective input / available input budget` when `B` is known, while its detail popover retains `W`, `O`, `S`, source/stage, breakdown, included/omitted counts, and summary state. Unknown configuration displays an unavailable denominator and explains the summary-plus-recent-message fallback.

## Verification

Verified against the context budget controller, persisted snapshots and summaries, additive protobuf/gateway mappings, HR composer utilities, full module tests, focused race tests, and repeated context tests on 2026-07-18.
