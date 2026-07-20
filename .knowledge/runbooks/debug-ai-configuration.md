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
  - smart-recruit-commons/migrations/000070_add_platform_ai_control_plane.sql
last_verified: 2026-07-20
review_after: 2026-10-14
---

# Debug AI Configuration Issues

Start in the platform console and verify `platform.ai.*` permissions and `/api/v1/platform/ai/**` routes. Technical configuration writes under `/api/v1/hr/admin/**` are intentionally absent after the one-time cutover. Then verify AI Agent configuration validation, default uniqueness, encrypted credential handling, prompt versioning, rollback behavior, embedding fallback, MCP transport policy, and platform frontend payload shape.

For an empty assistant model selector, inspect in order: the owner's `ai.<capability>.enabled` entitlement, its `ai.<capability>.release_version_id`, the matching capability audience, the release status/hash, the release model pool/default, and current model/provider enablement. Do not fix the symptom by returning all global models. For a fallback report, compare requested/effective model IDs and confirm the effective model is the default from the same release ID.

## Verification

Verified against the platform AI control plane, billing release resolution, gateway model-list handlers, and HR/candidate runtime model resolution on 2026-07-20.
