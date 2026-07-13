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
  - smart-recruit-ai-agent-service/internal/legacydomain/service/agent_context.go
  - smart-recruit-ai-agent-service/internal/legacydomain/repository/*memory*.go
  - smart-recruit-ai-agent-service/internal/legacydomain/repository/session_summary_repo.go
source_refs:
  - smart-recruit-ai-agent-service/internal/legacydomain/service/agent_context.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/candidate_agent_context.go
  - smart-recruit-ai-agent-service/internal/legacydomain/repository/memory_repo.go
  - smart-recruit-ai-agent-service/internal/legacydomain/repository/session_summary_repo.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/context_usage.go
  - smart-recruit-commons/migrations/000038_persist_chat_context_usage.sql
  - smart-recruit-commons/migrations/000045_add_ai_memory_importance.sql
last_verified: 2026-07-14
review_after: 2026-10-14
---

# Memory and Agent Context Domain

Agent context assembly combines recent messages, summaries, memories, selected skills, tool traces, and business records under configured limits. Keep candidate/staff data boundaries, prompt size limits, memory importance, context usage persistence, and fallback behavior intact.

## Verification

Verified against current repository files on 2026-07-14.
