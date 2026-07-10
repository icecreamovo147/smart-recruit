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
  - logic-grpc-service/ai/**
  - logic-grpc-service/service/ai_service.go
  - logic-grpc-service/service/agent_context.go
  - logic-grpc-service/service/agent_run_recorder.go
  - hr-frontend/src/views/hr/AIChatView.vue
source_refs:
  - README.md
  - logic-grpc-service/service/ai_service.go
  - logic-grpc-service/service/agent_context.go
  - logic-grpc-service/service/agent_run_recorder.go
  - logic-grpc-service/ai/adk_agent.go
last_verified: 2026-07-10
review_after: 2026-10-08
---

# Agent Runtime Architecture

Smart Recruit has an HR AI assistant and candidate AI assistant backed by logic-service AI orchestration. The runtime uses an ADK-style path and a legacy path controlled by configuration. The Agent context builder assembles recent messages, session summary, active system prompt template, long-term memories, and prompt budget estimates for a request.

Agent runtime behavior belongs in `logic-grpc-service/service/` and `logic-grpc-service/ai/`. Frontend chat views and gateway handlers should pass request state, render stream events, and expose trace or debug information, but they should not decide core runtime selection, memory ranking, or tool execution semantics.

## Runtime Inputs

- Session and current message state from chat repositories.
- Session summaries and long-term memories from repository-backed context layers.
- Active prompt templates for the HR agent.
- Available Agent Skills and runtime capabilities.
- Embedding-backed semantic scores when the embedding provider is available.

## Runtime Outputs

- Streamed chat events for frontend clients.
- Persisted chat history and run trace records.
- Debug metadata for retrieval, ranking, embedding provider, and pool confidence where exposed by current APIs.

## Impact Guidance

- Changes to context assembly should check memory and prompt budget tests.
- Changes to runtime path selection should check config defaults and both ADK and legacy behavior when present.
- Changes to trace recording should check agent run recorder tests and HR trace UI expectations.
- Changes to Agent Skill selection or semantic retrieval should also review `semantic-retrieval` and domain knowledge for Skill and Memory.

## Verification

This document was verified against `logic-grpc-service/service/agent_context.go`, `logic-grpc-service/service/ai_service.go`, `logic-grpc-service/service/agent_run_recorder.go`, and `logic-grpc-service/ai/adk_agent.go` on 2026-07-10.
