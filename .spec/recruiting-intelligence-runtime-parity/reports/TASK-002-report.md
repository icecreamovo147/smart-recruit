# TASK Report - TASK-002

## 1. TASK ID

`TASK-002` — Runtime Policy Wiring. Review round 1 failed; R1-H1 and R1-M1 have been repaired and an independent round-2 review is required.

Baseline: `base_sha=98165e7cdac06c0aa2a0685ecf1ad7c00a3eb518`, `base_tree=338c74c96c20f6f89b2ee8c4166347cbdd997468`.

## 2. Modified File List

- `smart-recruit-ai-agent-service/cmd/ai-agent-service/main.go`
- `smart-recruit-ai-agent-service/cmd/ai-agent-service/main_test.go`
- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/runtime_policy.go`
- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/runtime_policy_test.go`
- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/structured_runtime.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_recruiting_policy_test.go`
- `.spec/recruiting-intelligence-runtime-parity/pipeline-state.json` (root-owned TASK transition present in the baseline diff; Fixer did not edit it)
- `.spec/recruiting-intelligence-runtime-parity/reports/TASK-002-report.md`
- `.spec/recruiting-intelligence-runtime-parity/reports/TASK-002-evidence.json`

## 3. Change Summary by File

- `main.go`: maps the seven existing settings into the immutable policy.
- `main_test.go`: now tests all five boolean settings independently for nil/default and explicit false, and both timeout settings independently for default and explicit values.
- `runtime_policy.go`: adds immutable `CandidateMatchExecutionPlan` intent. `candidate_match` is the overall gate; semantic chooses enhanced versus deterministic primary; shadow chooses the opposite scorer.
- `runtime_policy_test.go`: covers all candidate-match semantic/shadow combinations, independent overall disablement, deterministic-only behavior, enhanced shadow structured calls, enhanced primary calls, timeouts, cancellation, and fallback flag preservation.
- `structured_runtime.go`: gates candidate structured calls on `candidate_match && (semantic || shadow)` through the execution plan rather than treating `semantic=false` as total failure.
- `native_servers.go` and `native_recruiting_policy_test.go`: retain the original immutable dependency injection coverage.
- report/evidence: preserve round-1 findings and record repair/check results truthfully.

## 4. Scope Check Result

PASS. `check-task-scope.sh TASK-002` classified the Harness as current and reported all 10 TASK-baseline files allowed. Forbidden and out-of-scope lists are empty. The Fixer did not edit `pipeline-state.json`, shared config, Commons, dependencies, Proto, schema, auth, deployment, frontend, or root docs.

## 5. SPEC Comparison Result

PASS against FR-003. The runtime consumes exactly the seven existing settings and now preserves their independent meanings: `candidate_match=false` disables matching globally; `semantic=false` retains the future deterministic primary; `shadow=true` permits enhanced shadow execution when semantic primary is disabled. No external contract changed.

## 6. SDD Comparison Result

PASS against SDD sections 3, 5, 7, 8, and 13. The value-based policy and execution plan are immutable and available to TASK-004 through TASK-006 without temporary policy mutation. Defaults remain owned by `serviceconfig`.

## 7. Acceptance Comparison Result

PASS for implementation/checks:

- all seven settings have independent mapping tests;
- resume and match timeouts remain distinct and cancellation propagates;
- candidate-match overall, semantic-primary, deterministic-primary, and shadow combinations are explicit and testable;
- deterministic-only mode does not call the LLM runtime;
- semantic-disabled plus shadow-enabled mode allows job-requirement/evaluator structured calls for enhanced shadow execution;
- no behavior outside recruiting intelligence changed.

Independent review round 2 returned `verdict: 通过`; TASK completion is established.

## 8. Test Commands and Results

| Command | Result | Duration |
|---|---|---:|
| `GOWORK=off go test -count=1 ./internal/application/recruiting_intelligence ./internal/interfaces/grpc ./cmd/ai-agent-service` in AI Agent | PASS | 4.5s |
| `GOWORK=off go test ./...` in AI Agent | PASS | 0.38s |
| `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 338c74c96c20f6f89b2ee8c4166347cbdd997468` | PASS; routed impact reviewed | 1.12s |
| `git diff --name-only` inventory and `git diff --check` | PASS | 0.1s |
| `bash .spec/recruiting-intelligence-runtime-parity/scripts/check-task-scope.sh TASK-002` | PASS | 0.86s |
| `bash .spec/recruiting-intelligence-runtime-parity/scripts/agent-check.sh` | PASS, including full AI Agent and Commons AI tests | 5.65s |
| pending-review evidence validator | PASS | recorded in evidence |

One preliminary full-test invocation was issued from the repository root and returned the expected Go “no main module” setup error. It was immediately rerun from `smart-recruit-ai-agent-service` as required and passed; this was a command working-directory error, not a product/test failure.

## 9. Knowledge Impact

Result: `candidate_required`; detector exit code `0`; coverage gap `false`. The detector routed 18 active documents. `agent-runtime`, `ai-configuration-governance`, and `resume-intelligence` remain `CANDIDATE` for TASK-007 because TASK-002 cannot edit `.knowledge`; all other routed documents are `UNCHANGED` after source-reference review.

## 10. Risks

- TASK-004 through TASK-006 must consume `CandidateMatchExecutionPlan` so orchestration follows the declared primary/shadow intent; those workflows are intentionally out of TASK-002 scope.
- Structured runtime admission indicates that an enhanced primary or enhanced shadow is permitted; later orchestration must call it only for the selected execution, not for deterministic-only work.
- No candidate text, raw prompt/output, credentials, or sensitive evidence was recorded.

## 11. Reviewer History

### Round 1 — `/root/task002_reviewer_r1`

`verdict: 不通过`

- `R1-H1` High: `structured_runtime.go` treated `candidate_match_semantic=false` as disabling job requirement extraction and candidate evaluation. This broke the deterministic/legacy primary intent and prevented `semantic=false, shadow=true` from running enhanced shadow calls, including when `fallbacks=false`.
- `R1-M1` Medium: mapping tests did not independently prove explicit `candidate_match=false` total gating, field-by-field nil/default/false mapping, or semantic/shadow combinations.

Required fixes were limited to explicit overall/primary/shadow execution intent, correct structured-call gating, and the missing focused tests. Both findings are addressed in this repair.

## 12. Repair Summary

### Failed Check

Independent self-review round 1 failed with R1-H1 and R1-M1.

### Root Cause

The runtime used `candidate_match_semantic` as a capability gate instead of a primary-scorer selector, and the initial tests asserted mixed flag values rather than each field and execution combination independently.

### Files Changed

- `runtime_policy.go`
- `runtime_policy_test.go`
- `structured_runtime.go`
- `main_test.go`
- TASK-002 report/evidence

### Fix Summary

Added an immutable execution plan with disabled/deterministic/enhanced modes, made the structured LLM gate depend on overall match plus enhanced primary-or-shadow intent, and added field-by-field and combination tests. No unrelated refactor or functionality was added.

### Re-run Commands and Results

All focused tests, AI Agent full tests, scope check, Harness agent check, knowledge impact detection, whitespace check, and pending-review evidence validation pass.

### Remaining Risks

Independent round-2 review passed with no findings.

## 13. Whether the Next TASK Can Start

Yes. TASK-002 checks and evidence pass, and independent review round 2 returned `verdict: 通过`.

## 14. Final Independent Review

Reviewer `/root/task002_reviewer_r2` independently verified configuration mapping, execution intent, timeout/cancellation behavior, scope, tests, knowledge impact, evidence, privacy, and contract preservation. It reported no findings and ended with `verdict: 通过`.
