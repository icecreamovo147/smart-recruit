# TASK Report - TASK-004

## 1. TASK ID

TASK-004 — Restore Job Requirement Extraction (completed after independent review round 3).

## 2. Modified File List

- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/job_requirement.go`
- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/job_requirement_test.go`
- `.spec/recruiting-intelligence-runtime-parity/reports/TASK-004-report.md`
- `.spec/recruiting-intelligence-runtime-parity/reports/TASK-004-evidence.json`

The root-owned `pipeline-state.json` differs from the TASK base tree because the root coordinator opened TASK-004. The Developer did not edit it.

## 3. Change Summary by File

- `job_requirement.go`: adds the internal job source/profile/item/result contracts; strict single-object JSON decoding with exact DB-prompt key presence and types; original-value validation for machine IDs/categories/priorities; bounded domain validation; conservative DB-prompt-aligned Simplified-Chinese display validation that rejects controlled Traditional/Japanese markers while permitting embedded technical names; boundary-safe deterministic minor weight-drift normalization with post-validation; schema-wrapped object/null errors; active DB System-prompt execution through the TASK-001 runtime; policy-controlled safe fallback provenance; and deterministic heuristic extraction suitable for TASK-005/006 consumers.
- `job_requirement_test.go`: covers active `job_requirement_extractor/system` role/message use, missing/wrong keys and types, unknown fields, top-level/nested null schema classification, single-JSON enforcement, invalid enums/IDs/aliases/counts/weights, duplicate IDs, surrounding-whitespace rejection for all machine fields under both fallback modes, controlled Traditional/Japanese display rejection under both fallback modes, Simplified Chinese display fields with Go/gRPC/Kubernetes technical names, tiny/material/boundary-invalid drift, all eligible fallback classes, overall/semantic/shadow gates, stable heuristic IDs/categories/priorities/weights/aliases, and defensive stable traversal.
- `TASK-004-report.md`: records contract comparison, checks, risks, knowledge impact, privacy, repair history, and the final independent verdict.
- `TASK-004-evidence.json`: records machine-readable baseline, scope, commands, knowledge impact, privacy, repair history, and the final independent verdict.

## 4. Scope Check Result

Passed against base tree `4ab11f06d69ea2c3a06912ea4ef95fd8c5ee992c`. No forbidden or out-of-scope files were changed by the Developer. No endpoint, native server, persistence adapter, Proto, migration, table, shared package, dependency, or configuration file changed.

## 5. SPEC Comparison Result

Passed for FR-002, FR-003, FR-006, FR-009 fallback semantics, and the internal-only portions of FR-010. The component reloads the exact active `job_requirement_extractor/system` prompt for every primary request through `Runtime.Complete`; accepts a scoped User message containing title, department, location, description, and requirements; strictly validates `job-requirement-profile-v1`; preserves overall candidate-match and semantic/shadow intent; and exposes only internal data structures. No later matching or aggregation behavior was implemented.

## 6. SDD Comparison Result

Passed against SDD sections 3, 4, 6.3, 7, 9, 10, and 13. The implementation reuses the structured runtime and immutable policy, does not cache prompts, does not mutate request policy, classifies fallback-eligible failures, and does not leak prompt/job/model bodies through error strings or fallback provenance.

## 7. Acceptance Comparison Result

Passed all TASK-004 acceptance criteria:

- requirement lists are non-empty and capped at 20;
- all DB schema keys are required at both object levels and retain exact JSON types;
- IDs are bounded lowercase kebab-case and unique, and are validated from the unmodified model value;
- categories and priorities use the exact migration enums and are validated from the unmodified model value;
- labels/descriptions are non-empty and bounded;
- weights are finite, positive, at most one, and sum to one;
- only total drift at or below `1e-6` is corrected, deterministically on the final item, and the adjusted item plus final sum are validated again;
- `label` and `description` conservatively require Han display prose and reject controlled Traditional/Japanese markers while allowing technical names such as Go, gRPC, and Kubernetes;
- knockout must be a JSON boolean and aliases must be a bounded array of unique, non-empty strings;
- prompt, provider, timeout, adapter-empty, content-empty, JSON, and schema failures use the heuristic only when fallbacks are enabled;
- policy-disabled execution never falls back;
- the heuristic produces stable IDs/categories/priorities/weights/aliases from all scoped job fields and an explicit general requirement when source detail is sparse.

The DB migration text simultaneously says “at least 2” and says sparse jobs should emit one general requirement. SPEC and acceptance consistently require a bounded non-empty list, so TASK-004 enforces one through twenty and does not invent a contradictory minimum of two.

## 8. Test Commands and Results

- `GOWORK=off go test -count=10 ./internal/application/recruiting_intelligence -run 'TestJobRequirement|TestStableJobRequirements'` after repair R1 — passed.
- `GOWORK=off go test ./...` in `smart-recruit-ai-agent-service` after repair R1 — passed.
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 4ab11f06d69ea2c3a06912ea4ef95fd8c5ee992c` — passed; result `update_required`, handled as deferred candidates/unchanged verdicts under TASK-004 scope.
- `git diff --name-only` inventory and `bash .spec/recruiting-intelligence-runtime-parity/scripts/check-task-scope.sh TASK-004` after repair R1 — passed; scope script identified only the root-owned state delta and TASK-004 implementation/report files.
- `bash .spec/recruiting-intelligence-runtime-parity/scripts/agent-check.sh` after repair R1 — passed, including feature validation, JSON parsing, whitespace, gofmt, full AI Agent tests, and Commons AI tests.
- `node .agents/skills/spec-harness/scripts/validate-evidence.mjs --file .spec/recruiting-intelligence-runtime-parity/reports/TASK-004-evidence.json --allow-pending-review --require-knowledge-impact` — passed.
- `GOWORK=off go test -count=10 ./internal/application/recruiting_intelligence -run 'TestJobRequirement|TestStableJobRequirements'` after repair R2 — passed.
- `GOWORK=off go test ./...` in `smart-recruit-ai-agent-service` after repair R2 — passed.

## 9. Knowledge Impact

Result: `candidate_required`; detector exit code `0`; coverage gap `false`. `agent-runtime`, `ai-configuration-governance`, and `resume-intelligence` require a TASK-007 documentation candidate because DB-backed structured job requirement extraction now exists internally while the current knowledge describes only the earlier runtime state. The other mechanically routed documents are `UNCHANGED`: this TASK changes no service ownership, public API, local startup, retrieval, memory, Agent Skill, MCP, embedding, or sensitive-data policy. No `.knowledge` file was edited because TASK-004 forbids it.

## 10. Risks

- The heuristic intentionally recognizes a bounded vocabulary plus experience/education markers; unmatched jobs safely receive one general requirement and are visibly marked as heuristic fallback.
- The `1e-6` drift tolerance accepts representational floating-point noise but rejects model arithmetic mistakes such as a `0.99` total.
- This TASK provides an internal component only. TASK-005/006 must wire it into evaluation and persistence orchestration before user-facing candidate matching gains the behavior.

Privacy check passed: no candidate PII, raw real job data, full DB prompt, model output, credential, or secret was written to logs, reports, or evidence. Test fixtures are synthetic and bounded.

## 11. Reviewer History

### Round 1 — `/root/task004_reviewer_r1`

`verdict: 不通过`

- `R1-H1` High: accepted weights `[1.0, 5e-7]` were within total drift tolerance, but final-item residual normalization produced a zero/non-positive weight without a boundary guard or post-normalization validation.
- `R1-M1` Medium: active DB prompt requirements that `label` and `description` use Simplified Chinese display prose, with technical names allowed, were not validated or tested under both fallback modes.
- `R1-M2` Medium: top-level `null` and `requirements:[null]` are legal JSON with invalid schema, but the `expected JSON object` cause did not wrap `ErrJobRequirementSchema`.

This review history is retained. The Fixer does not self-approve the repairs.

### Round 2 — `/root/task004_reviewer_r2`

`verdict: 不通过`

- `R2-H1` High: `id`, `category`, and `priority` were trimmed before validation, so schema-invalid machine values with surrounding whitespace were silently accepted.
- `R2-M1` Medium: the Han-only display check accepted Traditional Chinese and Japanese text even though the active database Prompt requires Simplified Chinese display prose.

Rounds 1 and 2 remain recorded as failed. Repair R2 is not a passing review verdict.

### Round 3 — `/root/task004_reviewer_r3`

`verdict: 通过`

- Independently verified R1-H1, R1-M1/R2-M1, R1-M2, and R2-H1 are closed.
- Re-ran focused tests, full AI Agent tests, scope, Harness, evidence, and knowledge-impact checks.
- Found no remaining Critical, High, Medium, or Low findings.

## 12. Repair Summary — R1

### Failed Check

Independent self-review round 1 failed with R1-H1, R1-M1, and R1-M2.

### Root Cause

Minor drift was adjusted only after the initial validation pass; display fields were checked only for non-empty/bounded text rather than the active DB prompt's Chinese-display contract; and JSON `null` decoded into a nil object map whose custom error lacked the schema sentinel.

### Files Changed

- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/job_requirement.go`
- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/job_requirement_test.go`
- TASK-004 report/evidence

### Fix Summary

- Guard the final residual before mutation, then revalidate every normalized weight and the normalized sum.
- Require Chinese display prose in `label` and `description` while accepting embedded technology terms such as Go, gRPC, and Kubernetes; verify English display prose uses schema fallback only when enabled.
- Wrap expected-object failures with `ErrJobRequirementSchema`, preserving top-level and nested null as schema failures.

### Re-run Commands and Results

Focused `-count=10`, full AI Agent tests, knowledge impact detection, TASK scope, Harness `agent-check`, pending-review evidence validation, and `git diff --check` pass.

### Remaining Risks

The standard library can reliably require Chinese-script display prose without maintaining a brittle technology whitelist; it does not provide a complete Traditional-to-Simplified lexicon. The validation therefore rejects English-only natural-language display fields and permits arbitrary technical names only when embedded in Chinese prose, matching the active prompt examples.

Independent review round 2 is required.

## 13. Repair Summary — R2

### Failed Check

Independent self-review round 2 failed with R2-H1 and R2-M1.

### Root Cause

Machine fields were normalized before their schema validation, changing invalid model output into accepted enum/ID values. Display validation used the Unicode Han script as a language proxy, although Han includes Simplified Chinese, Traditional Chinese, and Japanese kanji.

### Files Changed

- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/job_requirement.go`
- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/job_requirement_test.go`
- TASK-004 report/evidence

### Fix Summary

- Validate `id`, `category`, and `priority` exactly as decoded; no trim or mutation occurs before ID-pattern, duplicate, or enum validation.
- Replace the Han-only predicate with a documented conservative rule: require Han text and reject controlled Traditional variants, Japanese kana, and common Japanese-only character variants. This is intentionally not presented as complete language identification.
- Add fallback-disabled schema-error and fallback-enabled heuristic tests for whitespace-wrapped IDs/categories/priorities, the required pure-Traditional examples, Japanese kanji, and Japanese kana; keep the Simplified Chinese Go/gRPC/Kubernetes success case.

### Re-run Commands and Results

Focused `-count=10`, full AI Agent tests, knowledge impact detection, TASK scope, and Harness `agent-check` pass. Pending-review evidence validation and `git diff --check` also pass before handoff.

### Remaining Risks

Without a language-model or conversion dependency, no static validator can perfectly classify every CJK sentence. The controlled marker policy deliberately rejects known non-zh-CN forms and requires Han display prose; unlisted ambiguous all-common-Han phrases may remain indistinguishable. This limitation is explicit and covered by exact migration-prompt regression examples.

Independent review round 3 passed.

## 14. Follow-up Items

- TASK-007 should update the three deferred knowledge candidates after the complete pipeline stabilizes.

## 15. Whether the Next TASK Can Start

Yes. Repair R2 and all checks passed, and independent review round 3 returned exact verdict `verdict: 通过`; TASK-005 can start.
