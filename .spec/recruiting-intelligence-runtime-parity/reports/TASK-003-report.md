# TASK Report - TASK-003

## 1. TASK ID

`TASK-003` — Restore Resume Profile Extraction. Mode: `fix-check-failures`, repair round 1.

Baseline: `base_sha=98165e7cdac06c0aa2a0685ecf1ad7c00a3eb518`, `base_tree=9fdbe164e5c46b96a17d46f6722ab846d09cf33e`.

## 2. Modified File List

- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/resume_profile.go`
- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/resume_profile_test.go`
- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/runtime_policy.go`
- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/runtime_policy_test.go`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_recruiting_generation_test.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_resume_profile_test.go`
- `.spec/recruiting-intelligence-runtime-parity/reports/TASK-003-report.md`
- `.spec/recruiting-intelligence-runtime-parity/reports/TASK-003-evidence.json`
- `.spec/recruiting-intelligence-runtime-parity/pipeline-state.json` (root-owned; Fixer did not edit)
- `.spec/recruiting-intelligence-runtime-parity/task-scope.json` (root-owned approved test exception; Fixer did not edit)

## 3. Change Summary by File

- `resume_profile.go`: retains the database-prompt extractor and adds exact required-key presence/type enforcement for every active resume schema object layer. It also recognizes the real Commons `AIError{Type: AIEmptyReply}` through the wrapped Runtime error as `empty_response`.
- `resume_profile_test.go`: adds missing top-level, education, experience, project, and skill key cases, including scalar, boolean, and array presence plus typed-empty/null behavior.
- `runtime_policy.go` / test: add a feature-total resume deadline and a deterministic parsing window that reserves 20% up to two seconds for persistence, including shorter parent deadlines.
- `native_servers.go`: runs extraction with the shorter parsing context, saves with the total feature context, and refuses to start a save after parent/total cancellation.
- `native_resume_profile_test.go`: proves model parsing-window exhaustion still leaves a live save budget, parent cancellation prevents saving, and legacy fixtures satisfy rather than weaken the active schema.
- `native_recruiting_generation_test.go`: uses the real NativeStore to prove all profile scalars, dates, child rows, and nested JSON mappings; a SQLite child-insert trigger proves rollback of current/version, parse run, profile, and all children. It also exercises the real NativeStore → Commons structured adapter → wrapped Runtime → resume fallback chain for `AIEmptyReply`. Repair round 2 registers `sql.DB` cleanup for both SQLite paths owned by this file, so the fixed-name shared-memory adapter fixture is destroyed between repeated test iterations.
- report/evidence: preserve review round 1 and the user-approved test-only exception, record repair results, 19 knowledge verdicts, and a consistent pending-re-review timeline.

## 4. Scope Check Result

PASS. `check-task-scope.sh TASK-003` classified the Harness as current. The newly used persistence file is exactly the user-authorized test-only exception. Forbidden and out-of-scope lists are empty. No production persistence, schema/migration/db.sql, Proto, dependency, shared configuration, public API, auth, frontend, deployment, CI, root docs, or knowledge file changed.

Approved exception (preserved exactly from pipeline state):

- `approved_object`: `smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_recruiting_generation_test.go test-only scope expansion`
- `approved_at`: `2026-07-15T06:11:38Z`
- `scope`: `TASK-003 may modify only smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_recruiting_generation_test.go outside its original scope; production persistence, schema, Proto, dependencies, and public APIs remain forbidden.`
- `reason`: `Independent review requires real NativeStore transaction rollback and full resume child/date/JSON mapping evidence.`

## 5. SPEC Comparison Result

PASS against FR-002, FR-004, FR-005, FR-010, NFR persistence-budget requirements, AC-001, AC-003, AC-007, AC-008, and AC-010. Every active prompt schema key is required at every layer while unknown values remain representable only with the prompt's typed empty values. Empty-response, timeout, fallback, cancellation, and transactional behavior are explicitly covered.

## 6. SDD Comparison Result

PASS against SDD sections 3, 6.2, 7, 8, 9, 11, and 13. Parsing and persistence share one total deadline with a deterministic save reserve; transport remains orchestration-only; NativeStore production code and external contracts remain unchanged.

## 7. Acceptance Comparison Result

PASS for implementation and checks. Exactly one complete JSON object is accepted; `skills:["Go"]`, missing keys, invalid typed values, malformed dates, and empty material profiles fail schema validation. Fallback is policy-controlled. Fallback-disabled or canceled parsing creates no snapshot. Real NativeStore tests prove complete mapping and whole-snapshot rollback.

## 8. Test Commands and Results

| Command | Result |
|---|---|
| `GOWORK=off go test -count=1 ./internal/application/recruiting_intelligence ./internal/interfaces/grpc ./internal/infrastructure/persistence` | PASS |
| `GOWORK=off go test -count=5 ./internal/interfaces/grpc -run 'TestParseResumeProfile(ModelWindowExpiryLeavesPersistenceBudget|ParentCancellationDoesNotSaveFallback)$'` | PASS |
| `GOWORK=off go test ./...` in AI Agent | PASS |
| `GOWORK=off go test -count=10 ./internal/infrastructure/persistence -run TestNativeStoreStructuredAdapterClassifiesAIEmptyReplyForResumeFallback` | PASS; exact R2 reproduction condition |
| `GOWORK=off go test -count=2 ./internal/infrastructure/persistence -run 'TestNativeStore(SaveRecruitingResumeProfileDraftVersionsAndChildren|SaveRecruitingResumeProfileDraftRollsBackEveryTableOnChildFailure|StructuredAdapterClassifiesAIEmptyReplyForResumeFallback)$'` | PASS; related real-persistence tests |
| `GOWORK=off go test ./...` in AI Agent after repair round 2 | PASS |
| `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 9fdbe164e5c46b96a17d46f6722ab846d09cf33e` | PASS; 19 routed active documents |
| `git diff --name-only`; `git diff --check` | PASS |
| `bash .spec/recruiting-intelligence-runtime-parity/scripts/check-task-scope.sh TASK-003` | PASS |
| `bash .spec/recruiting-intelligence-runtime-parity/scripts/agent-check.sh` | PASS, including AI Agent and Commons AI tests |

The first post-edit focused run failed because the new presence errors were initially classified as JSON and a legacy test adapter omitted scalar keys. Both test-discovered issues were corrected without weakening production validation; every final rerun above passes.

An additional exploratory whole-persistence-package `-count=2` run found the same fixed-DSN cleanup defect in three pre-existing tests in `llm_runtime_prompt_test.go`. That shared helper and those tests are expressly outside the R2 finding and the user-authorized TASK-003 repair scope. The exact R2 test passes at `-count=10`, and every real-persistence test owned by the authorized file passes at `-count=2`.

## 9. Knowledge Impact

Result: `candidate_required`; detector exit code `0`; coverage gap `false`. The detector routed 19 active documents. `agent-runtime` and `resume-intelligence` remain `CANDIDATE` for TASK-007. The newly routed persistence/migration documents are `UNCHANGED`: only a test was strengthened, and no production adapter, schema, migration, ownership, or public contract changed. All other routed documents are unrelated and `UNCHANGED` after source-reference review.

## 10. Risks

- The deadline reserve is deterministic (20% of available feature time, capped at two seconds); extremely short caller deadlines can still be insufficient for a database write, which correctly returns non-success.
- SQLite triggers are deterministic fault injection for transaction semantics; production MySQL implementation remains covered by the same unchanged GORM transaction boundary.
- Live credentials are neither required nor recorded; the real adapter-chain test uses a local OpenAI-compatible HTTP fixture.

## 11. Follow-up Items

- Independent review round 3 returned `verdict: 通过`.
- TASK-007 should update the two deferred knowledge candidates after the complete pipeline stabilizes.
- The shared `newStructuredRuntimeTestStore` helper still lacks general cleanup for its own pre-existing direct callers; this is out of TASK-003 scope and should be handled separately.

## 12. Whether the Next TASK Can Start

Yes. All required checks pass and independent review round 3 returned `verdict: 通过`.

## 13. Independent Review Round 1

Reviewer `/root/task003_reviewer_r1` returned `verdict: 不通过` with H1 (missing required-key presence tracking), M1 (no reserved persistence budget), M2 (production `AIEmptyReply` misclassification), M3 (missing real NativeStore mapping/rollback proof), and M4 (inconsistent evidence timeline). This history is preserved.

## 14. Independent Review Round 2

Reviewer `/root/task003_reviewer_r2` returned `verdict: 不通过` with one Medium finding, R2-M1: the new real-adapter test called `newStructuredRuntimeTestStore`, whose SQLite DSN is based on fixed `t.Name()` and whose underlying database was not closed. Under the exact reviewer command, iteration one passed and later iterations failed with duplicate `llm_providers.id`. All R1 findings, scope, full tests, Harness, knowledge, and privacy checks passed. Root recorded the reviewer output receipt at `2026-07-15T06:43:47Z`; the review history is retained rather than overwritten.

## Repair Summary

### Failed Check

Independent review round 1 findings H1, M1, M2, M3, and M4.

### Root Cause

The original decoder distinguished zero values from only some omitted arrays; Runtime wrapped Commons AI errors without inspecting `AIEmptyReply`; the feature timeout was owned entirely by parsing; persistence assertions used a fake instead of NativeStore; and evidence ended before all recorded checks.

### Files Changed

The seven production/test files and two TASK report artifacts listed above. The two root-owned scope/state files were not edited by the Fixer.

### Fix Summary

All five findings are repaired in scope. H1 uses exact per-layer presence tracking. M1 separates parse and save contexts under one total deadline. M2 follows the real structured adapter error chain. M3 proves complete NativeStore mapping and rollback after a child insert failure. M4 records checks before repair completion and preserves review/exception history.

### Re-run Commands

Focused tests, repeated deadline/cancellation tests, AI Agent full tests, impact detection, diff/scope checks, agent-check, and pending-review evidence validation.

### Re-run Results

All final commands pass. Review status remains pending independent round 2 rather than self-approved.

### Remaining Risks

No known blocking implementation risk. Another self-review is required by the Harness.

## Repair Summary - Round 2

### Failed Check

R2-M1: `GOWORK=off go test -count=10 ./internal/infrastructure/persistence -run TestNativeStoreStructuredAdapterClassifiesAIEmptyReplyForResumeFallback` failed after its first iteration with `UNIQUE constraint failed: llm_providers.id`.

### Root Cause

The shared test helper uses a fixed, named, shared-memory SQLite DSN derived from `t.Name()`. The new TASK-003 test did not close the helper's underlying `sql.DB`, so the named in-memory database survived into later `-count` iterations.

### Files Changed

- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_recruiting_generation_test.go`
- `.spec/recruiting-intelligence-runtime-parity/reports/TASK-003-report.md`
- `.spec/recruiting-intelligence-runtime-parity/reports/TASK-003-evidence.json`

### Fix Summary

The real-adapter test now obtains and closes the helper-owned underlying `sql.DB` through `t.Cleanup`. The file-local real-persistence database helper also registers the same cleanup, keeping every SQLite instance owned by this authorized test file bounded to its test lifecycle. No shared helper or production persistence code changed.

### Re-run Commands

The exact R2 `-count=10` reproduction, the three related real-persistence tests at `-count=2`, full AI Agent tests, impact detection, diff/scope checks, agent-check, and pending-review evidence validation.

### Re-run Results

All required repair-round-2 commands pass. Review status remains pending independent round 3 rather than self-approved.

### Remaining Risks

Pre-existing direct callers of the shared fixed-DSN helper still do not close their databases and fail a whole-package `-count=2` exploratory run. Repairing them requires changes outside the only file authorized for R2-M1, so this does not widen TASK-003 and is recorded as follow-up debt.

## 15. Final Independent Review

Reviewer `/root/task003_reviewer_r3` independently verified every R1/R2 repair, repeated real-adapter and persistence tests, full module tests, scope, Harness, evidence, knowledge impact, privacy, exception boundaries, and timeline. It found no issues and ended with `verdict: 通过`. The unrelated repeated-test failures were proven to come from unchanged baseline helper callers outside TASK-003.
