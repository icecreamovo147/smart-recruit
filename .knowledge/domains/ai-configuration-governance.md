---
schema_version: 1
id: ai-configuration-governance
title: AI configuration governance
kind: domain
status: active
owners:
  - agent-platform
tags:
  - ai
  - configuration
  - governance
  - model
applies_to:
  - smart-recruit-ai-agent-service/**
  - smart-recruit-gateway/handler/hr/*config*.go
  - hr-frontend/src/views/hr/*ConfigView.vue
  - hr-frontend/src/views/hr/PromptManageView.vue
source_refs:
  - smart-recruit-ai-agent-service/internal/domain/model/agent.go
  - smart-recruit-ai-agent-service/internal/domain/policy/agent.go
  - smart-recruit-ai-agent-service/internal/domain/policy/capability.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/config_services.go
  - smart-recruit-ai-agent-service/internal/application/service/agent_service.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/config_store.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/llm_runtime.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/structured_runtime.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/recruiting_observability.go
  - smart-recruit-gateway/handler/hr/llm_config.go
  - smart-recruit-gateway/handler/hr/embedding_config.go
  - smart-recruit-gateway/handler/hr/prompt.go
  - smart-recruit-gateway/handler/hr/agent_config.go
last_verified: 2026-07-16
review_after: 2026-10-14
---

# AI Configuration Governance

AI configuration covers LLM providers/models, embedding providers/models, prompt templates and versions, agent configs, capability bindings, runtime policy, MCP policy, and Agent Skills. Preserve auditability, credential redaction, default uniqueness, prompt version history, rollback semantics, and admin permission checks.

The native AI Agent gRPC runtime now backs LLM, embedding, prompt, and agent configuration surfaces with `NativeStore` persistence instead of inherited unimplemented stubs. Live provider and embedding model tests validate persisted configuration and return explicit non-success unsupported/configuration responses when no provider client or worker is bound; secrets returned through configuration reads remain masked or redacted.

Structured recruiting operations resolve the active Prompt by exact `agent_type + system + is_active` predicates on every request, with deterministic latest-updated/latest-ID selection when multiple rows exist. Prompt content is not cached. Prompt activation therefore takes effect without process restart, while structured model clients may be reused by complete non-secret configuration fingerprint to preserve concurrency and circuit-breaker state. Governance diagnostics retain numeric Prompt ID/version and bounded, process-local HMAC correlations for the configured Prompt name and selected model; they never expose configured identity text, Prompt content, model response, provider error body, or credential material.

For HR AI chat, persisted Agent Tool/capability bindings are authorization allowlists, not advisory metadata. Existing configured Agents fail closed when concrete bindings are absent or disabled; the runtime must not synthesize the previous broad default recruiting Tool set. Only explicitly enabled, known, runtime-implemented builtin names may reach model Tool schemas or execution. MCP bindings also require explicit request selection before invocation.

The effective Agent's positive `max_iterations` is a real per-request Tool-loop control, not Prompt-only metadata. HR runtime clamps it to the service safety ceiling of 20 and passes it through the runtime provider into the shared AI client's additive per-call `MaxRounds` option; zero preserves the shared client default. The per-call option never mutates process-wide client configuration, so concurrently executing Agents retain independent limits and legacy callers keep existing behavior.

HR Agent Prompt bindings are valid only when the template is active, has the `system` role, and matches the Agent type. `hr_recruiting_agent` retains explicit read compatibility with the legacy `hr_agent` alias; blank or unrelated types are rejected. Agent create/update validates the binding, the admin selector applies the same compatibility filter, and runtime revalidates before use. Runtime substitution accepts only `hr_id`, `session_id`, `application_id`, and `current_date`; unknown, unmatched, nested, or overlapping template expressions omit the affected Prompt and produce privacy-safe governance error evidence instead of reaching the model verbatim.

Durable Run governance uses the configuration that was effective when the Run was created: `agent_runs` stores the effective Agent ID/type/name, while successful `run.result` evidence stores numeric Prompt and Agent Skill version identities, bounded Tool names/statuses, and selection mode. This evidence deliberately excludes configuration bodies, Tool payloads, and personal recruiting data. New Runs resolve current configuration; already persisted Run identity and evidence remain stable for audit and replay.

## Verification

Verified against cumulative configuration stores, Prompt binding/rendering, Tool allowlist/evidence-gate/iteration-control, durable governance, bounded aggregation, structured runtime, and prompt refresh/privacy tests on 2026-07-16.
