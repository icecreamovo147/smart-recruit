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
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_runtime.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_confirmation.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_observability.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_observability_test.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_store.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/config_store.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/mcp_skill_store.go
  - smart-recruit-ai-agent-service/internal/infrastructure/provider/doc.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/structured_runtime.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/strict_output_contract.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/resume_profile.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/job_requirement.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/candidate_match.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/recruiting_observability.go
  - smart-recruit-commons/ai/fallback.go
  - smart-recruit-commons/ai/anthropic_chatmodel.go
  - smart-recruit-commons/ai/planner.go
  - smart-recruit-proto/proto/recruitment.proto
  - smart-recruit-platform-go/serviceconfig/config.go
  - smart-recruit-commons/migrations/000090_agent_skill_package_v2.sql
  - hr-frontend/src/views/hr/AIChatView.vue
  - hr-frontend/src/components/chat/ChatMessageList.vue
  - hr-frontend/src/components/hr/ai/agentRunChatFlow.ts
  - hr-frontend/src/api/ai.ts
  - hr-frontend/src/utils/hrAgentRunReducer.ts
last_verified: 2026-07-30
review_after: 2026-10-21
---

# Agent Runtime Architecture

AI Agent runtime is owned by `smart-recruit-ai-agent-service/`. The current implementation combines domain/application capability models with native runtime, gRPC, provider, and persistence adapters for HR AI chat, candidate AI chat, tool traces, context assembly, memory, embedding, MCP, Agent Skills, and durable agent runs. Shared AI client/fallback/tool support lives in `smart-recruit-commons/ai/`.

Runtime-facing configuration services for LLM, prompt, agent, and embedding config use focused optional store capabilities layered on the native `AIStore`. This keeps chat/runtime persistence contracts stable while exposing DB-backed configuration management and explicit non-success responses for unavailable stores, unsupported live provider tests, and unconfigured embedding backfills.

MCP and Agent Skill Package admin services use the same native runtime pattern. Schema-backed server, policy, log, Package registry/version/section, and Tool management paths are DB-backed; MCP network/command execution and semantic debug paths return explicit non-success unsupported responses unless the required runtime runner is bound. Retired generic AI Skill registry routes are not part of the active HTTP surface.

Native AI chat/session/agent-run methods require a configured store for database-backed behavior and a configured provider for model-backed behavior. Missing dependencies must return explicit failures rather than synthetic sessions, empty lists, fallback runs, or provider placeholder text.

Candidate chat treats pre-provider billing and capability failures as durable conversation outcomes: it persists the User message first, then stores a structured failed Assistant message before returning the transport error. Clients render that metadata as an actionable state card and can recover the same state after refresh; failed content is not added to provider-generated context as a successful answer.

HR model selection is session-scoped for the next turn while every persisted message keeps the model actually used for that turn. `PreviewChatContext` recompiles persisted conversation history for a requested model without provider or Tool execution and without summary generation, then persists the selected-model state and `model_preview` context snapshot. The frontend treats this snapshot as model-relative: switching models shows the new denominator immediately in a calculating state and accepts only a matching-model snapshot as the numerator. The actual run and post-turn snapshot continue through the same budget controller, preventing preview/runtime context-policy drift.

Application-analysis session creation persists and returns a canonical non-empty User message that explicitly requests resume-to-job match evaluation. Both HR analysis entry paths reuse that message, with a planner-recognizable frontend fallback for legacy responses. Durable Run creation rejects blank messages before governance loading or dispatch. The Anthropic Messages adapter also rejects System-only input locally, so it never sends `messages: null` or an empty conversation to the provider.

HR AI chat resolves builtin Tool schemas from the effective Agent's explicitly enabled concrete bindings. A configured Agent with empty, disabled-only, abstract-only, unknown, or unimplemented bindings receives no default recruiting Tool authority. Executor argument, authorization/not-found, unsupported, and downstream failures are classified as non-success errors and persist as error Tool Traces/Run Steps; JSON error payloads are not useful Tool evidence. Recruitment-owned facts continue to be read through Recruitment gRPC clients.

Live recruiting questions pass through the same deterministic planner before either a model-native Tool Calling provider or a completion-only provider may answer. The planner declares mandatory evidence groups for job inventory, application listing, candidate search/detail, requested analytics metric families, comparison/match, interview preparation, and offer support. Every group must have a matching successful Tool Trace; missing inputs, disabled/unavailable Tools, and failed calls short-circuit to a deterministic clarification or limitation response. Direct factual inventory/metric answers are rendered only from Tool results, including authoritative empty results, so model free text cannot bypass the evidence gate. Status-change action Tools are excluded from automatic schemas and execution until a separate explicit-confirmation transport exists.

Prompt and Agent Skill compilation is fail closed. HR Prompts must be active compatible system templates and render only the allowlisted stable runtime variables; malformed or unknown expressions omit the Prompt and create governance evidence. Agent Skill Package v2 runtime accepts only exact immutable versions from the active capability release. It recompiles persisted manifest/Core/sections and verifies canonical content, compiled hash, section hashes, and token estimates before use. An empty release list loads none, a manual `agent_skill_version_ids` value must belong to the allowlist, and non-release calls do not fall back to the registry's `current_version_id`. The old `skill_md`/`flow_json` whole-prompt path is not executable.

Allowed Packages are ranked hierarchically—version first, then sections scoped to the selected versions—and composed as at most one Primary followed by one Supporting. Core is mandatory and sections are optional whole units. The effective budget is the strictest of 3,000 tokens, 15% of the model input budget, and the immutable release policy; at most two Packages may be composed. Core/section budget drops and integrity failures are retained as content-free runtime evidence.

The `skill_package_v2` feature defaults off. When off, runtime loads no Package and emits `skill_v2_disabled` evidence rather than reviving v1. Bounded metrics cover retrieval, selection, tokens, budget drops, confirmation, and output validation. The optional default-off Agent Skill judge is asynchronous and non-blocking. It never receives response text or reversible response fragments. Its response input is a fixed-schema JSON structural profile containing only `schema_version`, response presence, bounded rune/line counts, truncation state, and coarse format (`empty`, `text`, `json`, or `markdown_list`). Evaluation criteria are separately redacted for PII and selected entity values, deduplicated, count-bounded, and length-bounded. Queue capacity and a ten-second worker deadline bound judge work; saturation, shutdown, missing runner, errors, and pass/fail remain observable through bounded labels.

Tool schemas and the actual model ToolRunner independently enforce the active Agent allowlist, so Prompt or Skill text cannot cause an unbound builtin or MCP Tool to reach Recruitment services; attempted calls become classified error Traces.

Durable HR Agent Runs snapshot the effective persisted Agent ID, type, and name when the Run is created. Every successful Run emits a privacy-safe `run.result` whose raw governance evidence contains Prompt ID/version, Agent Skill ID/version, Tool name/status, and selection mode without Prompt/Skill bodies, Tool arguments/results, or recruiting personal data. Model-generated text chunks map to `assistant.delta`; planning, context, fallback, and generating statuses without text remain `process.delta`. When chunks were emitted, completion persists the final answer snapshot without appending a duplicate full-answer delta; non-streaming answers retain one assistant delta so consumers do not lose content. Tool Traces remain linked to their Run and Run Step with matching success/error state.

### Suggested follow-up questions

Successful HR chat and Agent Run completions may return `suggested_questions` as a repeated string field on `ChatResponse`, `ChatStreamResponse`, and `AgentRunResultMetadata`. Generation extracts a model-only JSON block from the reply, strips that block from user-visible streaming deltas, and normalizes to exactly three questions. Normalization fails closed to planner fallbacks when count, length, uniqueness, high-PII classification, or candidate/job-name leakage checks fail. Clients must consume the wire field rather than parsing hidden markers from assistant text. Candidate chat uses a separate extraction path and must not share HR privacy filters or copy.

### Process snapshot persistence

Near successful Agent Run completion the runtime appends a `process.snapshot` event whose payload includes `snapshot_text`: a privacy-safe multi-line display summary built from plan intent, tool-trace outcomes, context usage, and fallback flags—not raw tool arguments, resume text, or provider payloads. `NativeStore.AppendAgentRunEvent` mirrors a present `snapshot_text` onto `agent_runs.process_text` for reload. Gateway exposes `snapshot_text` on Agent Run events; the HR reducer treats `process.snapshot` as a full replacement of process text, while `process.delta` may replace when `snapshot_text` is present or otherwise append `delta`/`event_message`. Older clients that only read `process_text` or ignore `snapshot_text` remain compatible as long as the mirrored column stays populated.

Agent Run failures that contain `insufficient_credits` map to a stable `error_type` of `insufficient_credits` and a fixed user-facing credit message; the transport must not surface provider or ledger internals.

HR-wide application and candidate aggregation remains inside the AI Agent service and uses existing Recruitment RPCs. It sorts and limits the HR inventory to 100 jobs, fetches with at most four workers and ten 100-row pages per job, then sorts by job ID/application ID and returns at most 5,000 rows. Context cancellation stops the aggregation. Some failed jobs produce successful rows plus `partial`, bounded `failed_job_count`, and non-sensitive warnings; an all-job failure is a non-nil Tool error and contributes no useful facts.

MCP pre-context execution requires explicit `capability_keys` intersected with enabled Agent MCP bindings. An empty selection executes no MCP tool. MCP approval is independent from Agent Skill selection: confirmation-required durable Runs pause, issue a short-lived random request bound to the exact capability and argument hash, and resume only after a matching explicit approval. Direct chat fails closed when it cannot host that durable confirmation exchange. The MCP runner and policy path enforce configured policy, argument redaction, and audit persistence before a selected call can succeed.

The HR client treats `waiting_confirmation` as an exclusive Run state. Its MCP
confirmation card exposes both explicit approval and explicit rejection;
rejection invokes the durable Run cancel command and does not submit an approval
payload. While a confirmation is pending, message composition is disabled so a
second Run cannot silently replace the session's active Run. The client carries
the server expiry into the bound approval payload, detects an expired request
before confirmation, cancels that Run, and asks the user to start a fresh
request. Refresh recovery rebuilds the same confirmation card from the durable
Run snapshot.

High/critical Agent Skill compositions require a separate durable confirmation before provider or Tool execution. The confirmation is bound to tenant, actor, Run, exact user-message identity/digest, capability release ID/hash, exact Package version IDs/compiled hashes, selection mode, role, risk, and expiry. Approval consumption is CAS-protected; durable pending/ready/claimed handoff, execution leases and fencing, recovery, and cancellation prevent duplicate dispatch or writes by a stale worker. This protocol never approves an MCP Tool, and an MCP payload never approves a Package.

Generic HR chat supports `none` and advisory output contracts. Advisory adds deterministic bounded system guidance, still permits natural language, and performs no validation, repair, retry, or rejection. A strict contract on this free-text path fails before provider or Tool execution.

Recruiting intelligence uses an internal structured runtime rather than the generic HR Markdown completion path. When v2 is enabled for the exact capability release, it resolves exactly one released Primary Package matching `resume_profile_extractor`, `job_requirement_extractor`, or `candidate_match_evaluator`. A strict contract may run unattended only for low/medium auto risk. It requires exactly one JSON value in the supported schema dialect, permits at most one repair call, bills both provider calls, and suppresses resume/job/match fallback after contract, validation, or downstream domain failure. Resume extraction, job-requirement extraction, deterministic-first per-requirement matching, deterministic aggregation, and versioned persistence remain internal to the AI Agent service.

Recruiting stage diagnostics pass through an idempotent fail-closed normalization boundary before application observers and again before Zap logging. Every non-empty externally propagated request ID becomes a process-keyed, domain-separated HMAC correlation token; safe-looking syntax is never trusted and low-entropy values are not reversibly logged. Operation, resource type, stage, category, agent type, fallback, and outcome use explicit fixed mappings. Database Prompt/model identities become bounded domain-separated correlations after UTF-8/control validation and their 256/128-byte schema limits are recognized; only exact internal parser/scorer constants remain readable. Unknown classifications become `unknown`, invalid UTF-8/control identity text is omitted, and numeric resource/count/duration fields are bounded. Prompt bodies, User messages, resume/job text, raw model output, evidence content, error bodies, and credentials are not observability fields.

`ParseResumeProfile` and `EvaluateCandidateMatch` install one deferred method-boundary finalizer before request validation. Every return path therefore emits exactly one terminal outcome, including nil/invalid input, dependency and authorization failures, disabled or compatibility paths, source/generation/aggregation/timeout/persistence failures, fallback, and success. Intermediate success remains non-terminal, and overall success is not emitted until persistence or a compatibility read has succeeded.

## Verification

Verified against current repository files and cumulative HR Tool, Package v2 exact release selection/composition/budget/hash tests, feature-off evidence and bounded metrics, the fixed-schema non-reversible Agent Skill judge profile and bounded criteria/queue/deadline tests, durable Agent Skill and MCP-independent confirmation tests, advisory/strict output and fallback-suppression tests, live-data evidence gate, durable Run, suggested-questions privacy filters, process.snapshot persistence, application-analysis message, Anthropic envelope, and recruiting runtime tests on 2026-07-30.
