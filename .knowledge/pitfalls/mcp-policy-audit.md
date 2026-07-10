---
schema_version: 1
id: mcp-policy-audit
title: MCP policy and audit drift
kind: pitfall
status: active
owners:
  - agent-platform
tags:
  - mcp
  - audit
  - security
  - policy
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
  - hr-frontend/src/api/mcp.ts
  - hr-frontend/src/views/hr/admin/McpManageView.vue
last_verified: 2026-07-10
review_after: 2026-10-08
---

# MCP Policy and Audit Drift

MCP integrations are powerful because they bridge the agent platform to external tools. Drift between tool registration, policy evaluation, capability binding, and audit logs can create confusing runtime behavior or incomplete review trails.

## Common Failure Modes

- A server is enabled and exposes tools, but no enabled policy exists for a risky tool. Current behavior allows the call with a no-policy reason.
- A policy exists but frontend serialization changes array/object fields, causing role, scope, argument, redaction, or rate-limit rules to evaluate differently than the admin intended.
- A route or menu permission allows a user to change MCP servers or policies without matching governance expectations.
- Agent config binds a capability key that no longer matches the MCP server/tool capability key.
- Tool logs omit the policy decision, reason, snapshot, caller, session, args, result, or error needed to reconstruct a call.
- Redaction fields are configured but downstream display or logging changes still expose sensitive args or result fragments.

## Checks Before MCP Changes Ship

- Confirm server create/update validation still rejects unsupported transports and unsafe config according to current config rules.
- Confirm new or changed policy fields are persisted, returned by API, transformed by frontend API helpers, and displayed/editable by `McpManageView`.
- Confirm `evaluateToolPolicy` and `CallMCPTool` agree on decision names and log fields.
- Confirm tool logs still include enough context for audit review without exposing unnecessary sensitive content.
- Confirm agent capability listing still maps MCP tools into stable source/key/name fields.

## Report Guidance

When a TASK touches MCP behavior, include the server ID/name, tool name, policy ID, decision, reason, caller role/scope if available, and whether a log row was created. Do not include real command env vars, credentials, private endpoint tokens, or raw sensitive tool results.
