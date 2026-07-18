# TASK Report - TASK-007

## 1. TASK ID

TASK-007 - Restore MCP Runtime Execution

## 2. Status

Completed.

Independent self-review round 1 returned `不通过`; fixes were applied for policy-specific HR trace redaction, invalid-args audit logging, and missing evidence generation. Round 2 returned `通过`.

Human confirmation was required and was already granted by the user at pipeline start for all `requiresHumanConfirmation=true` tasks, including TASK-007.

## 3. Modified File List

- `.spec/ai-agent-runtime-recovery/pipeline-state.json`
- `.spec/ai-agent-runtime-recovery/reports/TASK-007-report.md`
- `.spec/ai-agent-runtime-recovery/reports/TASK-007-evidence.json`
- `smart-recruit-ai-agent-service/internal/infrastructure/mcp/runner.go`
- `smart-recruit-ai-agent-service/internal/infrastructure/mcp/runner_test.go`
- `smart-recruit-ai-agent-service/internal/infrastructure/mcp/native_runtime_integration_test.go`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/mcp_skill_store.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/mcp_skill_services.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go`

`git diff --name-only` does not list untracked files before staging; the new MCP test/runner files are included from `git status --short` and verified by the TASK scope checker.

## 4. Change Summary by File

- `internal/infrastructure/mcp/runner.go`: adds a native MCP runner with HTTP JSON-RPC and stdio JSON-RPC support, explicit non-success SSE handling, transport validation, timeout handling, tool discovery, tool calls, and runtime result types.
- `internal/infrastructure/mcp/runner_test.go`: verifies the default HTTP runner performs real JSON-RPC discovery and tool execution against an `httptest` server.
- `internal/infrastructure/mcp/native_runtime_integration_test.go`: verifies the runtime factory wires MCP service and HR AI runtime through a fake runner/store, covering connection, discovery, policy confirmation denial, redacted audit logs, and HR runtime MCP tool invocation.
- `internal/infrastructure/persistence/mcp_skill_store.go`: adds store-side runtime ports for raw MCP server config lookup, runtime status updates, enabled policy lookup, recent-call counts, and audit log writes. `NativeStore` still does not execute network/command runtime work.
- `internal/interfaces/grpc/mcp_skill_services.go`: changes `TestMCPConnection`, `ListMCPTools`, and `CallMCPTool` to coordinate store + runner. Calls now enforce policy order, confirmation, rate limit inputs, redaction, and mandatory audit logging before returning success.
- `internal/interfaces/grpc/native_servers.go`: injects a shared MCP runner through `RuntimeDeps`, wires the MCP service with that runner, and lets HR runtime invoke selected `capability_source=mcp` bindings with key format `<serverID>:<toolName>`.
- `pipeline-state.json`: records TASK-007 base tree and preauthorized human confirmation.

## 5. Scope Check Result

Passed.

Command:

```bash
TASK_BASE_TREE=947270a22584868dfe68bbee773b65117f8cc53b bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh TASK-007
```

Result:

```text
scope_result: PASS (TASK-007)
```

## 6. SPEC Comparison Result

Passed.

The implementation restores live MCP connection validation, tool discovery, tool execution, policy/confirmation/redaction/audit behavior, and HR runtime use of approved MCP capability bindings without proto, schema, auth/RBAC, dependency, global config, K8s, or root docs changes.

## 7. SDD Comparison Result

Passed.

Live MCP behavior is moved out of `NativeStore` into an injected runner under `internal/infrastructure/mcp`. `nativeMCPService` coordinates store + runner, while persistence remains responsible for server, policy, status, and log data.

## 8. Acceptance Comparison Result

Passed.

- MCP connection test attempts runner-backed live validation and updates server runtime status.
- MCP tool discovery returns runner-provided schemas.
- MCP tool execution enforces persisted policy, confirmation, redaction, timeout, and audit logging.
- Unsafe or unsupported runtime conditions remain explicit non-success responses, including unbound runner, disabled server, policy denial, confirmation required, and unsupported SSE.
- HR runtime can invoke approved MCP tools when the active AgentConfig has selected `mcp` capability bindings.

## 9. Test Commands and Results

| Command | Result |
|---|---|
| `GOWORK=off go test ./internal/infrastructure/mcp -run 'TestDefaultRunnerHTTPDiscoveryAndCall|TestNativeMCPRuntimeConnectionDiscoveryPolicyAuditAndHRTool' -count=1 -v` from `smart-recruit-ai-agent-service` | Passed |
| `GOWORK=off go test ./...` from `smart-recruit-ai-agent-service` | Passed |
| `git diff --name-only` | Passed; command does not list untracked new MCP files before staging |
| `git status --short` | Passed; used to record untracked new MCP runner/test files |
| `TASK_BASE_TREE=947270a22584868dfe68bbee773b65117f8cc53b bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh TASK-007` | Passed |
| `TASK_BASE_TREE=947270a22584868dfe68bbee773b65117f8cc53b bash .spec/ai-agent-runtime-recovery/scripts/agent-check.sh` | Passed |
| `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 947270a22584868dfe68bbee773b65117f8cc53b` | Passed; result `update_required` |

Gateway and HR frontend files were not touched, so gateway-specific mapping changes and `pnpm --filter hr-frontend typecheck` were not required by TASK-007 acceptance. `agent-check.sh` still ran gateway tests due earlier staged gateway changes and they passed.

## 10. Knowledge Impact

Result: `update_required`.

Routed active knowledge was reviewed: `agent-runtime`, `mcp-tool-governance`, `mcp-policy-audit`, `ai-configuration-governance`, and `api-contracts-and-gateway`. Additional documents were reported by impact detection because broad service routes matched the changed files.

No `.knowledge/**` files were edited because they are outside TASK-007 scope. Knowledge debt is recorded for MCP live runtime support replacing the previous “unsupported until runner is bound” statement.

## 11. Human Confirmation

Required and confirmed.

Confirmation text recorded in pipeline state:

```text
User explicitly authorized execution of all requiresHumanConfirmation=true TASKs within feature ai-agent-runtime-recovery, including but not limited to TASK-002 and TASK-007, and instructed the pipeline to record this confirmation in pipeline-state.json and corresponding TASK evidence/report without stopping again before those TASKs.
```

## 12. Dev Reference

Read-only dev reference subagent inspected:

- `origin/dev:logic-grpc-service/service/mcp_service.go`
- `origin/dev:logic-grpc-service/service/mcp_policy.go`
- `origin/dev:logic-grpc-service/service/mcp_policy_api.go`
- `origin/dev:logic-grpc-service/service/mcp_log_api.go`
- `origin/dev:logic-grpc-service/repository/mcp_repo.go`
- `origin/dev:logic-grpc-service/service/ai_service.go`
- `origin/dev:web-gin-service/handler/hr/mcp.go`

The restored behavior follows dev semantics for store/runner separation, policy order, audit logging, redaction, and HR Agent MCP capability binding while preserving the current microservice boundary.

## 13. Risks

- The native runner supports HTTP JSON-RPC and stdio; SSE remains an explicit unsupported non-success response in this TASK.
- Current persisted server config has no user-facing allow-private-network field, so private/local HTTP MCP endpoints remain blocked by policy validation.
- HR runtime invokes selected MCP capability bindings deterministically with a `query` argument; richer model-driven multi-tool planning remains bounded by the existing runtime architecture.

## 14. Follow-up Items

- Update `.knowledge/**` MCP runtime documents in a future knowledge-maintenance scope.
- If product requirements need SSE or private-network allowlists, add a separately scoped TASK with explicit schema/config confirmation.

## 15. Repair Summary

### Failed Review

Independent reviewer Zeno reported:

- HR runtime MCP tool traces used generic redaction and ignored policy-specific `redact_fields`.
- Malformed `args_json` returned before writing an MCP audit log.
- The report listed `TASK-007-evidence.json` before the file existed.

### Root Cause

The initial service correctly used policy redaction for MCP audit logs, but HR runtime separately persisted `ToolTraceRow.ArgsJSON` with generic redaction. Invalid argument parsing was handled before the policy/audit phase. Evidence generation had not yet been completed before the first review.

### Files Changed

- `smart-recruit-ai-agent-service/internal/interfaces/grpc/mcp_skill_services.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go`
- `smart-recruit-ai-agent-service/internal/infrastructure/mcp/native_runtime_integration_test.go`
- `.spec/ai-agent-runtime-recovery/reports/TASK-007-report.md`
- `.spec/ai-agent-runtime-recovery/reports/TASK-007-evidence.json`

### Fix Summary

- Added `nativeMCPService.RedactedArgsForTool` so HR runtime traces use the same policy-configured redaction fields as MCP audit logs.
- Changed malformed `args_json` handling to write a denied audit log with `policy_reason=invalid_args_json` when server context is available.
- Added test assertions for policy-specific redaction of HR tool trace args and invalid-args audit logging.
- Created the machine-readable evidence file.

### Re-run Commands

- `GOWORK=off go test ./internal/infrastructure/mcp -run 'TestNativeMCPRuntimeConnectionDiscoveryPolicyAuditAndHRTool|TestDefaultRunnerHTTPDiscoveryAndCall' -count=1 -v`
- `GOWORK=off go test ./...`
- `TASK_BASE_TREE=947270a22584868dfe68bbee773b65117f8cc53b bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh TASK-007`
- `TASK_BASE_TREE=947270a22584868dfe68bbee773b65117f8cc53b bash .spec/ai-agent-runtime-recovery/scripts/agent-check.sh`
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 947270a22584868dfe68bbee773b65117f8cc53b`

### Re-run Results

All re-run commands passed.

### Remaining Risks

Same as above: SSE and private-network allowlists remain explicit non-success/out-of-scope behavior.

## 16. Review Rounds

- Round 1: `不通过`; HR trace redaction, invalid-args audit logging, and missing evidence issues.
- Round 2: `通过`; fixes verified and no remaining acceptance/security/reporting issue found.

## 17. Whether the Next TASK Can Start

Yes. TASK-007 checks passed and independent self-review returned `通过`; TASK-008 can start.
