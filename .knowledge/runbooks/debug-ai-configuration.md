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
  - smart-recruit-gateway/handler/hr/*config*.go
  - hr-frontend/src/views/hr/*ConfigView.vue
source_refs:
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go
  - smart-recruit-ai-agent-service/internal/application/service/agent_service.go
  - smart-recruit-ai-agent-service/internal/runtime/runtime.go
  - smart-recruit-gateway/router/router.go
last_verified: 2026-07-14
review_after: 2026-10-14
---

# Debug AI Configuration Issues

Verify gateway route permissions and HR admin handlers, then AI Agent config validation, default uniqueness, encrypted credential handling, prompt versioning, rollback behavior, embedding fallback, MCP transport policy, and frontend payload shape.

## Verification

Verified against current repository files on 2026-07-14.
