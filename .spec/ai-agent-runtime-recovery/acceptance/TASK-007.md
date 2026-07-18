# Acceptance - TASK-007

## TASK Summary

Restore MCP live runtime behavior for connection tests, tool discovery, tool execution, policy enforcement, and audit logging.

## SPEC References

- FR-007 MCP runtime restoration
- Observability and Debug Requirements 5
- Error Handling and Fallback Requirements 5
- Security and Safety Requirements 4, 5
- Acceptance Criteria 11

## SDD References

- Proposed Design 3.7
- Algorithm or Workflow Changes 6.5
- Error Handling and Fallback Design
- Observability and Debug Output Design
- Migration Risks 4, 7

## Acceptance Criteria

- MCP connection test performs live validation and returns real status/details.
- Tool discovery returns real tool schemas.
- Tool execution enforces policy, confirmation, redaction, timeout, and audit logging.
- Unsafe or unsupported runtime conditions remain explicit non-success responses.
- Approved MCP tools can be used by HR runtime when configured.

## Required Checks

- Targeted AI Agent service tests for connection, discovery, execution, policy denial, redaction, and logs.
- Gateway handler tests if HTTP mapping changes.
- `GOWORK=off go test ./...` from `smart-recruit-ai-agent-service`.
- `GOWORK=off go test ./...` from `smart-recruit-gateway` if gateway is touched.
- `pnpm --filter hr-frontend typecheck` if HR frontend is touched.
- `git diff --name-only`.
- `bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh TASK-007`.
- `bash .spec/ai-agent-runtime-recovery/scripts/agent-check.sh`.

## Manual Verification, if needed

Live MCP smoke requires a configured safe test server. If unavailable, document skipped live smoke and provide fake-runner policy/audit test evidence.

## Out-of-Scope

- New MCP transports not required by dev-compatible behavior.
- Dependency additions without confirmation.
- Schema/proto changes.
