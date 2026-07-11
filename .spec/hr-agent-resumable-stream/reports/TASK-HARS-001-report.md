# TASK Report - TASK-HARS-001

## 1. TASK ID

TASK-HARS-001

## 2. Modified File List

- `logic-grpc-service/migrations/000050_add_resumable_agent_runs.sql`
- `logic-grpc-service/migrations/000050_add_resumable_agent_runs.down.sql`
- `logic-grpc-service/model/model.go`
- `logic-grpc-service/repository/repo_test_helper.go`
- `logic-grpc-service/repository/agent_run_repo_test.go`
- `db.sql`
- `.spec/hr-agent-resumable-stream/reports/TASK-HARS-001-report.md`
- `.spec/hr-agent-resumable-stream/reports/TASK-HARS-001-evidence.json`
- `.spec/hr-agent-resumable-stream/pipeline-state.json`

## 3. Change Summary by File

| File | Summary |
| --- | --- |
| `000050_add_resumable_agent_runs.sql` | Extends `agent_runs` with idempotency, snapshot, sequence, and cancel fields; adds `agent_run_events`; adds `ai_chat_sessions.active_run_id` and `ai_chat_history.agent_run_id` with required indexes. |
| `000050_add_resumable_agent_runs.down.sql` | Rollback removes only this feature's columns, indexes, and `agent_run_events`. Named with runner convention `*.down.sql`. |
| `model/model.go` | Aligns `AgentRun`, `AIChatSession`, `AIChatHistory`; adds `AgentRunEvent` model and table name. |
| `repo_test_helper.go` | AutoMigrates `AgentRunEvent` in repository test DB helper. |
| `agent_run_repo_test.go` | Adds focused persistence test for idempotency unique key, event seq uniqueness, active run pointer, and history association. |
| `db.sql` | Aligns baseline schema with migration 000050. |
| report/evidence/pipeline-state | Harness artifacts for this TASK. |

## 4. Scope Check Result

PASS. `bash .spec/hr-agent-resumable-stream/scripts/check-task-scope.sh TASK-HARS-001` against refreshed `base_tree=dd2280e566fadae98b97ae5edeef6ef6572108bb` (orchestrator F2 allowlist correction absorbed) reported all TASK-local changes ALLOWED.

## 5. SPEC Comparison Result

Aligned with FR-001/002 (idempotency key), FR-005 (ordered events), FR-006 (snapshot fields), FR-015/016 (active run + history association), and AC-001–AC-003/AC-010 persistence prerequisites. No runtime behavior implemented (later TASKs).

## 6. SDD Comparison Result

Matches SDD §4: extend `agent_runs`, add `agent_run_events`, align `ai_chat_sessions.active_run_id`, align history `agent_run_id` (table name verified as `ai_chat_history`), and add the four required index classes. Rollback limited to this feature's schema.

## 7. Acceptance Comparison Result

- Forward + rollback migrations: yes
- `db.sql` aligned: yes
- Go models updated: yes
- Idempotent create lookup support: unique `(hr_id, session_id, client_request_id)`
- Active run lookup: `active_run_id` + `(session_id, status)` index
- Final assistant association: `ai_chat_history.agent_run_id` + index
- Event replay index: unique `(run_id, seq)`
- Rollback removes only this feature: yes
- Down migration filename matches runner `*.down.sql` convention: yes (repair F1)

## 8. Test Commands and Results

| Command | Result |
| --- | --- |
| `git diff --name-only` | OK |
| `cd logic-grpc-service && go test ./migration -run TestLoadMigrations -count=1` | PASS (loaded 50 migrations; down file no longer treated as duplicate up) |
| `cd logic-grpc-service && go test ./migration -run 'TestLoadMigrations\|Test.*Down.*' -count=1` | PASS |
| `bash .spec/hr-agent-resumable-stream/scripts/check-task-scope.sh TASK-HARS-001` | PASS |
| `bash .spec/hr-agent-resumable-stream/scripts/agent-check.sh TASK-HARS-001` | PASS (`service`/`repository` tests ok) |
| Focused Go test `TestAgentRunResumableSchemaPersistence` | PASS (via agent-check repository suite) |

MySQL consistency test (`-tags=mysql`) not run: requires `MYSQL_DSN` and is outside this TASK's required local checks.

`agent-check.sh` was not modified to include `./migration` (scripts/ outside allowedFiles); migration load/down tests were run and recorded manually.

## 9. Knowledge Impact

Result: `update_required` (knowledge edits out of scope).

| Document | Verdict | Notes |
| --- | --- | --- |
| `.knowledge/architecture/persistence-and-migrations.md` | UNCHANGED | Process guidance still accurate; `last_verified` should be refreshed after this schema lands. |
| `.knowledge/runbooks/protobuf-and-migration-change.md` | UNCHANGED | Migration path followed; down file uses `*.down.sql`. |
| `.knowledge/pitfalls/migration-model-drift.md` | UNCHANGED | Prevention steps followed (migration + `db.sql` + model + test helper). |

Debt: optional later update to mention `agent_run_events` / resumable snapshot fields; `agent-runtime.md` may need a later TASK review when runtime behavior lands.

## 10. Risks

1. Unique `(hr_id, session_id, client_request_id)` allows multiple NULL `client_request_id` rows (MySQL unique NULL behavior); legacy rows remain valid.
2. MySQL end-to-end migration vs `db.sql` consistency not executed in this environment.

## 11. Follow-up Items

- TASK-HARS-002 should implement repositories/state helpers on this schema.
- Refresh knowledge `last_verified` when a knowledge-scoped TASK allows it.

## 12. Whether the Next TASK Can Start

Repair complete. Ready for re-review (round 1). Do **not** start TASK-HARS-002 until review passes and pipeline advances.

## Repair Summary

### Failed Check

Self-review round 1 findings F1 (Critical) and F2 (Critical): down migration named `*_down.sql` instead of runner convention `*.down.sql`, causing `TestLoadMigrations` failure / duplicate v50 up load; allowlist still referenced the wrong name until orchestrator correction.

### Root Cause

Filename followed an incorrect `_down.sql` suffix; migration runner only skips/resolves `*.down.sql`.

### Files Changed

- Renamed `logic-grpc-service/migrations/000050_add_resumable_agent_runs_down.sql` → `000050_add_resumable_agent_runs.down.sql`
- Verified `.spec/hr-agent-resumable-stream/task-scope.json` allowlist already lists `000050_add_resumable_agent_runs.down.sql` (orchestrator F2)
- Refreshed `pipeline-state.json` `base_tree` to `dd2280e566fadae98b97ae5edeef6ef6572108bb` so the allowlist-only orchestrator edit is baseline (no further scope expansion)
- Updated report + evidence

### Fix Summary

F1 fixed by rename. F2 verified (allowlist correct). Down path `000050_add_resumable_agent_runs.down.sql` exists and resolves from up name via runner convention. `TestLoadMigrations` loads exactly 50 migrations.

### Re-run Commands

- `cd logic-grpc-service && go test ./migration -run TestLoadMigrations -count=1`
- `cd logic-grpc-service && go test ./migration -run 'TestLoadMigrations|Test.*Down.*' -count=1`
- `bash .spec/hr-agent-resumable-stream/scripts/check-task-scope.sh TASK-HARS-001`
- `bash .spec/hr-agent-resumable-stream/scripts/agent-check.sh TASK-HARS-001`
- `node .agents/skills/spec-harness/scripts/validate-evidence.mjs --file .spec/hr-agent-resumable-stream/reports/TASK-HARS-001-evidence.json --allow-pending-review --require-knowledge-impact`

### Re-run Results

All listed commands PASS (evidence validation run after this report write).

### Remaining Risks

None related to F1/F2. Prior MySQL NULL uniqueness and unrun MySQL consistency checks remain as non-blocking notes.
