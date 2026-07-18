# TASK Report - TASK-005

## 1. TASK ID

TASK-005 — Restore Per-Requirement Candidate Matching (completed after independent review round 3).

## 2. Modified File List

- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/candidate_match.go`
- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/candidate_match_test.go`
- `.spec/recruiting-intelligence-runtime-parity/reports/TASK-005-report.md`
- `.spec/recruiting-intelligence-runtime-parity/reports/TASK-005-evidence.json`

The root-owned `pipeline-state.json` differs from base tree `0f73c2d91a874b2981497d8623edbc8630010bd4` because the coordinator opened TASK-005. The Developer did not edit it.

## 3. Change Summary by File

- `candidate_match.go`: adds a transport-neutral evidence source contract using persisted non-zero IDs; an immutable, allowlisted, deterministic evidence index capped at 100 units and 240 runes per snippet; centralized email and bounded phone redaction/detection for mainland mobile, separated fixed-line, and representative international formats; exclusion of redaction markers from deterministic search; recursive duplicate-key rejection; matcher-specific Simplified Chinese natural-language validation that permits only an explicit normalized technical vocabulary and never trusts case shape or evidence content as vocabulary authority; exact per-requirement matcher JSON decoding; trusted provenance validation; deterministic-first matching; one active `candidate_match_evaluator/system` call per unresolved requirement; and policy-correct fallback provenance. It does not aggregate or select an overall score.
- `candidate_match_test.go`: covers all six source-table allowlist values, stable real IDs, zero-ID rejection, deterministic ordering/deduplication, immutable defensive copies, global/snippet bounds, contact-format coverage and numeric false positives, index/User-message privacy, redaction-marker alias counterexamples, deterministic zero-call behavior, one matcher call per unresolved requirement, exact System/User behavior, recursive duplicate JSON keys, strict enums/ranges/provenance, lowercase/all-uppercase/camelCase English explanation rejection, controlled `Node.js`/`Go`/`gRPC`/`Kubernetes` technical names, fallback enabled/disabled semantics, deterministic result retention, and semantic/shadow intent.
- `TASK-005-report.md`: records scope, contract comparison, validation, knowledge impact, privacy, risks, repair history, and the final independent verdict.
- `TASK-005-evidence.json`: records the same baseline, changed files, real command results, review history, final verdict, and knowledge-impact evidence in machine-readable form.

## 4. Scope Check Result

Passed against TASK base tree `0f73c2d91a874b2981497d8623edbc8630010bd4`. The scope checker identified only the root-owned pipeline state, the two TASK-005 implementation files, and TASK-005 report/evidence as allowed. No forbidden or out-of-scope Developer edit exists. No gRPC/native adapter, persistence, public API, Proto, schema/migration, dependency, Commons, configuration, frontend, root-doc, or knowledge file changed.

## 5. SPEC Comparison Result

Passed for FR-002, FR-003, FR-007, FR-008, FR-009's “model shall not decide final score” boundary, and the privacy/error portions of FR-009/FR-010. Every enhanced matcher call uses TASK-001 `Runtime.Complete` with exact agent type `candidate_match_evaluator`, which reloads the active System prompt. Deterministic evidence is evaluated first and suppresses the model call. Model evidence is accepted only when its allowlisted table, real source ID, and exact bounded snippet already exist in the index. Individual failures retain deterministic judgments only when fallback policy allows the enhanced pipeline to continue; fallback-disabled failure returns a non-success with a complete deterministic result set for downstream non-persistence handling.

TASK-006 aggregation, recommendation, score breakdown, and persistence were not implemented.

## 6. SDD Comparison Result

Passed against SDD sections 3, 4, 6.4, 6.5, 7, 9, 10, 11, and 13. The implementation is internal to the AI Agent application package, consumes the immutable runtime policy, uses one bounded request deadline, sends only indexed/redacted evidence, preserves stable source provenance, and returns safe classification errors without prompt, candidate text, or raw model output. The index exposes defensive copies so callers cannot mutate trusted provenance after construction.

## 7. Acceptance Comparison Result

Passed all TASK-005 acceptance criteria:

- only `resume_skills`, `resume_experiences`, `resume_projects`, `resume_educations`, `resumes`, and `candidate_profiles` enter the index;
- zero/synthetic IDs and empty evidence are excluded, duplicates are resolved deterministically by table/ID/snippet ordering, and the index is capped at 100 units;
- every snippet is contact-redacted for email, mainland mobile, separated fixed-line, and representative international formats, whitespace-normalized, capped at 240 runes, and protected behind a defensive-copy API; ordinary years and numeric identifiers are not broadly erased;
- `[PHONE]` and `[EMAIL]` remain visible privacy markers but are excluded from normalized/searchable terms and deterministic alias matching;
- deterministic direct evidence produces a stable result and exactly zero matcher calls;
- semantic primary or enhanced shadow intent invokes the matcher exactly once per unresolved requirement, while deterministic-only intent invokes none;
- the active candidate matcher prompt is loaded with role `system` on each call and scoped requirement/index JSON is the User message;
- output requires exactly `status`, `score`, `confidence`, `risk`, and `evidence`, while each evidence object requires exactly the migration fields;
- invalid enum/range, missing/unknown field, duplicate key at any object level, trailing JSON, unknown table/ID, duplicate reference, fabricated snippet, excessive evidence, missing required evidence, mixed-English risk/reason, or contact leakage is rejected;
- model failure retains every deterministic result; fallback enabled continues with the deterministic missing judgment and sets fallback provenance, while fallback disabled aborts before downstream persistence with `FallbackUsed=false`;
- no overall score, recommendation, or aggregation output exists in TASK-005.

## 8. Test Commands and Results

- Repair R1 `GOWORK=off go test -count=10 ./internal/application/recruiting_intelligence` — passed, exit `0`, 711 ms package time / 3.3 s command wall time.
- Repair R1 `GOWORK=off go test ./...` in `smart-recruit-ai-agent-service` — passed, exit `0`, 4.68 s wall time.
- Repair R1 `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 0f73c2d91a874b2981497d8623edbc8630010bd4` — passed, exit `0`, 1.28 s; detector result `update_required`, recorded below as TASK-007 candidates/unchanged verdicts because TASK-005 cannot edit knowledge.
- Repair R2 `GOWORK=off go test -count=10 ./internal/application/recruiting_intelligence` — passed, exit `0`, 3.4 s wall time.
- Repair R2 `GOWORK=off go test ./...` in `smart-recruit-ai-agent-service` — passed, exit `0`, 5.25 s wall time.
- Repair R2 `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 0f73c2d91a874b2981497d8623edbc8630010bd4` — passed, exit `0`, 2.49 s; detector result remains `update_required` and the existing TASK-007 candidate verdicts remain accurate.
- Repair R2 `git diff --name-only` inventory, `check-task-scope.sh TASK-005`, and `git diff --check` — passed, exit `0`, 1.4 s.
- Repair R2 `bash .spec/recruiting-intelligence-runtime-parity/scripts/agent-check.sh` — passed, exit `0`, 1.6 s, including feature/JSON/whitespace/gofmt, AI Agent, and Commons AI checks.
- Repair R2 pending-review evidence validator, final `git diff --check`, and TASK scope rerun — passed, exit `0`, 1.3 s.
- `git diff --name-only` inventory plus `bash .spec/recruiting-intelligence-runtime-parity/scripts/check-task-scope.sh TASK-005` — passed.
- `bash .spec/recruiting-intelligence-runtime-parity/scripts/agent-check.sh` — passed, including feature validation, JSON parsing, whitespace, gofmt, full AI Agent tests, and Commons AI tests.
- Pending-review evidence validator and final `git diff --check`/scope rerun — passed.

During initial implementation, one timing wrapper used the GNU-only `date +%s%3N` form on macOS. All underlying Go packages passed, but the wrapper then exited on timestamp arithmetic. That failed wrapper remains excluded from successful evidence; the repair commands above use portable timing and exited successfully.

## 9. Knowledge Impact

Result: `candidate_required`; detector exit code `0`; coverage gap `false`. `agent-runtime`, `ai-configuration-governance`, `resume-intelligence`, and `resume-sensitive-data` need TASK-007 candidates because active DB-prompt per-requirement evaluation and the bounded/redacted provenance index now exist internally while current knowledge describes the older runtime. Mechanically routed Agent Skill, retrieval, embedding, local-development, MCP, memory/context, semantic-retrieval, service-boundary, and system-overview documents are `UNCHANGED`. Their source references were checked against current runtime/native/policy files; this TASK changes none of those concerns. No `.knowledge` file was edited.

## 10. Risks

- TASK-005 supplies an internal component. TASK-006 must map the current persisted recruiting snapshot rows into `CandidateEvidenceSource`, invoke extraction/evaluation, aggregate deterministically, and persist transactionally before user-facing candidate matching gains this path.
- Deterministic matching is deliberately conservative: it requires a direct bounded alias/label occurrence and delegates only unresolved judgments to the active evaluator.
- Phone redaction is deliberately bounded to validated mainland mobile, separated mainland fixed-line, and representative country-code formats. Unseparated ambiguous numeric strings are preserved to avoid erasing ordinary identifiers. Candidate profile contact fields are structurally absent from the input contract.
- Matcher Simplified-Chinese validation uses only a conservative explicit technical vocabulary. Unknown uppercase, title-case, lowercase, or camelCase English tokens are rejected; new legitimate product names require a reviewed vocabulary addition.

Privacy check passed: reports/evidence contain no candidate PII, raw real resume/job data, complete DB prompt, raw model response, credential, or secret. Tests use synthetic bounded fixtures.

## 11. Reviewer History

### Round 1 — `/root/task005_reviewer_r1`

`verdict: 不通过`

- `R1-H1` High: contact redaction/detection did not cover mainland fixed lines or representative international formats, and tests did not prove raw values were absent from both the index and matcher User message.
- `R1-M1` Medium: fallback-disabled matcher failure incorrectly set `FallbackUsed=true` even though deterministic fallback was neither allowed nor adopted.
- `R1-M2` Medium: `[PHONE]` and `[EMAIL]` entered searchable terms/raw deterministic matching, so phone/email aliases could incorrectly skip the LLM.
- `R1-M3` Medium: matcher risk/reason reused a Han-presence display check and accepted mixed English explanation such as `风险 candidate lacks required experience` despite the active database prompt.
- `R1-M4` Medium: standard map decoding silently accepted duplicate JSON keys at top-level and nested evidence-object levels.

### Round 2 — `/root/task005_reviewer_r2`

`verdict: 不通过`

- `R2-M1` Medium: unknown all-uppercase tokens of at most ten characters and unknown internal-uppercase tokens were accepted as technical names, allowing `风险 CANDIDATE LACKS REQUIRED EXPERIENCE` to bypass the Chinese explanation requirement, while legitimate `Node.js` was rejected because `js` was absent from the controlled vocabulary.

Both failed reviews are retained. The Fixer did not self-approve the repair.

### Round 3 — `/root/task005_reviewer_r3`

`verdict: 通过`

- Independently verified R1-H1, R1-M1 through R1-M4, and R2-M1 are closed.
- Re-ran focused tests, full AI Agent tests, scope, Harness, evidence, knowledge-impact, and diff checks.
- Found no remaining Critical, High, Medium, or Low findings and no TASK-006 scope leakage.

## 12. Repair Summary — R1

### Failed Check

Independent self-review round 1 failed with R1-H1 and R1-M1 through R1-M4.

### Root Cause

Contact handling used one mainland-mobile regex; redaction markers shared the raw deterministic search path; fallback provenance was mutated before checking policy; generic display validation only required some Han text; and `encoding/json` map decoding overwrote duplicate keys.

### Files Changed

- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/candidate_match.go`
- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/candidate_match_test.go`
- TASK-005 report/evidence

### Fix Summary

- Centralize phone match ranges for both redaction and leak detection, add bounded mainland mobile/fixed-line and representative international formats, and preserve normal years/numeric identifiers.
- Strip `[PHONE]`/`[EMAIL]` only from searchable content and normalized terms while retaining markers in the privacy-safe snippet/User payload.
- Set `FallbackUsed=true` only after fallback policy permits retaining the deterministic missing judgment.
- Add matcher-only Simplified Chinese natural-language validation, rejecting English prose while allowing explicit technical names, bounded acronyms, and camel-case identifiers.
- Recursively walk JSON tokens and reject duplicate keys in every object before exact schema decoding.

### Re-run Commands and Results

Focused `-count=10`, full AI Agent tests, knowledge impact detection, TASK scope, Harness `agent-check`, pending-review evidence validation, and `git diff --check` pass.

### Remaining Risks

The bounded phone rules intentionally prefer low false positives over speculative detection. The round-1 language repair was superseded by the stricter round-2 controlled-vocabulary repair; independent review round 3 passed.

## 13. Repair Summary — R2

### Failed Check

Independent self-review round 2 failed with R2-M1.

### Root Cause

The matcher language validator treated all-uppercase length and internal-uppercase shape as evidence that an unknown English token was technical. Separately, the controlled list omitted the `js` token needed when `Node.js` is lexically split.

### Files Changed

- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/candidate_match.go`
- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/candidate_match_test.go`
- TASK-005 report/evidence

### Fix Summary

- Remove all case-shape exemptions; every English token in risk/reason must normalize to an explicitly reviewed technical term.
- Add only `js` to support `Node.js`; do not derive trusted terms from candidate evidence.
- Add regressions for unknown all-uppercase and camelCase English explanation clauses plus positive `Node.js`, `Go`, `gRPC`, and `Kubernetes` contexts.

### Re-run Commands and Results

Focused `-count=10`, full AI Agent tests, knowledge impact detection, TASK scope, Harness `agent-check`, pending-review evidence validation, and diff checks pass.

### Remaining Risks

The explicit vocabulary intentionally rejects unknown product names until reviewed. Independent review round 3 passed.

## 14. Whether the Next TASK Can Start

Yes. Independent TASK-005 self-review round 3 returned exact `verdict: 通过`, and required evidence remains valid; TASK-006 can start after the root coordinator records completion.
