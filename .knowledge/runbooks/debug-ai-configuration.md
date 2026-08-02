---
schema_version: 1
id: debug-ai-configuration
title: Debug AI configuration issues
kind: runbook
status: active
owners:
  - agent-platform
tags:
  - ai
  - configuration
  - debug
applies_to:
  - smart-recruit-ai-agent-service/**
  - smart-recruit-gateway/handler/platform_ai.go
  - platform-frontend/src/views/ai/**
source_refs:
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go
  - smart-recruit-ai-agent-service/internal/application/service/agent_service.go
  - smart-recruit-ai-agent-service/internal/runtime/runtime.go
  - smart-recruit-gateway/router/router.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/platform_ai_control_plane.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/platform_ai_agent_skill_release.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/platform_ai_embedding_readiness.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/embedding_outbox.go
  - smart-recruit-ai-agent-service/internal/infrastructure/embeddingqueue/consumer.go
  - smart-recruit-ai-agent-service/internal/infrastructure/embeddingqueue/store.go
  - smart-recruit-worker-service/internal/outbox/dispatcher.go
  - smart-recruit-commons/mq/consumer.go
  - smart-recruit-ai-agent-service/internal/domain/agentskill/compiler.go
  - smart-recruit-platform-go/serviceconfig/config.go
  - smart-recruit-commons/migrations/archive/pre-baseline-000089/000070_add_platform_ai_control_plane.sql
  - smart-recruit-commons/migrations/000090_agent_skill_package_v2.sql
last_verified: 2026-07-30
review_after: 2026-10-14
---

# Debug AI Configuration Issues

Start in the platform console and verify `platform.ai.*` permissions and `/api/v1/platform/ai/**` routes. Technical configuration writes under `/api/v1/hr/admin/**` are intentionally absent after the one-time cutover. Then verify AI Agent configuration validation, default uniqueness, encrypted credential handling, prompt versioning, rollback behavior, embedding fallback, MCP transport policy, and platform frontend payload shape.

For an empty assistant model selector, inspect in order: the owner's `ai.<capability>.enabled` entitlement, its `ai.<capability>.release_version_id`, the matching capability audience, the release status/hash, the release model pool/default, and current model/provider enablement. Do not fix the symptom by returning all global models. For a fallback report, compare requested/effective model IDs and confirm the effective model is the default from the same release ID.

For an Agent Skill Package issue, inspect `AGENT_FEATURE_SKILL_PACKAGE_V2` first, then the exact release snapshot's `agent_skill_version_ids`, `skill_runtime_policy`, evaluation suite/result hashes, and snapshot hash. Recompile the listed immutable Package and compare manifest/Core/section canonical content, hashes, token estimates, Agent/capability compatibility, and composition. Do not repair execution by activating a registry current version: runtime never substitutes `current_version_id` for a missing release version. A feature-off run should show `skill_v2_disabled`; an empty allowlist should load none.

When Package publication reports embedding not ready, diagnose the asynchronous chain before the evaluator:

1. Confirm Package/version creation committed the corresponding `embedding.upsert` `event_outbox` row whose aggregate/object is the immutable version and whose idempotency key includes the version ID and compiled hash.
2. Distinguish a pending/retrying/dead outbox row from a published row. The Worker outbox dispatcher owns bounded publish retries and the outbox dead state.
3. For a published event, inspect the `ai-agent-embedding-consumer` checkpoint by `(consumer_name, event_id)`. `processed` means duplicate delivery is ignored; `failed` means the next broker delivery may reclaim it. Check the RabbitMQ retry header and embedding queue DLQ separately from the database outbox state.
4. Check enabled/default release embedding model resolution and provider availability, then verify one ready version row and ready rows for every persisted section. Compare object type, `scope_type=agent_skill_version`, version scope ID, model name, compiled-hash metadata, declared dimension, decoded vector length, and non-empty vector.
5. Only after readiness passes, investigate evaluator availability, suite/result hashes, and case failures. Runtime lexical fallback is not evidence that publication readiness should pass.

Do not replay by editing `event_inbox` to processed, copying payloads into knowledge or tickets, or publishing a hand-written replacement with a new identity. Preserve the original event identity/idempotency and use the governed outbox/retry/DLQ path.

For high/critical Packages, use a durable Run and inspect the Agent-Skill-specific confirmation ID, exact version/hash/message/release binding, expiry, CAS handoff, lease, and cancellation evidence. Do not use the MCP opaque confirmation payload. For output failures, distinguish advisory (guidance only), generic HR strict rejection before provider/Tool, and structured strict validation with at most one billed repair and no fallback after strict/domain failure.

## Verification

Verified against the platform AI control plane, Package v2 compiler/release evaluator, transactional embedding outbox, Worker dispatcher, AI Agent inbox consumer, publication readiness gate, feature defaults, exact-version runtime and confirmation/output-contract tests, billing release resolution, gateway model/capability handlers, and HR/candidate runtime model resolution on 2026-07-30.
