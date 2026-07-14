---
schema_version: 1
id: mcp-tool-governance
title: MCP tool governance
kind: domain
status: active
owners:
  - agent-platform
tags:
  - mcp
  - tool
  - policy
  - audit
  - agent
applies_to:
  - smart-recruit-ai-agent-service/**
  - smart-recruit-gateway/handler/hr/mcp.go
  - hr-frontend/src/views/hr/admin/McpManageView.vue
source_refs:
  - smart-recruit-ai-agent-service/internal/domain/model/capability.go
  - smart-recruit-ai-agent-service/internal/domain/policy/capability.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/mcp_skill_services.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/mcp_skill_store.go
  - smart-recruit-ai-agent-service/internal/infrastructure/mcp/doc.go
  - smart-recruit-gateway/handler/hr/mcp.go
  - smart-recruit-gateway/router/router.go
  - hr-frontend/src/views/hr/admin/McpManageView.vue
last_verified: 2026-07-14
review_after: 2026-10-14
---

# MCP Tool Governance

MCP governance covers server configuration, tool policy evaluation, policy APIs, logs, and admin surfaces. Keep deny/rate-limit/confirmation order, private-network blocking, stdio command allowlists, URL host constraints, argument redaction, and audit persistence aligned.

Native MCP governance now persists server CRUD, tool policy CRUD/list, and tool log list responses through the AI Agent database tables. Server env vars, MCP args, runtime config, log result payloads, policy details, and error strings must stay redacted or truncated on read paths. Live MCP connection tests, tool discovery, and tool execution are explicit non-success unsupported responses until a safe MCP runner is bound.

## Verification

Verified against current repository files on 2026-07-14.
