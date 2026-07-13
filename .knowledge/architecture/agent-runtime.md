---
schema_version: 1
id: agent-runtime
title: Agent runtime architecture
kind: architecture
status: active
owners:
  - agent-platform
tags:
  - agent
  - runtime
  - ai
  - context
applies_to:
  - smart-recruit-ai-agent-service/**
  - smart-recruit-commons/ai/**
  - hr-frontend/src/views/hr/AIChatView.vue
source_refs:
  - smart-recruit-ai-agent-service/internal/runtime/runtime.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/legacy_servers.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/ai_agent_runtime.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/ai_service.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/candidate_ai_service.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/agent_run_recorder.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/agent_context.go
  - smart-recruit-ai-agent-service/internal/legacydomain/ai/adk_agent.go
  - smart-recruit-commons/ai/fallback.go
  - hr-frontend/src/views/hr/AIChatView.vue
last_verified: 2026-07-14
review_after: 2026-10-14
---

# Agent Runtime Architecture

AI Agent runtime is owned by `smart-recruit-ai-agent-service/`. The current implementation combines newer domain/application capability models with service-local legacy-domain adapters for HR AI chat, candidate AI chat, tool traces, context assembly, memory, embedding, MCP, Agent Skills, and durable agent runs. Shared AI client/fallback/tool support lives in `smart-recruit-commons/ai/`.

## Verification

Verified against current repository files on 2026-07-14.
