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
last_verified: 2026-07-15
review_after: 2026-10-14
---

# AI Configuration Governance

AI configuration covers LLM providers/models, embedding providers/models, prompt templates and versions, agent configs, capability bindings, runtime policy, MCP policy, and Agent Skills. Preserve auditability, credential redaction, default uniqueness, prompt version history, rollback semantics, and admin permission checks.

The native AI Agent gRPC runtime now backs LLM, embedding, prompt, and agent configuration surfaces with `NativeStore` persistence instead of inherited unimplemented stubs. Live provider and embedding model tests validate persisted configuration and return explicit non-success unsupported/configuration responses when no provider client or worker is bound; secrets returned through configuration reads remain masked or redacted.

Structured recruiting operations resolve the active Prompt by exact `agent_type + system + is_active` predicates on every request, with deterministic latest-updated/latest-ID selection when multiple rows exist. Prompt content is not cached. Prompt activation therefore takes effect without process restart, while structured model clients may be reused by complete non-secret configuration fingerprint to preserve concurrency and circuit-breaker state. Governance diagnostics retain numeric Prompt ID/version and bounded, process-local HMAC correlations for the configured Prompt name and selected model; they never expose configured identity text, Prompt content, model response, provider error body, or credential material.

## Verification

Verified against current configuration stores, structured runtime, and prompt refresh/privacy tests on 2026-07-15.
