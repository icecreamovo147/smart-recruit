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
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/structured_runtime.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/resume_profile.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/job_requirement.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/candidate_match.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/recruiting_observability.go
  - smart-recruit-commons/ai/fallback.go
  - hr-frontend/src/views/hr/AIChatView.vue
last_verified: 2026-07-15
review_after: 2026-10-14
---

# Agent Runtime Architecture

AI Agent runtime is owned by `smart-recruit-ai-agent-service/`. The current implementation combines domain/application capability models with native runtime, gRPC, provider, and persistence adapters for HR AI chat, candidate AI chat, tool traces, context assembly, memory, embedding, MCP, Agent Skills, and durable agent runs. Shared AI client/fallback/tool support lives in `smart-recruit-commons/ai/`.

Runtime-facing configuration services for LLM, prompt, agent, and embedding config use focused optional store capabilities layered on the native `AIStore`. This keeps chat/runtime persistence contracts stable while exposing DB-backed configuration management and explicit non-success responses for unavailable stores, unsupported live provider tests, and unconfigured embedding backfills.

MCP, Skill registry, and Agent Skill admin services use the same native runtime pattern. Schema-backed server, policy, log, skill, version, tool, and Agent Skill management paths are DB-backed; MCP network/command execution and semantic debug paths return explicit non-success unsupported responses unless a runtime runner is bound.

Native AI chat/session/agent-run methods require a configured store for database-backed behavior and a configured provider for model-backed behavior. Missing dependencies must return explicit failures rather than synthetic sessions, empty lists, fallback runs, or provider placeholder text.

Recruiting intelligence uses an internal structured runtime rather than the generic HR Markdown completion path. On every structured operation it loads the current active `system` Prompt for the exact `resume_profile_extractor`, `job_requirement_extractor`, or `candidate_match_evaluator` agent type, then sends distinct System and User messages through the shared structured provider controls. Resume extraction, job-requirement extraction, deterministic-first per-requirement matching, deterministic aggregation, and versioned persistence remain internal to the AI Agent service; public gRPC shapes are unchanged.

Recruiting stage diagnostics pass through an idempotent fail-closed normalization boundary before application observers and again before Zap logging. Every non-empty externally propagated request ID becomes a process-keyed, domain-separated HMAC correlation token; safe-looking syntax is never trusted and low-entropy values are not reversibly logged. Operation, resource type, stage, category, agent type, fallback, and outcome use explicit fixed mappings. Database Prompt/model identities become bounded domain-separated correlations after UTF-8/control validation and their 256/128-byte schema limits are recognized; only exact internal parser/scorer constants remain readable. Unknown classifications become `unknown`, invalid UTF-8/control identity text is omitted, and numeric resource/count/duration fields are bounded. Prompt bodies, User messages, resume/job text, raw model output, evidence content, error bodies, and credentials are not observability fields.

`ParseResumeProfile` and `EvaluateCandidateMatch` install one deferred method-boundary finalizer before request validation. Every return path therefore emits exactly one terminal outcome, including nil/invalid input, dependency and authorization failures, disabled or compatibility paths, source/generation/aggregation/timeout/persistence failures, fallback, and success. Intermediate success remains non-terminal, and overall success is not emitted until persistence or a compatibility read has succeeded.

## Verification

Verified against current repository files and focused recruiting runtime/privacy tests on 2026-07-15.
