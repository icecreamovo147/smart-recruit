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
  - smart-recruit-ai-agent-service/internal/domain/agentskill/compiler.go
  - smart-recruit-platform-go/serviceconfig/config.go
  - smart-recruit-commons/migrations/archive/pre-baseline-000089/000070_add_platform_ai_control_plane.sql
  - smart-recruit-commons/migrations/000090_agent_skill_package_v2.sql
last_verified: 2026-07-28
review_after: 2026-10-14
---

# Debug AI Configuration Issues

Start in the platform console and verify `platform.ai.*` permissions and `/api/v1/platform/ai/**` routes. Technical configuration writes under `/api/v1/hr/admin/**` are intentionally absent after the one-time cutover. Then verify AI Agent configuration validation, default uniqueness, encrypted credential handling, prompt versioning, rollback behavior, embedding fallback, MCP transport policy, and platform frontend payload shape.

For an empty assistant model selector, inspect in order: the owner's `ai.<capability>.enabled` entitlement, its `ai.<capability>.release_version_id`, the matching capability audience, the release status/hash, the release model pool/default, and current model/provider enablement. Do not fix the symptom by returning all global models. For a fallback report, compare requested/effective model IDs and confirm the effective model is the default from the same release ID.

For an Agent Skill Package issue, inspect `AGENT_FEATURE_SKILL_PACKAGE_V2` first, then the exact release snapshot's `agent_skill_version_ids`, `skill_runtime_policy`, evaluation suite/result hashes, and snapshot hash. Recompile the listed immutable Package and compare manifest/Core/section canonical content, hashes, token estimates, Agent/capability compatibility, and composition. Do not repair execution by activating a registry current version: runtime never substitutes `current_version_id` for a missing release version. A feature-off run should show `skill_v2_disabled`; an empty allowlist should load none.

For high/critical Packages, use a durable Run and inspect the Agent-Skill-specific confirmation ID, exact version/hash/message/release binding, expiry, CAS handoff, lease, and cancellation evidence. Do not use the MCP opaque confirmation payload. For output failures, distinguish advisory (guidance only), generic HR strict rejection before provider/Tool, and structured strict validation with at most one billed repair and no fallback after strict/domain failure.

## Verification

Verified against the platform AI control plane, Package v2 compiler/release evaluator, feature defaults, exact-version runtime and confirmation/output-contract tests, billing release resolution, gateway model/capability handlers, and HR/candidate runtime model resolution on 2026-07-28.
