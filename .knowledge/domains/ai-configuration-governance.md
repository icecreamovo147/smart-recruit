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
  - smart-recruit-gateway/handler/platform_ai.go
  - smart-recruit-gateway/router/router.go
  - platform-frontend/src/views/ai/**
source_refs:
  - smart-recruit-ai-agent-service/internal/domain/model/agent.go
  - smart-recruit-ai-agent-service/internal/domain/policy/agent.go
  - smart-recruit-ai-agent-service/internal/domain/policy/capability.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/config_services.go
  - smart-recruit-ai-agent-service/internal/application/service/agent_service.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/config_store.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/llm_runtime.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/llm_discovery.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/model_catalog.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/platform_ai_control_plane.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/platform_ai_release_guard.go
  - smart-recruit-commons/migrations/000070_add_platform_ai_control_plane.sql
  - smart-recruit-commons/migrations/000078_repair_hr_capability_prompt_releases.sql
  - smart-recruit-commons/migrations/000060_add_llm_model_catalog.sql
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/structured_runtime.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/recruiting_observability.go
  - smart-recruit-gateway/handler/hr/llm_config.go
  - smart-recruit-gateway/handler/hr/embedding_config.go
  - smart-recruit-gateway/handler/hr/prompt.go
  - smart-recruit-gateway/handler/hr/agent_config.go
  - smart-recruit-gateway/handler/platform_ai.go
  - smart-recruit-gateway/router/router.go
  - platform-frontend/src/views/ai/AICapabilityReleaseView.vue
last_verified: 2026-07-20
review_after: 2026-10-14
---

# AI Configuration Governance

AI configuration covers LLM providers/models, embedding providers/models, prompt templates and versions, agent configs, capability bindings, runtime policy, MCP policy, and Agent Skills. These technical resources are platform-global configuration owned by the platform console, not tenant-maintained resources. The enterprise workspace and candidate portal consume published capabilities but do not expose technical configuration CRUD. Preserve auditability, credential redaction, default uniqueness, prompt version history, rollback semantics, and platform permission checks.

The pre-launch one-time cutover promotes the default tenant's technical AI configuration to the platform baseline and removes tenant ownership from technical configuration tables. Runtime sessions, Runs, usage, tool logs, recruiting facts, and other business evidence remain tenant- or user-scoped. Migration `000070` must fail closed if non-default tenant technical configuration exists, because silently merging conflicting tenant configuration would make the promoted baseline ambiguous.

`platform_ai_capabilities` separates capability identity and audience (`tenant_hr` or `candidate`) from immutable release content. Draft snapshots may be edited; publishing validates the model pool plus every Agent, Prompt, Agent Skill version, AI Skill version, and MCP policy reference. Published snapshots are immutable, carry a SHA-256 hash, and are referenced by `*.release_version_id` plan or price entitlements. Referenced mutable configuration rows cannot be edited or deleted in place; administrators create a new configuration/version and then publish a new capability release. A sellable subscription price snapshot is generated on the server from the published platform plan or candidate capability release; the frontend is not an authority for entitlement JSON.

For Agent-backed releases (`ai.chat`, `ai.agent_run`, and `ai.application_analysis`), validation also enforces reference closure: every released Agent must bind an active compatible system Prompt, and that Prompt ID must be present in the same immutable release snapshot. Corrective data migrations append a replacement release and advance entitlement pointers instead of editing an already-published snapshot.

Enterprise and candidate assistants retain runtime model selection. Their `/ai/models` APIs first resolve the purchaser's fixed capability release, then return only the safe models in that release's audience-specific pool. A requested model outside the pool is rejected. If an allowed model later becomes unavailable, runtime may fall back only to that same release's default model and must report the requested model, effective model, fallback reason, release ID, and snapshot hash. It must not silently move the request to a newer release.

LLM model discovery uses the same canonical protocol and authentication strategy as inference; there is no independent discovery protocol. Provider list/detail responses are stored as field-level observations, then merged with an expiring, reviewed model catalog only for fields the provider did not supply. Each effective preset carries per-field provenance, unknown values remain unknown, and saving a model captures an editable user-owned snapshot rather than creating a live dependency on later catalog updates. Bundled catalog imports are versioned and idempotent, never overwrite administrator-managed rows, and expired observations are removed during synchronization.

The native AI Agent gRPC runtime now backs LLM, embedding, prompt, and agent configuration surfaces with `NativeStore` persistence instead of inherited unimplemented stubs. Live provider and embedding model tests validate persisted configuration and return explicit non-success unsupported/configuration responses when no provider client or worker is bound; secrets returned through configuration reads remain masked or redacted.

Structured recruiting operations invoked through the paid gateway carry the entitlement's fixed capability release into the AI service. The structured runtime resolves the effective model from that release and restricts Prompt lookup to the release's Prompt IDs; legacy internal calls without a release retain the exact `agent_type + system + is_active` compatibility path. Prompt content is not cached. Structured parse/match persistence records the requested/effective model, fallback reason, release ID, and snapshot hash. Governance diagnostics retain numeric Prompt ID/version and bounded, process-local HMAC correlations for the configured Prompt name and selected model; they never expose configured identity text, Prompt content, model response, provider error body, or credential material.

For HR AI chat, persisted Agent Tool/capability bindings are authorization allowlists, not advisory metadata. Existing configured Agents fail closed when concrete bindings are absent or disabled; the runtime must not synthesize the previous broad default recruiting Tool set. Only explicitly enabled, known, runtime-implemented builtin names may reach model Tool schemas or execution. MCP bindings also require explicit request selection before invocation.

The effective Agent's positive `max_iterations` is a real per-request Tool-loop control, not Prompt-only metadata. HR runtime clamps it to the service safety ceiling of 20 and passes it through the runtime provider into the shared AI client's additive per-call `MaxRounds` option; zero preserves the shared client default. The per-call option never mutates process-wide client configuration, so concurrently executing Agents retain independent limits and legacy callers keep existing behavior.

HR Agent Prompt bindings are valid only when the template is active, has the `system` role, and matches the Agent type. `hr_recruiting_agent` retains explicit read compatibility with the legacy `hr_agent` alias; blank or unrelated types are rejected. Agent create/update validates the binding, the admin selector applies the same compatibility filter, and runtime revalidates before use. Runtime substitution accepts only `hr_id`, `session_id`, `application_id`, `current_date`, and the bounded legacy context fields `context_line`, `summary_section`, and `memory_section`; unknown, unmatched, nested, or overlapping template expressions omit the affected Prompt and produce privacy-safe governance error evidence instead of reaching the model verbatim.

Durable Run governance uses the configuration that was effective when the Run was created: `agent_runs` stores the effective Agent ID/type/name and the requested/effective model, fallback reason, capability release ID, and snapshot hash. Successful `run.result` evidence stores numeric Prompt and Agent Skill version identities, bounded Tool names/statuses, and selection mode. Billable usage metadata carries the same release/model trace. This evidence deliberately excludes configuration bodies, Tool payloads, credentials, and personal recruiting data. New Runs resolve the release pinned by the purchaser's entitlement; already persisted Run identity and evidence remain stable for audit and replay.

## Verification

Verified against the platform AI control-plane migration and services, platform route permissions, immutable release/model-pool tests, billing entitlement resolution, HR/candidate model-list contracts, cumulative runtime tests, and frontend type checks on 2026-07-20.
