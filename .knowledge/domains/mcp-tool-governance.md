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
  - logic-grpc-service/service/mcp_service.go
  - logic-grpc-service/service/mcp_policy.go
  - logic-grpc-service/service/mcp_policy_api.go
  - logic-grpc-service/service/mcp_log_api.go
  - logic-grpc-service/repository/mcp_repo.go
  - web-gin-service/handler/hr/mcp.go
  - hr-frontend/src/views/hr/admin/McpManageView.vue
source_refs:
  - logic-grpc-service/service/mcp_service.go
  - logic-grpc-service/service/mcp_policy.go
  - logic-grpc-service/service/mcp_policy_api.go
  - logic-grpc-service/service/mcp_log_api.go
  - logic-grpc-service/repository/mcp_repo.go
  - logic-grpc-service/model/model.go
  - web-gin-service/handler/hr/mcp.go
  - web-gin-service/router/router.go
  - hr-frontend/src/views/hr/admin/McpManageView.vue
last_verified: 2026-07-10
review_after: 2026-10-08
---

# MCP Tool Governance

MCP support lets admins register external tool servers, inspect exposed tools, test connectivity, define per-tool policies, call tools, and review call logs. The governance boundary has three layers: server registration, policy evaluation, and persisted audit logs.

## Server Registration

`MCPService` accepts `stdio`, `sse`, and `http` transports, validates command or URL configuration, clamps timeout values against MCP config, and creates servers disabled by default. Listing tools and testing connections use longer MCP timeouts than ordinary admin CRUD because external processes or network services can be slow.

Server config can include command arguments and environment variables. Treat those as sensitive configuration: describe shape and behavior, not values.

## Policy Evaluation

`mcp_policy.go` evaluates enabled policies for a server/tool pair. The current decision space is allow, deny, confirmation required, or rate limited. Evaluation can consider:

- policy effect;
- caller role and caller scope;
- required and denied arguments;
- JSON argument rules with enum, regex, min, max, required, and deny semantics;
- per-tool rate limits based on recent tool logs;
- confirmation approval;
- fields to redact from logged output.

When no enabled policy exists, current behavior allows the tool call and records the reason as no policy. This is current-code behavior, not a recommendation.

## Audit Logs

MCP tool calls record server, tool, args, result content, duration, error, caller HR ID, session ID, policy ID, policy decision, policy reason, and policy snapshot. UI and handlers expose logs per server for operational review.

## Capability Binding

Enabled MCP tools become agent capabilities through `AgentConfigService.ListCapabilities`, which combines builtin, MCP, and SKILL capabilities. Agent config changes can therefore change which MCP tools the runtime can use even if the MCP server itself is unchanged.

## Review Triggers

- New MCP transport types, connection validation rules, timeout defaults, or server enablement defaults.
- Policy decision names, argument validation semantics, redaction, rate limiting, confirmation, or log snapshot behavior.
- Admin route permissions or frontend policy editing behavior.
- Agent capability binding that changes whether MCP tools can be selected by runtime agents.

## Verification

Verified against current MCP service, policy evaluator, policy API, log API, repository, gateway handler, router, and HR MCP management view on 2026-07-10.
