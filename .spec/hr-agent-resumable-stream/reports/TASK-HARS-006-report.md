# TASK Report - TASK-HARS-006

## 1. TASK ID

TASK-HARS-006

## 2. Modified File List

- `hr-frontend/src/types/agentRun.ts` (create)
- `hr-frontend/src/api/agentRun.ts` (create)
- `hr-frontend/src/utils/hrAgentRunReducer.ts` (create)
- `hr-frontend/src/utils/hrAgentRunReducer.test.ts` (create)
- `hr-frontend/src/composables/useHrAgentRun.ts` (create)
- `hr-frontend/src/composables/useHrAgentRun.test.ts` (create)
- `.spec/hr-agent-resumable-stream/reports/TASK-HARS-006-report.md`
- `.spec/hr-agent-resumable-stream/reports/TASK-HARS-006-evidence.json`
- `.spec/hr-agent-resumable-stream/pipeline-state.json`

Unchanged allowed files (not required for this TASK):

- `hr-frontend/src/api/ai.ts` — legacy chat stream helpers preserved; durable APIs live in `agentRun.ts`
- `hr-frontend/src/types/ai.ts` — legacy chat types preserved; durable types live in `agentRun.ts`

## 3. Change Summary by File

| File | Summary |
| --- | --- |
| `hr-frontend/src/types/agentRun.ts` | Adds snake_case gateway-aligned types for statuses, event types, snapshot, events, create/get/active/cancel/confirm request/response, confirmation, result metadata, and terminal/active helpers. |
| `hr-frontend/src/api/agentRun.ts` | Typed REST helpers via existing `request` wrapper; SSE subscribe via `fetch` + ReadableStream (cookie auth, silent refresh, AbortSignal cancels subscription only). Supports `after_seq` / `Last-Event-ID`. |
| `hr-frontend/src/utils/hrAgentRunReducer.ts` | Pure TS reducer: `createInitialHrAgentRunState`, `hydrateFromSnapshot`, `reduceAgentRunEvent`, `applyAgentRunEvents`. Ignores wrong `run_id` and `seq <= lastEventSeq`; deltas append, snapshots replace; handles status/confirmation/result/error/cancel/completion/heartbeat/tool events. |
| `hr-frontend/src/utils/hrAgentRunReducer.test.ts` | Vitest coverage for deltas, snapshots, duplicates, stale run, confirmation, terminal states, heartbeat, tools. |
| `hr-frontend/src/composables/useHrAgentRun.ts` | Vue composable owning startRun, subscribe, hydrateFromActive/RunId, cancel, confirm, applyEvent, dispose/reset. `onUnmounted` / `dispose` abort subscription only — never calls `cancelAgentRun`. |
| `hr-frontend/src/composables/useHrAgentRun.test.ts` | Mocks API; covers create+subscribe, abort≠cancel, explicit cancel, hydrate, confirm, applyEvent. |
| report/evidence/pipeline-state | Harness artifacts; `current_phase=review`. |

## 4. Scope Check Result

**PASS** — TASK-local changed files match allowed globs.

```bash
TASK_BASE_TREE=157c9702fa5d78169d7265ee5776c45513d5ac74 \
  bash .spec/hr-agent-resumable-stream/scripts/check-task-scope.sh TASK-HARS-006
# exit_code=0
```

Allowed hits:

- `hr-frontend/src/api/agentRun.ts`
- `hr-frontend/src/types/agentRun.ts`
- `hr-frontend/src/utils/hrAgentRunReducer.ts`
- `hr-frontend/src/utils/hrAgentRunReducer.test.ts`
- `hr-frontend/src/composables/useHrAgentRun.ts`
- `hr-frontend/src/composables/useHrAgentRun.test.ts`
- `.spec/.../pipeline-state.json` (+ report/evidence written with TASK)

Note: Restored prior `TASK-HARS-005-evidence.json` blob from `base_tree` before final scope pass (unrelated drift from base_tree, same class of fix as earlier TASK-005 notes).

No forbidden or out-of-scope business files. No package.json/lockfile/backend/AIChatView changes.

## 5. SPEC Comparison Result

Aligned with:

- FR-012: unified HR Agent runtime composable + pure TS reducer
- FR-013: ignore stale run ids and already-applied sequence numbers
- AC-002 (foundation): hydrate from active snapshot + resume after `last_event_seq` (view mount wiring is TASK-007/008)
- AC-007 (foundation): single runtime path ready for all entry points (migration is TASK-007)
- AC-010 (partial): Vitest coverage for reducer and composable behavior

Not claimed:

- FR-014 view migration (TASK-007)
- FR-015 full page-refresh wiring in AIChatView (TASK-008)
- FR-016 history reconciliation (later TASKs)

## 6. SDD Comparison Result

Matches SDD §3 / §5 / §9:

| Design element | Implementation |
| --- | --- |
| HR frontend API wrapper | `api/agentRun.ts` for create/get/active/subscribe/cancel/confirm |
| `useHrAgentRun` composable | Subscription lifecycle owner; dispose ≠ cancel |
| Pure TS reducer | `utils/hrAgentRunReducer.ts`, framework-independent |
| Gateway endpoints | Paths match TASK-005 REST/SSE routes |
| Event types | Constants aligned with logic-service / SDD §6 |
| Error handling | Stale run/seq ignored; abort is subscription-only |

## 7. Acceptance Comparison Result

| Criterion | Result |
| --- | --- |
| Typed API helpers for create/get/active/subscribe/cancel/confirm | Pass |
| Framework-independent pure TS reducer | Pass |
| Ignores stale run ids and duplicate/stale seq | Pass (tests) |
| Handles delta, snapshot, status, confirmation, result, error, cancel, completion | Pass (tests) |
| Vue composable: create, subscribe, hydrate, cancel, confirm, cleanup | Pass |
| Aborting subscription does NOT call cancel unless explicit | Pass (test) |
| Vitest for reducer + composable | Pass (17 new tests) |

## 8. Test Commands and Results

| Command | Result |
| --- | --- |
| `git diff --name-only` | OK (prior feature dirty tree + TASK-006 frontend files) |
| `TASK_BASE_TREE=157c9702fa5d78169d7265ee5776c45513d5ac74 bash .spec/.../check-task-scope.sh TASK-HARS-006` | PASS (exit 0) |
| `bash .spec/hr-agent-resumable-stream/scripts/agent-check.sh TASK-HARS-006` | PASS (exit 0) |
| `pnpm --filter hr-frontend typecheck` (via agent-check / local) | PASS |
| `pnpm --filter hr-frontend test` | PASS — 26 tests (10 reducer + 7 composable + 9 existing token) |

## 9. Knowledge Impact

- **result**: `update_required` (knowledge files out of scope for this TASK; no `.knowledge` edits)
- **review**:
  - `.knowledge/architecture/frontend-apps.md` — **UNCHANGED** for architecture model (Vue app structure, request wrappers, API/types co-location). Docs do not yet mention durable run runtime composable/reducer; update later when scope allows.
  - `.knowledge/architecture/api-contracts-and-gateway.md` — **UNCHANGED** for gateway boundary. Still lags durable run HTTP paths (known lag from TASK-005); frontend now consumes those paths.
  - `.knowledge/runbooks/frontend-validation.md` — **UNCHANGED** for validation procedure; typecheck + vitest remain the correct post-change checks used here.
- **coverageGap**: false
- No STALE/CONFLICT verdicts (debt via `update_required` + UNCHANGED lag notes only).

## 10. Risks

- SSE subscribe relies on browser `fetch` streaming; reverse proxies must not buffer long-lived connections (gateway already sets `X-Accel-Buffering: no`).
- `startRun`/`hydrate*` auto-subscribe is fire-and-forget; callers that need strict await should call `subscribe` explicitly or pass `{ autoSubscribe: false }`.
- Confirmation payload parsing accepts both structured `confirmation` fields and `payload_json` string; malformed JSON falls back to `{ required: true, raw_json }`.
- AIChatView still uses legacy `sendMessageStream` until TASK-007; dual paths coexist by design (FR-017).

## 11. Follow-up Items

- TASK-HARS-007: migrate AIChatView entry points to `useHrAgentRun`
- TASK-HARS-008: refresh recovery end-to-end validation
- Optional knowledge updates for frontend-apps / api-contracts durable run runtime
- Do not mark pipeline complete; do not implement TASK-007 in this report

## 12. Whether the Next TASK Can Start

**No.** Implementation complete and ready for **self-review**. Next TASK (TASK-HARS-007) must not start until TASK-HARS-006 review passes and confirmation is recorded.
