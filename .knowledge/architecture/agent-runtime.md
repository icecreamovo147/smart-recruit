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
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/config_services.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/mcp_skill_services.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_store.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/config_store.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/mcp_skill_store.go
  - smart-recruit-ai-agent-service/internal/infrastructure/provider/doc.go
  - smart-recruit-commons/ai/fallback.go
  - hr-frontend/src/views/hr/AIChatView.vue
last_verified: 2026-07-14
review_after: 2026-10-14
---

# Agent Runtime Architecture

AI Agent runtime is owned by `smart-recruit-ai-agent-service/`. The current implementation combines domain/application capability models with native runtime, gRPC, provider, and persistence adapters for HR AI chat, candidate AI chat, tool traces, context assembly, memory, embedding, MCP, Agent Skills, and durable agent runs. Shared AI client/fallback/tool support lives in `smart-recruit-commons/ai/`.

Runtime-facing configuration services for LLM, prompt, agent, and embedding config use focused optional store capabilities layered on the native `AIStore`. This keeps chat/runtime persistence contracts stable while exposing DB-backed configuration management and explicit non-success responses for unavailable stores, unsupported live provider tests, and unconfigured embedding backfills.

MCP, Skill registry, and Agent Skill admin services use the same native runtime pattern. Schema-backed server, policy, log, skill, version, tool, and Agent Skill management paths are DB-backed; MCP network/command execution and semantic debug paths return explicit non-success unsupported responses unless a runtime runner is bound.

Native AI chat/session/agent-run methods require a configured store for database-backed behavior and a configured provider for model-backed behavior. Missing dependencies must return explicit failures rather than synthetic sessions, empty lists, fallback runs, or provider placeholder text.

## Verification

Verified against current repository files on 2026-07-14.
