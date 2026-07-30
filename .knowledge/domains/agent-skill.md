---
schema_version: 1
id: agent-skill
title: Agent Skill domain
kind: domain
status: active
owners:
  - agent-platform
tags:
  - agent
  - skill
  - retrieval
applies_to:
  - smart-recruit-ai-agent-service/internal/domain/agentskill/**
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_*.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/*agent_skill*.go
  - smart-recruit-commons/migrations/000090_agent_skill_package_v2.sql
  - smart-recruit-proto/proto/recruitment.proto
  - hr-frontend/src/views/hr/AIChatView.vue
  - platform-frontend/src/views/ai/AgentSkillManageView.vue
source_refs:
  - smart-recruit-ai-agent-service/internal/domain/agentskill/types.go
  - smart-recruit-ai-agent-service/internal/domain/agentskill/compiler.go
  - smart-recruit-ai-agent-service/internal/domain/agentskill/ranking.go
  - smart-recruit-ai-agent-service/internal/domain/agentskill/strict_json_schema.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_runtime.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_confirmation.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_observability.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/strict_output_contract.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/mcp_skill_store.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/embedding_outbox.go
  - smart-recruit-ai-agent-service/internal/infrastructure/embeddingqueue/consumer.go
  - smart-recruit-ai-agent-service/internal/infrastructure/embeddingqueue/store.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/platform_ai_embedding_readiness.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/platform_ai_agent_skill_release.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/platform_ai_agent_skill_release_test.go
  - smart-recruit-ai-agent-service/internal/infrastructure/provider/embedding.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_runtime_test.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_confirmation_test.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_observability_test.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_output_contract_test.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/strict_output_contract_test.go
  - smart-recruit-commons/migrations/000090_agent_skill_package_v2.sql
  - smart-recruit-commons/migration/agent_skill_package_v2_migration_test.go
  - smart-recruit-proto/proto/recruitment.proto
  - smart-recruit-gateway/router/contract_baseline_test.go
  - hr-frontend/src/views/hr/AIChatView.vue
  - platform-frontend/src/views/ai/AgentSkillManageView.vue
  - scripts/check-agent-skill-v2-cutover.mjs
last_verified: 2026-07-30
review_after: 2026-10-21
---

# Agent Skill Domain

Agent Skill Package v2 is the only active Agent Skill contract. It is a pre-launch destructive cutover: migration `000090_agent_skill_package_v2.sql` retires the currently published tenant HR chat/run releases, clears the test-only Skill data and old embeddings, recreates the registry/version tables, and adds immutable version sections. There is no runtime compatibility path for v1 `skill_md`, `flow_json`, whole-prompt injection, numeric Skill IDs, or boolean Skill confirmation. Proto reserves retired field names/tags, gateway negative tests reject legacy JSON, and the repository cutover scanner permits old literals only in migrations, generated reserved descriptors, archives, or explicit rejection tests.

## Immutable package and registry

The canonical compiler accepts a schema-version-2 manifest, required Core Markdown, and at most 20 ordered reference sections. It normalizes sets and Markdown, derives activation policy from risk (`low|medium -> auto`, `high -> confirm`, `critical -> manual_only`), enforces composition and output-contract rules, and produces canonical manifest/Package JSON, deterministic compiled Markdown, SHA-256 `compiled_hash`, section content hashes, and conservative token estimates. Core is capped at 800 tokens, each section at 1,200, and the compiled artifact at 12,000. Supporting Skills cannot define output contracts.

`agent_skills` remains a mutable registry containing identity, enablement, manual-invocation policy, and `current_version_id` for platform administration and HR discovery. The immutable executable content lives in `agent_skill_versions` plus `agent_skill_version_sections`. Create, preview, and version-create use the same compiler; version and section persistence is transactional. Activation verifies version ownership. Published releases reference exact version IDs and cannot be changed by later registry activation.

Creating a Package or a new immutable version writes an `embedding.upsert` row to the shared `event_outbox` in the same database transaction as the version and sections. Its idempotency key binds the version ID and compiled hash. The Worker outbox dispatcher publishes the event; the AI Agent `ai-agent-embedding-consumer` claims `(consumer_name, event_id)` in `event_inbox`, then regenerates the version embedding and every section embedding. A successfully processed inbox row is not processed again. A processing failure marks the inbox row failed and returns an error so the shared RabbitMQ retry/DLQ policy can redeliver it; a failed inbox row may be claimed again, while processed or dead rows are not reclaimed.

## Release, retrieval, and composition

Runtime execution is release-only. `configuration_refs.agent_skill_version_ids` is an exact allowlist; an empty list loads no Package, unavailable or tampered listed versions fail closed, and a manually supplied `agent_skill_version_ids` value must be an allowed exact version. Non-release calls do not fall back to `current_version_id`. HR discovery may show an enabled/manual-invocable current version, but the actual message snapshot stores and submits its exact version ID.

Retrieval is hierarchical. The runtime first ranks allowed immutable versions, then ranks reference sections only within selected versions. Hybrid ranking uses vector, lexical, and metadata signals; candidates without a compatible stored vector use relevance-gated lexical+metadata ranking. Business priority is bounded and applies only after relevance passes the gate.

Publication is stricter than runtime retrieval fallback. Before the deterministic release evaluator runs, every referenced version and every persisted section must have a current `ready` embedding for the release-selected enabled model. Version rows require `object_type=agent_skill_version`, `scope_type=agent_skill_version`, and `scope_id=version_id`; section rows require `object_type=agent_skill_section`, the same scope type, and `scope_id=skill_version_id`. Each row must match the model, compiled hash, declared dimension, and non-empty vector length. Missing storage/model/configuration, a pending or failed row, a stale hash, a scope mismatch, or an invalid vector fails publication closed with `ErrAgentSkillEmbeddingNotReady`.

Release publication validates composition by each compiled manifest's `agent_type + scenario` group: a group admits at most one Primary plus one Supporting and never Supporting alone. HR runtime then compares Primary and Supporting by normalized agent type and scenario (trimmed, internal whitespace collapsed, and case-folded). Automatic selection records mismatch evidence, skips an incompatible Supporting, and continues to the next ranked Supporting; manual exact-version selection fails closed and records the rejected version/reason for cross-type or cross-scenario composition. A valid composition injects Primary Core before Supporting Core and whole sections.

Default runtime limits are two Skills, 3,000 Skill tokens, and 15% of the model input budget; a release snapshot may only make those limits stricter. Core or whole sections are dropped rather than truncated, and every include/drop/integrity decision is recorded as privacy-safe evidence with exact version/hash and section hashes, never content.

Required capabilities are eligibility constraints, not authority grants. `capability_keys` is the per-message data/Tool selection field and is intersected with the effective Agent bindings; Agent Skill instructions cannot expand builtin or MCP Tool authority.

## Risk, outputs, and observability

Low/medium Packages may activate automatically. High/critical compositions stop before any provider or Tool call and require the Agent-Skill-specific durable confirmation protocol; this is independent from MCP Tool confirmation. The approval is bound to tenant, user/HR, Run, exact user-message identity/digest, capability release ID/snapshot hash, exact version IDs and compiled hashes, selection mode, role, risk, and expiry. CAS consumption, durable handoff state, execution leases/fencing, cancellation, and recovery prevent replay or a stale worker from producing Tool/evidence/terminal writes.

An advisory output contract adds a bounded deterministic system instruction only: natural-language output remains valid and the runtime does not validate, repair, retry, or reject it. Generic HR chat rejects strict output contracts before provider/Tool execution. Structured recruiting agents may apply one exact released Primary strict contract only when it is low/medium auto risk; output must be one JSON value matching the supported fail-closed schema dialect. One repair call is allowed, both calls count toward billing, and a strict contract or downstream domain failure suppresses heuristic/deterministic fallback.

Both `skill_package_v2` and the optional asynchronous `agent_skill_judge` default off. Feature-off execution loads no Package and emits `skill_v2_disabled` evidence; it never revives v1. Bounded metrics cover retrieval, selection, loaded tokens, budget drops, confirmation, and output validation. The judge is non-blocking and never receives response text or reversible response fragments. It receives only a fixed JSON structural profile—schema version, response presence, bounded rune/line counts, truncation state, and coarse format—plus evaluation criteria that have been PII/entity-redacted, deduplicated, count-bounded, and length-bounded. Queue capacity and a ten-second worker deadline bound asynchronous evaluation.

## Verification

Verified against the canonical compiler/schema/ranker, transactional version/outbox persistence, AI Agent embedding consumer/inbox flow, publication embedding-readiness gate, Package v2 migration and migration tests, immutable persistence and grouped release composition evaluation, normalized HR runtime composition and mismatch evidence tests, hierarchical retrieval, durable Agent Skill confirmation, fixed-schema non-reversible judge input and bounded criteria/queue/deadline tests, advisory/strict output tests, Proto/gateway cutover contracts, HR exact-version payloads, platform Package editor, feature defaults/metrics, and the cutover scanner on 2026-07-30.
