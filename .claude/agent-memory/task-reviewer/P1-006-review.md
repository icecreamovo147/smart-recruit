---
name: P1-006-review
description: Review of P1-006 MCP tool center backend: Round 2 PASS after all fixes verified
metadata:
  type: reference
---

## Review: P1-006 MCP Tool Center Backend (Round 2, 2026-06-27)

**Verdict**: PASS -- Ready for merge.

**Branch**: `agent/P1-006-mcp-tool-center-backend`

### Round 1 Issues (all verified fixed)

1. **desensitizeJSON was no-op** -- FIXED: Now recursive JSON walk (`desensitizeValue`) with 30+ sensitive field name patterns + regex PII fallback (`desensitizePlainText`) for non-JSON strings.
2. **Zero tests** -- FIXED: 39 test functions across `mcp_service_test.go` (22), `mcp_repo_test.go` (7), `mcp_test.go` web-gin handler (7), plus `repo_test_helper.go`.
3. **Task tracking not updated** -- FIXED: EXECUTION_LOG shows "fixing" with branch; TRACEABILITY_MATRIX shows "审查中"; task file has fix content and verification results.
4. **MCP injection only in Legacy** -- FIXED: Both `runADKChat` and `runLegacyChat` merge MCP tools (lines 456-463 and 504-509 of `ai_service.go`). ADK uses `CollectEnabledMCPCallableTools` (tool.BaseTool wrappers), Legacy uses `CollectEnabledMCPToolInfos` (schema.ToolInfo).
5. **env_vars not desensitized** -- FIXED: `serverToInfo` calls `desensitizeEnvVars` which masks all values to "***". Tested in `TestServerToInfo_DesensitizesEnvVars`.

### Key Architecture Decisions

- MCP SDK: `github.com/mark3labs/mcp-go v0.55.1`
- ADR at `docs/agent-harness/decisions/20260627-mcp-sdk-selection.md`
- MCP servers default to disabled (`is_enabled=false`)
- Connection test timeout: 10s
- All routes use `PermSystemConfigManage` permission
- Tool call logs stored in `mcp_tool_logs` with desensitized args/results

### Pre-existing `go vet` Warnings

4 warnings in `web-gin-service/handler/hr/*cursor_test.go` and `handler/candidate/apply_cursor_test.go` (proto MessageState lock copy) -- NOT related to this task.
