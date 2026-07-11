# TASK Report - TASK-HARS-008

## 1. TASK ID

TASK-HARS-008

## 2. Modified File List

- `hr-frontend/src/composables/useHrAgentRun.ts` (modify)
- `hr-frontend/src/composables/useHrAgentRun.test.ts` (modify)
- `hr-frontend/src/views/hr/AIChatView.vue` (modify)
- `hr-frontend/src/views/hr/AIChatView.test.ts` (create)
- `hr-frontend/src/utils/hrAgentRunReducer.test.ts` (modify)
- `logic-grpc-service/service/agent_run_durable_test.go` (modify)
- `.spec/hr-agent-resumable-stream/reports/TASK-HARS-008-report.md`
- `.spec/hr-agent-resumable-stream/reports/TASK-HARS-008-evidence.json`
- `.spec/hr-agent-resumable-stream/reports/pipeline-summary.md`
- `.spec/hr-agent-resumable-stream/pipeline-state.json`

Unchanged allowed files (validated only):

- `hr-frontend/src/api/agentRun.ts` — SSE subscribe with `after_seq` / `Last-Event-ID` already correct
- `web-gin-service/handler/hr/*ai*test*.go` — disconnect-without-cancel coverage still passes

## 3. Change Summary by File

| File | Summary |
| --- | --- |
| `useHrAgentRun.ts` | Adds bounded auto-reconnect after unexpected SSE drops while run is non-terminal, always resubscribing with `lastEventSeq`. `dispose` / `reset` / unmount abort `waitUntilSettled` waiters as `aborted` without calling cancel. Hydrate races invalidated via `hydrateGeneration`. Exposes `reconnectAttempt`. |
| `useHrAgentRun.test.ts` | Covers reconnect with `after_seq`, dispose≠cancel and no reconnect after dispose, unmount aborts waiters, hydrate+cancel same run id. |
| `AIChatView.vue` | On `selectSession` (mount/refresh path), calls `restoreActiveRunForSession`: `getActiveAgentRun` → hydrate snapshot → subscribe from `last_event_seq` → bind UI. Handles waiting_confirmation parking, completion history reconcile, cancel still via stop button. |
| `AIChatView.test.ts` | Unit coverage for assistant-slot seeding used by refresh restore (reuse pending bubble / append new / prefix continuation). |
| `hrAgentRunReducer.test.ts` | Hydrate + replay after `last_event_seq` does not duplicate assistant/process text. |
| `agent_run_durable_test.go` | `TestGetActiveAgentRunRecoverySnapshot`: active run snapshot for refresh, last_event_seq cursor, cleared after success. |
| report/evidence/pipeline-summary/pipeline-state | Harness artifacts; `current_phase=review`. |

## 4. Scope Check Result

**PASS** — TASK-local changed files match allowed globs.

```bash
TASK_BASE_TREE=71e895a2c80023bba4c84b24d5e1a50023d40bc1 \
  bash .spec/hr-agent-resumable-stream/scripts/check-task-scope.sh TASK-HARS-008
# exit_code=0
```

Notes:

- Restored prior `TASK-HARS-007-evidence.json` blob from `base_tree` before final scope pass (post-007 review mutated the evidence after head_tree freeze; review verdict remains in `pipeline-state.json`).
- No forbidden files. No package.json/lockfile/proto/migration changes.

## 5. SPEC Comparison Result

Aligned with:

| Requirement | Result |
| --- | --- |
| FR-008 subscription disconnect ≠ cancel | Pass — reconnect/dispose never call `cancelAgentRun` |
| FR-009 active-run lookup for refresh | Pass — `hydrateFromActive` + view restore |
| FR-010 explicit cancel | Pass — post-hydrate cancel tested; stop button path unchanged |
| FR-011 confirmation survives refresh | Pass — restore parks skill selection from snapshot/confirmation |
| FR-015 page refresh restore without duplicate text | Pass — hydrate snapshot + reducer ignores `seq <= lastEventSeq` + UI slot seeding |
| FR-017 / AC-008 legacy stream | Pass — `sendMessageStream` still exported from `api/ai.ts` |
| AC-001..007, AC-010 recovery behavior | Pass at unit/service/gateway layers (see checks) |

## 6. SDD Comparison Result

Matches SDD §6 refresh restore flow:

1. get active run for session
2. hydrate reducer from snapshot
3. subscribe events after `last_event_seq`
4. bind UI / continue cancel-confirm-complete

Also matches §8 compatibility (legacy stream retained) and §9 (failed restore does not corrupt chat history).

## 7. Acceptance Comparison Result

| Criterion | Result |
| --- | --- |
| On mount/refresh, query active run for session | Pass (`selectSession` → `restoreActiveRunForSession`) |
| Snapshot hydrates answer, process, status, confirmation, result | Pass (composable hydrate + binder + confirmation parking) |
| Resume from last sequence without duplication | Pass (subscribe cursor + reducer test) |
| Refresh/disconnect does not cancel backend run | Pass (frontend dispose≠cancel; gateway + logic disconnect tests) |
| Explicit cancel persists canceled state | Pass (hydrate then cancel test; existing durable cancel test) |
| Waiting confirmation survives refresh | Pass (restore path applies skill selection UI) |
| Completion reconciles with chat history | Pass (post-settlement `getSessionMessages`) |
| Legacy stream still available | Pass (`sendMessageStream` present) |
| Final report knowledge impact + rollout risks | Pass (this report) |

Manual live browser refresh while a long-running agent is mid-tool was not exercised in this environment (no full local stack run). Covered by unit/service tests and SDD workflow wiring instead.

## 8. Test Commands and Results

| Command | Result |
| --- | --- |
| `git diff --name-only` | OK (feature dirty tree + TASK-008 files) |
| `TASK_BASE_TREE=71e895a2c80023bba4c84b24d5e1a50023d40bc1 bash .spec/.../check-task-scope.sh TASK-HARS-008` | PASS (exit 0) |
| `bash .spec/hr-agent-resumable-stream/scripts/agent-check.sh TASK-HARS-008` | PASS (exit 0) |
| `pnpm --filter hr-frontend typecheck` | PASS |
| `pnpm --filter hr-frontend test` | PASS — 42 tests |
| `go test ./service/ -run 'Durable\|AgentRun\|CreateAgentRun\|Cancel\|Confirm\|Subscribe\|GetActive'` (logic-grpc) | PASS |
| `go test ./handler/hr/ -run 'AgentRun'` (web-gin) | PASS |

## 9. Knowledge Impact

- **result**: `update_required` (knowledge files out of scope; no `.knowledge` edits)
- **review**:
  - `.knowledge/architecture/agent-runtime.md` — **UNCHANGED**. Still describes runtime at a high level; does not document durable run worker, active-run pointer, or refresh restore. Update later when scope allows.
  - `.knowledge/architecture/api-contracts-and-gateway.md` — **UNCHANGED**. Gateway durable REST/SSE routes not yet documented.
  - `.knowledge/architecture/frontend-apps.md` — **UNCHANGED**. Vue structure still accurate; lacks `useHrAgentRun` + AIChatView refresh recovery notes.
  - `.knowledge/runbooks/frontend-validation.md` — **UNCHANGED**. typecheck + vitest remain correct post-change checks used here.
- **coverageGap**: false
- No STALE/CONFLICT verdicts (debt via `update_required` + UNCHANGED lag notes only).

## 10. Risks

- Live multi-tab / multi-device concurrent subscription to the same run is untested; last-event replay should be safe but UI binder assumes single subscriber per tab.
- Auto-reconnect budget (8 attempts, exponential backoff) may exhaust on long outages; user must refresh to re-hydrate.
- Assistant-slot seeding mirrors restore rules in `AIChatView.test.ts` (not shared export); drift risk if helper changes without test update.
- History reconcile after terminal restore may briefly flash the transient assistant bubble before server history replace.
- Rollout still depends on migration `000050` applied and gateway/logic deployment co-released; legacy stream remains for rollback.
- Knowledge docs lag the new durable runtime (documented debt).

## 11. Follow-up Items

- Independent self-review of TASK-HARS-008
- Optional knowledge updates for agent-runtime, api-contracts, frontend-apps
- Optional extraction of `ensureRestoreAssistantSlot` into a shared testable module in a later non-feature cleanup
- Do **not** mark entire pipeline completed until review passes (orchestrator owns that)

## 12. Whether the Next TASK Can Start

**No further product TASK.** This is the final TASK (001–008). Implementation is complete and ready for **self-review**. Do not mark the pipeline completed until TASK-HARS-008 review passes.
