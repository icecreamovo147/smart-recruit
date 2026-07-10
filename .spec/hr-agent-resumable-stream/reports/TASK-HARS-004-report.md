# TASK Report - TASK-HARS-004

## 1. TASK ID

TASK-HARS-004

## 2. Modified File List

- `logic-grpc-service/service/ai_service.go`
- `logic-grpc-service/service/services.go`
- `logic-grpc-service/service/agent_run_events.go` (new)
- `logic-grpc-service/service/agent_run_hub.go` (new)
- `logic-grpc-service/service/agent_run_service.go` (new)
- `logic-grpc-service/service/agent_run_worker.go` (new)
- `logic-grpc-service/service/agent_run_durable_test.go` (new)
- `logic-grpc-service/repository/agent_run_repo.go`
- `logic-grpc-service/server/server.go` (**scope exception**)
- `.spec/hr-agent-resumable-stream/reports/TASK-HARS-004-report.md`
- `.spec/hr-agent-resumable-stream/reports/TASK-HARS-004-evidence.json`
- `.spec/hr-agent-resumable-stream/pipeline-state.json`

## 3. Change Summary by File

| File | Summary |
| --- | --- |
| `ai_service.go` | Adds `agentRunEvents`, in-process `eventHub`, injectable `durableRunExecutor`; `WithAgentRunEventRepo`; durable hooks in `runToolCallingChatWithUsage` (reuse existing recorder, park skill confirmation, skip stream-owned cancel/final persist when durable). |
| `services.go` | Wires `repository.NewAgentRunEventRepo(db)` into AIService. |
| `agent_run_events.go` | Durable event type constants + plan request key. |
| `agent_run_hub.go` | In-process fan-out hub for live subscribers (DB poll remains source of multi-instance truth). |
| `agent_run_service.go` | Create/Get/Active/Cancel/Confirm/Subscribe RPCs; ownership; idempotent create; snapshot/event mapping; event append + status transitions. |
| `agent_run_worker.go` | Detached `context.Background()` worker; cancel poller; delta batching; waiting_confirmation park; terminal complete/cancel/fail; idempotent final assistant history via `agent_run_id`. |
| `agent_run_durable_test.go` | Lifecycle tests with fake executor: idempotency, cancel, confirm resume, event ordering, final message once, subscribe disconnect. |
| `agent_run_repo.go` | `UpdateRunHistoryID`, `CountAssistantHistoryByAgentRunID` for idempotent final assistant association. |
| `server/server.go` | Thin Server forwards for the six durable RPCs (required for gRPC exposure). |
| report/evidence/pipeline-state | Harness artifacts; `current_phase=review`. |

## 4. Scope Check Result

**FAILED** solely for intentional scope exception:

- `logic-grpc-service/server/server.go` is required to forward CreateAgentRun / GetAgentRun / GetActiveAgentRun / SubscribeAgentRunEvents / CancelAgentRun / ConfirmAgentRun.
- `task-scope.json` for TASK-HARS-004 did not list `server.go` (gap called out in implement instructions).
- Implementation kept server methods as thin one-line forwards matching ChatStream/GetAgentRuns style.
- **Did not** modify `task-scope.json`.

All other TASK-local files matched allowed globs.

```bash
TASK_BASE_TREE=e52fdde754caca080ecb6b13df938d6bcb042745 \
  bash .spec/hr-agent-resumable-stream/scripts/check-task-scope.sh TASK-HARS-004
# exit_code=1 (server.go OUT_OF_SCOPE)
```

## 5. SPEC Comparison Result

Aligned with:

- FR-001/002: CreateAgentRun with session/message/action + client_request_id; idempotent replay
- FR-003: Worker uses detached background context (not HTTP/SSE/subscribe ctx)
- FR-004: State machine via existing transitions + durable service transitions
- FR-005/006: Ordered events + snapshot fields updated during run
- FR-008: Subscribe ctx cancel ends stream only
- FR-009/010/011: GetActiveAgentRun, CancelAgentRun, ConfirmAgentRun same run_id
- FR-016: Single assistant history write associated with agent_run_id
- FR-017 / AC-008: ChatStream path unchanged and still available
- AC-001..006, AC-008: covered at service layer (gateway/UI later)

## 6. SDD Comparison Result

Matches SDD §3 / §6 / §9:

- Command/execution/subscription split
- Event types from SDD §6
- Cancel as command; disconnect is not cancel
- waiting_confirmation pause + Confirm resume
- MySQL events as durable source; in-process hub acceleration only
- Legacy ChatStream compatibility retained

## 7. Acceptance Comparison Result

| Criterion | Result |
| --- | --- |
| Create idempotent for HR+session+client_request_id | PASS (test) |
| Execution not owned by HTTP/SSE/subscription ctx | PASS (worker Background) |
| Worker appends ordered events + snapshots | PASS (worker + test) |
| Explicit cancel → cancel_requested → canceled | PASS (test) |
| Subscription disconnect does not cancel run | PASS (test) |
| waiting_confirmation + confirm same run | PASS (test) |
| Final assistant message once | PASS (test) |
| Legacy ChatStream remains | PASS (untouched path) |

## 8. Test Commands and Results

| Command | Result |
| --- | --- |
| `git diff --name-only` | OK (exit 0) |
| `TASK_BASE_TREE=e52fdde... bash .../check-task-scope.sh TASK-HARS-004` | FAIL (exit 1) — server.go exception |
| `bash .../agent-check.sh TASK-HARS-004` | PASS (exit 0) |
| `go test ./service/ -run 'AgentRun\|Durable\|CreateAgentRun\|...'` | PASS |
| `go test ./repository/ -run 'AgentRun'` | PASS |
| `go build ./...` (logic-grpc-service) | PASS |

## 9. Knowledge Impact

| Document | Verdict | Notes |
| --- | --- | --- |
| `.knowledge/architecture/agent-runtime.md` | UNCHANGED (update_required debt) | Still stream-centric; durable worker/event ownership not described. Out of allowedFiles; report debt. |
| `.knowledge/architecture/service-boundaries.md` | UNCHANGED | Logic owns lifecycle; gateway remains transport — still accurate. |
| `.knowledge/domains/notification-outbox.md` | UNCHANGED | Durable runs use in-process dispatch, not outbox; no conflict. |

Overall knowledge result: `update_required` (agent-runtime narrative lag). No `.knowledge` edits (out of scope).

## 10. Risks

- Production default executor still depends on full AI pipeline; unit tests use injectable fake executor.
- In-process hub is single-instance; multi-instance relies on DB poll (eventual consistency).
- Worker restart after process crash not auto-recovered in this TASK (partial/failed on crash is acceptable per SPEC).
- Scope exception for `server.go` should be accepted in review or folded into task-scope later.
- Concurrent SQLite unit tests required `SetMaxOpenConns(1)`; MySQL production locking is different.

## 11. Follow-up Items

- Self-review TASK-HARS-004
- TASK-HARS-005: gateway REST/SSE
- Knowledge update for agent-runtime durable runs (later allowed TASK)

## 12. Whether the Next TASK Can Start

**No** — not until self-review passes and user/pipeline confirms. Implementation is ready for review (`current_phase=review`). Do not start TASK-HARS-005 yet.


## Repair Summary

### Failed Check
Self-review verdict 不通过 (round 1): F1 final assistant history suppressed by user message_id pollution of history_id; F2 missing production-path regression test; F4 option_context_json not patched.

### Root Cause
`UpdateRunMessageID` wrote both `message_id` and `history_id` to the user history id. `persistFinalAssistantOnce` short-circuited on any non-zero `history_id`, so production durable completion (with `skipFinalAssistantPersist`) could finish without an assistant history row.

### Files Changed
- `logic-grpc-service/repository/agent_run_repo.go` — `UpdateRunMessageID` only sets message_id
- `logic-grpc-service/repository/agent_run_repo_test.go` — assert separation of message_id vs history_id
- `logic-grpc-service/service/agent_run_worker.go` — final persist uses assistant count; tolerate legacy pollution; patch option_context_json
- `logic-grpc-service/service/agent_run_durable_test.go` — `TestFinalAssistantAfterUserMessageID`

### Fix Summary
Separated user `message_id` from assistant `history_id`. Final assistant write keys off assistant history count and only treats distinct history_id as already-written. Option context snapshot written from result metadata. Regression test covers user-link-then-final path including legacy pollution.

### Re-run Commands
```
go test ./service/ -count=1 -timeout 120s -run 'FinalAssistant|Durable|CreateAgentRun|CancelAgent|ConfirmAgent|SubscribeAgent'
go test ./repository/ -count=1 -timeout 60s -run 'AgentRun'
go build ./...
```

### Re-run Results
All PASS (2026-07-10 repair).

### Remaining Risks
- server.go still OUT_OF_SCOPE (documented exception, thin forwards only)
- Real AI pipeline still covered via hooks + fake executor unit tests

