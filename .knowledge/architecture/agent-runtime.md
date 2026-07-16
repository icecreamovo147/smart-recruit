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
  - smart-recruit-commons/ai/anthropic_chatmodel.go
  - hr-frontend/src/views/hr/AIChatView.vue
last_verified: 2026-07-16
review_after: 2026-10-14
---

# Agent Runtime Architecture

AI Agent runtime is owned by `smart-recruit-ai-agent-service/`. The current implementation combines domain/application capability models with native runtime, gRPC, provider, and persistence adapters for HR AI chat, candidate AI chat, tool traces, context assembly, memory, embedding, MCP, Agent Skills, and durable agent runs. Shared AI client/fallback/tool support lives in `smart-recruit-commons/ai/`.

Runtime-facing configuration services for LLM, prompt, agent, and embedding config use focused optional store capabilities layered on the native `AIStore`. This keeps chat/runtime persistence contracts stable while exposing DB-backed configuration management and explicit non-success responses for unavailable stores, unsupported live provider tests, and unconfigured embedding backfills.

MCP, Skill registry, and Agent Skill admin services use the same native runtime pattern. Schema-backed server, policy, log, skill, version, tool, and Agent Skill management paths are DB-backed; MCP network/command execution and semantic debug paths return explicit non-success unsupported responses unless a runtime runner is bound.

Native AI chat/session/agent-run methods require a configured store for database-backed behavior and a configured provider for model-backed behavior. Missing dependencies must return explicit failures rather than synthetic sessions, empty lists, fallback runs, or provider placeholder text.

Application-analysis session creation persists and returns a canonical non-empty User message that explicitly requests resume-to-job match evaluation. Both HR analysis entry paths reuse that message, with a planner-recognizable frontend fallback for legacy responses. Durable Run creation rejects blank messages before governance loading or dispatch. The Anthropic Messages adapter also rejects System-only input locally, so it never sends `messages: null` or an empty conversation to the provider.

HR AI chat resolves builtin Tool schemas from the effective Agent's explicitly enabled concrete bindings. A configured Agent with empty, disabled-only, abstract-only, unknown, or unimplemented bindings receives no default recruiting Tool authority. Executor argument, authorization/not-found, unsupported, and downstream failures are classified as non-success errors and persist as error Tool Traces/Run Steps; JSON error payloads are not useful Tool evidence. Recruitment-owned facts continue to be read through Recruitment gRPC clients.

Live recruiting questions pass through the same deterministic planner before either a model-native Tool Calling provider or a completion-only provider may answer. The planner declares mandatory evidence groups for job inventory, application listing, candidate search/detail, requested analytics metric families, comparison/match, interview preparation, and offer support. Every group must have a matching successful Tool Trace; missing inputs, disabled/unavailable Tools, and failed calls short-circuit to a deterministic clarification or limitation response. Direct factual inventory/metric answers are rendered only from Tool results, including authoritative empty results, so model free text cannot bypass the evidence gate. Status-change action Tools are excluded from automatic schemas and execution until a separate explicit-confirmation transport exists.

Prompt and Agent Skill compilation is fail closed. HR Prompts must be active compatible system templates and render only the four allowlisted stable runtime variables; malformed or unknown expressions omit the Prompt and create governance evidence. Selected Agent Skills contribute instructions only when their exact current version is valid and non-empty, and evidence records the exact version without persisting its body. Tool schemas and the actual model ToolRunner both enforce the active Agent allowlist, so Prompt or Skill text cannot cause an unbound builtin Tool to reach Recruitment services; attempted calls become classified error Traces.

Durable HR Agent Runs snapshot the effective persisted Agent ID, type, and name when the Run is created. Every successful Run emits a privacy-safe `run.result` whose raw governance evidence contains Prompt ID/version, Agent Skill ID/version, Tool name/status, and selection mode without Prompt/Skill bodies, Tool arguments/results, or recruiting personal data. Model-generated text chunks map to `assistant.delta`; planning, context, fallback, and generating statuses without text remain `process.delta`. When chunks were emitted, completion persists the final answer snapshot without appending a duplicate full-answer delta; non-streaming answers retain one assistant delta so consumers do not lose content. Tool Traces remain linked to their Run and Run Step with matching success/error state.

HR-wide application and candidate aggregation remains inside the AI Agent service and uses existing Recruitment RPCs. It sorts and limits the HR inventory to 100 jobs, fetches with at most four workers and ten 100-row pages per job, then sorts by job ID/application ID and returns at most 5,000 rows. Context cancellation stops the aggregation. Some failed jobs produce successful rows plus `partial`, bounded `failed_job_count`, and non-sensitive warnings; an all-job failure is a non-nil Tool error and contributes no useful facts.

MCP pre-context execution requires an explicit `skill_capability_keys` selection intersected with enabled Agent MCP bindings. An empty selection executes no MCP tool. The MCP runner and policy path enforce configured policy, confirmation, argument redaction, and audit persistence before a selected call can succeed.

Recruiting intelligence uses an internal structured runtime rather than the generic HR Markdown completion path. On every structured operation it loads the current active `system` Prompt for the exact `resume_profile_extractor`, `job_requirement_extractor`, or `candidate_match_evaluator` agent type, then sends distinct System and User messages through the shared structured provider controls. Resume extraction, job-requirement extraction, deterministic-first per-requirement matching, deterministic aggregation, and versioned persistence remain internal to the AI Agent service; public gRPC shapes are unchanged.

Recruiting stage diagnostics pass through an idempotent fail-closed normalization boundary before application observers and again before Zap logging. Every non-empty externally propagated request ID becomes a process-keyed, domain-separated HMAC correlation token; safe-looking syntax is never trusted and low-entropy values are not reversibly logged. Operation, resource type, stage, category, agent type, fallback, and outcome use explicit fixed mappings. Database Prompt/model identities become bounded domain-separated correlations after UTF-8/control validation and their 256/128-byte schema limits are recognized; only exact internal parser/scorer constants remain readable. Unknown classifications become `unknown`, invalid UTF-8/control identity text is omitted, and numeric resource/count/duration fields are bounded. Prompt bodies, User messages, resume/job text, raw model output, evidence content, error bodies, and credentials are not observability fields.

`ParseResumeProfile` and `EvaluateCandidateMatch` install one deferred method-boundary finalizer before request validation. Every return path therefore emits exactly one terminal outcome, including nil/invalid input, dependency and authorization failures, disabled or compatibility paths, source/generation/aggregation/timeout/persistence failures, fallback, and success. Intermediate success remains non-terminal, and overall success is not emitted until persistence or a compatibility read has succeeded.

## Verification

Verified against current repository files and cumulative HR Tool, bounded aggregation, MCP, live-data evidence-gate, Prompt/Skill, durable Run, application-analysis message, Anthropic envelope, and recruiting runtime tests on 2026-07-16.
