# TASK Report - TASK-002

## 1. TASK ID

TASK-002 - Restore HR AI Chat Runtime

## 2. Modified File List

- `.spec/ai-agent-runtime-recovery/pipeline-state.json`
- `.spec/ai-agent-runtime-recovery/reports/TASK-002-report.md`
- `.spec/ai-agent-runtime-recovery/reports/TASK-002-evidence.json`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_ai_chat_test.go`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_store.go`

## 3. Change Summary by File

- `native_servers.go`: replaced HR Chat/ChatStream direct raw `complete()` flow with a native HR runtime path that creates/validates HR sessions, persists the user message, loads recent HR history, executes an approved `get_application_snapshot` owner-service tool when `application_id` is present, persists tool traces, estimates context usage, calls the provider with structured HR context, emits stream status/tool/context/fallback/done events, and uses deterministic HR fallback when provider generation fails after useful tool results.
- `native_ai_chat_test.go`: added focused HR Chat/ChatStream tests for tool-context prompt construction, trace persistence, context usage, provider-failure fallback, fallback stream events, and existing session ownership behavior.
- `native_store.go`: added `AppendToolTrace` and corrected `ai_tool_traces` GORM column mapping to the existing schema (`arguments_json`, `result_summary`, `error_message`) without schema changes.
- `.spec/ai-agent-runtime-recovery/pipeline-state.json`: recorded TASK-002 base tree and the user's prior human confirmation grant.
- TASK report/evidence files: record implementation, checks, knowledge impact, and review state.

## 4. Scope Check Result

Passed.

Command:

```bash
bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh TASK-002
```

Result:

```text
base_tree: e7902b30ceb5de303d03620cfb43296989f4cc70
scope_result: PASS (TASK-002)
```

## 5. SPEC Comparison Result

Passed.

- Restores HR Chat/ChatStream away from raw prompt-only provider calls.
- Adds service-owned runtime context using current microservice boundaries and existing `ApplicationOwnerService.GetApplicationSnapshot`.
- Persists tool traces and assistant process content where existing schema supports it.
- Emits compatible stream events and context usage using existing protobuf fields.
- Uses deterministic fallback from `smart-recruit-commons/ai.BuildHRFallbackReply` after useful tool results.

## 6. SDD Comparison Result

Passed with bounded parity.

- Implements the first HR runtime slice described by SDD 3.2 and 6.1 without reintroducing old monolith dependencies.
- Keeps gateway/proto/schema/auth untouched.
- Full Agent Skill semantic selection, MCP tools, and broader HR tool catalog remain intentionally scoped to later TASKs.

## 7. Acceptance Comparison Result

Passed.

- HR Chat now builds recruitment-aware context and uses an approved owner-service tool when application context is available.
- ChatStream emits `thinking`, `tool_calling`, `tool_done`, `context_usage`, `generating`, `fallback`, and `done` events in covered paths.
- Tool traces are persisted via `AppendToolTrace` and remain retrievable through the existing trace list path.
- Context usage is estimated and returned/emitted.
- Provider failure after tool results produces deterministic fallback.
- Dev references used: `origin/dev:logic-grpc-service/service/ai_service.go`, `origin/dev:logic-grpc-service/service/agent_context.go`, `origin/dev:logic-grpc-service/ai/hr_adk_tools.go`, `origin/dev:logic-grpc-service/ai/tool_executor.go`, `origin/dev:logic-grpc-service/repository/chat_repo.go`, and `origin/dev:logic-grpc-service/repository/tool_trace_repo.go`.

## 8. Test Commands and Results

| Command | Result |
|---|---|
| `GOWORK=off go test ./internal/interfaces/grpc -run 'TestHRChat|TestHRChatStream'` from `smart-recruit-ai-agent-service` | Passed; includes failed-tool/no-fallback and fallback-event coverage |
| `GOWORK=off go test ./...` from `smart-recruit-ai-agent-service` | Passed |
| `git diff --name-only` | Passed; tracked cumulative diff includes TASK-001 `go.sum` plus TASK-002 production/test files. Untracked `.spec` report/evidence files are covered by TASK scope check. |
| `bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh TASK-002` | Passed |
| `bash .spec/ai-agent-runtime-recovery/scripts/agent-check.sh` | Passed |
| `node .knowledge/scripts/detect-impact.mjs --root . --base-tree e7902b30ceb5de303d03620cfb43296989f4cc70 --json` | Passed |

## 9. Knowledge Impact

Result: `update_required`.

Mechanical routes required review of AI runtime, service boundary, gateway/proto, and persistence knowledge because TASK-002 changed AI Agent runtime adapter and persistence mapping. No `.knowledge/**` files were edited because they are outside TASK scope. No conflict or coverage gap was found.

Reviewed documents: `agent-runtime`, `agent-skill`, `ai-configuration-governance`, `api-contracts-and-gateway`, `debug-agent-retrieval`, `embedding-fallback`, `local-development`, `mcp-policy-audit`, `mcp-tool-governance`, `memory-and-context`, `migration-model-drift`, `persistence-and-migrations`, `protobuf-and-migration-change`, `protobuf-synchronization`, `resume-intelligence`, `resume-sensitive-data`, `semantic-retrieval`, `service-boundaries`, and `system-overview`.

## 10. Risks

- This TASK restores a bounded HR runtime slice using current owner-service contracts. Full dev parity for every HR tool still depends on later TASKs and may require additional owner-service read ports if missing.
- Agent Skill semantic selection, MCP live tools, and embedding-aware selection are intentionally deferred to TASK-006, TASK-007, and TASK-008.
- No live provider smoke was run; fake-provider tests cover runtime behavior without credentials.

## 11. Follow-up Items

- TASK-003 should reuse the native HR runtime path for durable Agent Runs.
- TASK-006 should connect AgentConfig/Prompt/Agent Skill selection more completely.
- TASK-007/TASK-008 should add MCP and semantic tool integrations.

## 12. Whether the Next TASK Can Start

Yes. Independent self-review round 2 returned `verdict: 通过`, and canonical evidence validation passed.

## Repair Summary

### Failed Check

First independent self-review returned `verdict: 不通过`.

### Root Cause

The initial implementation treated any trace, including failed/no-data traces, as sufficient for deterministic fallback and emitted a successful `tool_done` event even when the tool failed.

### Files Changed

- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_ai_chat_test.go`
- `.spec/ai-agent-runtime-recovery/reports/TASK-002-report.md`
- `.spec/ai-agent-runtime-recovery/reports/TASK-002-evidence.json`

### Fix Summary

- Fallback now requires at least one useful tool result: non-empty `ResultContent` and empty `ErrorMsg`.
- Failed HR tool calls now emit `event_type=error` with `error_type=TOOL_ERROR` instead of a misleading successful `tool_done`.
- Added tests for missing application owner client and failed snapshot responses to ensure provider failure is not converted into fallback without useful tool data.

### Re-run Commands

```bash
GOWORK=off go test ./internal/interfaces/grpc -run 'TestHRChat|TestHRChatStream'
GOWORK=off go test ./...
bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh TASK-002
bash .spec/ai-agent-runtime-recovery/scripts/agent-check.sh
```

### Re-run Results

- Targeted HR Chat/ChatStream tests: passed.
- AI Agent service suite: passed.
- Scope check: passed.
- Agent check: passed.

### Remaining Risks

No remaining TASK-002 blocker known. Broader HR tool catalog and semantic/MCP integrations remain later TASKs.
