# TASK Report - TASK-009

## 1. TASK ID

TASK-009 - Verify Frontend Contract Compatibility

## 2. Status

Completed.

Independent self-review round 1 returned `不通过`; fixes were applied so gateway-shaped `process.delta` events with top-level `result_metadata` are also normalized to reducer-compatible `run.result` events, and regression coverage was added.

Independent self-review round 2 returned `通过`.

Human confirmation was not required for TASK-009.

## 3. Modified File List

- `.spec/ai-agent-runtime-recovery/pipeline-state.json`
- `.spec/ai-agent-runtime-recovery/reports/TASK-009-report.md`
- `.spec/ai-agent-runtime-recovery/reports/TASK-009-evidence.json`
- `hr-frontend/src/api/agentRun.ts`
- `hr-frontend/src/api/agentRun.test.ts`
- `hr-frontend/src/types/agentRun.ts`

`git diff --name-only` does not list untracked files before staging; the new API test file is included from `git status --short` and verified by the TASK scope checker.

## 4. Change Summary by File

- `hr-frontend/src/api/agentRun.ts`: adds `normalizeAgentRunEvent` at the SSE boundary. Durable run `process.delta` events carrying `payload_json.result_metadata` are normalized into non-terminal `run.result` events so the existing reducer/UI can consume restored context usage metadata without changing out-of-scope reducer utilities.
- `hr-frontend/src/api/agentRun.test.ts`: adds focused Vitest coverage for context usage metadata normalization and unchanged process events without metadata.
- `hr-frontend/src/types/agentRun.ts`: adds optional `event_message` to durable event types for restored runtime event metadata compatibility.
- `pipeline-state.json`: records TASK-009 base tree and non-required human confirmation.

## 5. Scope Check Result

Passed.

Command:

```bash
TASK_BASE_TREE=3408c36d27dffeb8a9609c066cea63d124e85c45 bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh TASK-009
```

Result:

```text
scope_result: PASS (TASK-009)
```

## 6. SPEC Comparison Result

Passed.

The change preserves public routes and payloads while improving HR durable run frontend compatibility with restored runtime metadata. No backend, proto, schema, package, lockfile, global config, root docs, or UI redesign changes were introduced.

## 7. SDD Comparison Result

Passed.

The fix is applied at the HR frontend API/SSE adapter boundary, matching the compatibility strategy of preserving existing UI reducers and views while normalizing restored backend metadata into already supported frontend event shapes.

## 8. Acceptance Comparison Result

Passed.

- HR AI Chat stream and durable run UI can consume restored context usage metadata through existing result metadata handling.
- Agent Skill selection and candidate option parsing were rechecked and left unchanged because existing types/composables already support the restored payloads.
- MCP, Embedding, and Semantic Debug admin views were rechecked and left unchanged because existing types/views already support runtime success/failure, provider/model/dim/latency, score breakdown, and fallback fields.
- Candidate AI API/types were rechecked and left unchanged because no restored backend mismatch was found in the candidate stream/non-stream contract.
- No unrelated UI redesign or global frontend config change was introduced.

## 9. Test Commands and Results

| Command | Result |
|---|---|
| `pnpm --filter hr-frontend test -- --run src/api/agentRun.test.ts` | Passed; due current script argument forwarding, Vitest ran all 9 HR test files / 69 tests |
| `pnpm --filter hr-frontend typecheck` | Passed |
| `git diff --name-only` | Passed; command does not list untracked new API test file before staging |
| `git status --short hr-frontend/src/api/agentRun.test.ts hr-frontend/src/api/agentRun.ts hr-frontend/src/types/agentRun.ts hr-frontend/src/utils/hrAgentRunReducer.ts` | Passed; confirms no remaining reducer utility diff after reverting out-of-scope exploration |
| `TASK_BASE_TREE=3408c36d27dffeb8a9609c066cea63d124e85c45 bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh TASK-009` | Passed |
| `TASK_BASE_TREE=3408c36d27dffeb8a9609c066cea63d124e85c45 bash .spec/ai-agent-runtime-recovery/scripts/agent-check.sh` | Passed |
| `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 3408c36d27dffeb8a9609c066cea63d124e85c45` | Passed; result `update_required` |

`pnpm --filter user-frontend typecheck` was not required because no user frontend files were touched.

## 10. Knowledge Impact

Result: `update_required`.

Routed active knowledge was reviewed: `frontend-apps`, `frontend-validation`, `frontend-menu-consistency`, `agent-runtime`, and `resume-intelligence`. Impact detection also matched broad frontend/service overview routes.

No `.knowledge/**` files were edited because they are outside TASK-009 scope. Knowledge debt is recorded for HR durable run frontend metadata normalization and frontend validation evidence.

## 11. Human Confirmation

Not required.

## 12. Dev Reference

TASK-009 used current restored backend behavior from TASK-002 through TASK-008 as the contract source and compared it against the existing frontend API/types/views. No additional monolith runtime dependency was introduced.

## 13. Risks

- The normalization is intentionally narrow: only `process.delta` events carrying result metadata are converted to non-terminal `run.result` events. Other process events stay unchanged.
- The focused test imported the API module, so it exercises the exported helper but not a browser `fetch` stream end-to-end.

## 14. Follow-up Items

- Update `.knowledge/**` frontend validation notes in a future knowledge-maintenance scope.

## 15. Repair Summary

### Failed Review Round 1

Independent reviewer Descartes reported:

- Gateway-shaped `process.delta` events with top-level `result_metadata` stayed as `process.delta`, so the existing reducer ignored context usage metadata.
- Focused test coverage only exercised `payload_json.result_metadata`, not the real gateway top-level metadata shape.

### Round 1 Fix Summary

- `normalizeAgentRunEvent` now converts `process.delta` to `run.result` whenever result metadata exists from either top-level gateway fields or parsed `payload_json`.
- `agentRun.test.ts` now covers gateway-shaped events with top-level `result_metadata`.

## 16. Review Rounds

- Round 1: `不通过`; gateway-shaped top-level result metadata was not normalized.
- Round 2: `通过`; no blocking findings.

## 17. Whether the Next TASK Can Start

Yes. TASK-009 checks passed and independent self-review returned `通过`; TASK-010 can start.
