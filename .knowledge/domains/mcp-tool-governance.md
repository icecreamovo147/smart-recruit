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
  - platform-frontend/src/views/ai/McpManageView.vue
source_refs:
  - smart-recruit-ai-agent-service/internal/domain/model/capability.go
  - smart-recruit-ai-agent-service/internal/domain/policy/capability.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/mcp_skill_services.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/mcp_skill_store.go
  - smart-recruit-ai-agent-service/internal/infrastructure/mcp/doc.go
  - smart-recruit-gateway/handler/hr/mcp.go
  - smart-recruit-gateway/router/router.go
  - hr-frontend/src/components/chat/ChatMessageList.vue
  - hr-frontend/src/components/hr/ai/agentRunChatFlow.ts
  - hr-frontend/src/views/hr/AIChatView.vue
  - platform-frontend/src/views/ai/McpManageView.vue
  - smart-recruit-commons/migrations/archive/pre-baseline-000089/000070_add_platform_ai_control_plane.sql
last_verified: 2026-07-28
review_after: 2026-10-14
---

# MCP Tool Governance

MCP governance covers platform-global server configuration, tool policy evaluation, policy APIs, tenant-scoped runtime logs, and platform admin surfaces. Enterprises consume MCP capability through published platform AI releases and do not maintain server or policy configuration. Keep deny/rate-limit/confirmation order, private-network blocking, stdio command allowlists, URL host constraints, argument redaction, and audit persistence aligned.

Native MCP governance persists server CRUD, tool policy CRUD/list, and tool log list responses through the AI Agent database tables. Server env vars, MCP args, runtime config, log result payloads, policy details, and error strings must stay redacted or truncated on read paths.

When a safe MCP runner is bound, live connection tests, discovery, and policy-governed calls are supported. HR chat invokes an Agent-bound MCP tool only when its exact capability key is explicitly selected in `skill_capability_keys`; an empty selection invokes none. The runtime still evaluates enablement, deny/rate-limit/confirmation policy, transport allowlists/private-network constraints, redaction, and audit logging before returning success. `agent_skill_selection_confirmed` never grants MCP authority. A confirmation-required durable Run pauses in `waiting_confirmation` and stores a random, ten-minute confirmation request bound to the exact capability key and canonical argument hash. Resume accepts only an explicit `mcp_tool` payload matching that request; changed capability/arguments, expired requests, generic Skill confirmations, and replay against a different call fail closed. Direct non-durable chat returns explicit confirmation-required failure instead of bypassing the policy. Without a bound runner or when policy blocks the call, the operation is explicit non-success.

The HR confirmation surface is two-way: approval resumes the exact Run, while
rejection calls the Run cancel endpoint and reaches the canceled terminal state
without granting tool authority. The composer remains disabled throughout
`waiting_confirmation`, preventing a second active Run for the same session.
The approval payload retains `expires_at`; when the client detects expiry it
cancels the parked Run and requires a fresh request instead of repeatedly
submitting an invalid approval.

## Verification

Verified against the platform ownership migration and routes plus cumulative MCP runtime integration, explicit/empty selection, independent bound confirmation, client approval/rejection/expiry handling, policy failure, audit, and HR Agent runtime tests on 2026-07-28.
