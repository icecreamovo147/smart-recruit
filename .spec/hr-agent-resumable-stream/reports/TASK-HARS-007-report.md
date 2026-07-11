# TASK Report - TASK-HARS-007

## 1. TASK ID

TASK-HARS-007

## 2. Modified File List

- `hr-frontend/src/views/hr/AIChatView.vue` (modify)
- `hr-frontend/src/composables/useHrAgentRun.ts` (modify — waitUntilSettled)
- `hr-frontend/src/components/hr/ai/agentRunChatFlow.ts` (create)
- `hr-frontend/src/components/hr/ai/agentRunChatFlow.test.ts` (create)
- `.spec/hr-agent-resumable-stream/reports/TASK-HARS-007-report.md`
- `.spec/hr-agent-resumable-stream/reports/TASK-HARS-007-evidence.json`
- `.spec/hr-agent-resumable-stream/pipeline-state.json`

Unchanged allowed files:

- `hr-frontend/src/composables/useHrAgentRun.test.ts` — existing composable tests still pass; settlement/entry-path coverage added under `agentRunChatFlow.test.ts`
- `hr-frontend/src/views/hr/AIChatView.test.ts` — not created; shared flow helper tests cover submit + skill confirm entry paths with mocked agentRun API

## 3. Change Summary by File

| File | Summary |
| --- | --- |
| `AIChatView.vue` | Removes direct `sendMessageStream` orchestration for all five entry points. Wires `useHrAgentRun` + shared durable chat flow. Normal submit, retry, route-created analysis, candidate option analysis, and skill confirmation now create/confirm durable runs and reduce events into the existing message list, process trace, loading, errors, candidate options, and skill selection UI. Page leave / session switch disposes subscription only; stop button issues explicit cancel. Legacy stream helper remains in `api/ai.ts` for rollout (FR-017 / AC-008). |
| `useHrAgentRun.ts` | Adds `waitUntilSettled` for terminal / waiting_confirmation / aborted outcomes used by chat entry flows. |
| `agentRunChatFlow.ts` | Shared create/confirm chat adapters: client request ids, confirmation payload normalization (including nested `agent_skill_selection`), UI state binder (assistant/process deltas & snapshots, model, context usage, candidate options), and execute helpers. |
| `agentRunChatFlow.test.ts` | Vitest coverage for submit path, skill-confirm path, waiting_confirmation parking, confirmation mapping, and `waitUntilSettled`. |
| report/evidence/pipeline-state | Harness artifacts; `current_phase=review`. |

## 4. Scope Check Result

**PASS** — TASK-local changed files match allowed globs.

```bash
TASK_BASE_TREE=d3d2d1f7b929e4fbea03a0d0fec222f3817c7ad7 \
  bash .spec/hr-agent-resumable-stream/scripts/check-task-scope.sh TASK-HARS-007
# exit_code=0
```

Allowed hits:

- `hr-frontend/src/views/hr/AIChatView.vue`
- `hr-frontend/src/composables/useHrAgentRun.ts`
- `hr-frontend/src/components/hr/ai/agentRunChatFlow.ts`
- `hr-frontend/src/components/hr/ai/agentRunChatFlow.test.ts`
- `.spec/.../pipeline-state.json` (+ report/evidence written with TASK)

No forbidden or out-of-scope business files. No package.json/lockfile/backend changes.

## 5. SPEC Comparison Result

Aligned with:

- FR-014: migrate normal submit, retry, route candidate analysis, candidate option actions, skill confirmation to unified runtime
- FR-012/013: continues using composable + pure reducer (stale run/seq handling unchanged)
- AC-007: all five entry points share the same durable runtime path (`executeCreateChatRun` / `executeConfirmChatRun`)
- AC-008 / FR-017: legacy `sendMessageStream` remains available in `api/ai.ts`; page no longer calls it
- FR-015 refresh restore: **not implemented** (TASK-HARS-008)

## 6. SDD Comparison Result

Matches SDD §3 / §8 / §13:

| Design element | Implementation |
| --- | --- |
| Browser as event subscriber | create/confirm + subscribe; dispose ≠ cancel |
| Unified runtime | `useHrAgentRun` + shared chat flow helper |
| UI stability | Existing message list, typewriter, process, options, skill cards retained |
| Compatibility | Legacy stream API left in place during rollout |
| Implementation boundary | HR frontend view/composable/components only |

## 7. Acceptance Comparison Result

| Criterion | Result |
| --- | --- |
| Normal submit uses unified runtime | Pass |
| Retry uses unified runtime | Pass |
| Route-created candidate analysis uses unified runtime | Pass |
| Candidate option actions use unified runtime | Pass |
| Skill confirmation uses unified runtime | Pass (prefer `confirm` on same run; create fallback if run id lost) |
| UI stable for messages, process, loading, errors, options | Pass (binders map reducer state to existing UI) |
| Duplicated stream orchestration removed/reduced | Pass (`sendMessageStream` removed from view) |
| Focused tests cover ≥2 formerly separate entry paths | Pass (submit + skill confirm + waiting_confirmation) |

## 8. Test Commands and Results

| Command | Result |
| --- | --- |
| `git diff --name-only` | OK (feature dirty tree + TASK-007 files) |
| `TASK_BASE_TREE=d3d2d1f7b929e4fbea03a0d0fec222f3817c7ad7 bash .spec/.../check-task-scope.sh TASK-HARS-007` | PASS (exit 0) |
| `bash .spec/hr-agent-resumable-stream/scripts/agent-check.sh TASK-HARS-007` | PASS (exit 0) |
| `pnpm --filter hr-frontend typecheck` | PASS (~5.8s) |
| `pnpm --filter hr-frontend test` | PASS — 34 tests (8 new flow + 7 composable + 10 reducer + 9 token) |

## 9. Knowledge Impact

- **result**: `update_required` (knowledge files out of scope; no `.knowledge` edits)
- **review**:
  - `.knowledge/architecture/frontend-apps.md` — **UNCHANGED** for Vue app structure/API co-location. Still does not document durable HR Agent run runtime migration of AIChatView entry points; update later when scope allows.
  - `.knowledge/runbooks/frontend-validation.md` — **UNCHANGED** for validation procedure; typecheck + vitest remain the correct post-change checks used here.
- **coverageGap**: false
- No STALE/CONFLICT verdicts (debt via `update_required` + UNCHANGED lag notes only).

## 10. Risks

- Candidate option action now creates an analysis session first (required for `session_id`), then runs durable analysis — behavior is closer to the route analysis path and may differ slightly from the old application_id-only stream create.
- Skill confirmation prefers `confirm` on the parked run; if composable state was reset without refresh recovery (TASK-008), falls back to create with confirmation flags.
- Intermediate stream status waiting-text strings from legacy SSE status events are not fully mirrored; default 分析中/响应中 remains until content arrives.
- Stop button now cancels the backend run; page leave / session switch still only disposes the subscription.
- Full refresh recovery of an in-flight run is intentionally deferred to TASK-HARS-008.

## 11. Follow-up Items

- TASK-HARS-008: refresh recovery, reconnect with last sequence, end-to-end validation
- Optional knowledge updates for frontend-apps durable chat runtime
- Do not mark pipeline complete; do not implement TASK-008 in this report

## 12. Whether the Next TASK Can Start

**No.** Implementation complete and ready for **self-review**. Next TASK (TASK-HARS-008) must not start until TASK-HARS-007 review passes and confirmation is recorded (TASK-008 also requires human confirmation).
