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
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_store.go
  - smart-recruit-ai-agent-service/internal/infrastructure/provider/doc.go
  - smart-recruit-commons/ai/fallback.go
  - hr-frontend/src/views/hr/AIChatView.vue
last_verified: 2026-07-14
review_after: 2026-10-14
---

# Agent Runtime Architecture

AI Agent runtime is owned by `smart-recruit-ai-agent-service/`. The current implementation combines domain/application capability models with native runtime, gRPC, provider, and persistence adapters for HR AI chat, candidate AI chat, tool traces, context assembly, memory, embedding, MCP, Agent Skills, and durable agent runs. Shared AI client/fallback/tool support lives in `smart-recruit-commons/ai/`.

## Verification

Verified against current repository files on 2026-07-14.
