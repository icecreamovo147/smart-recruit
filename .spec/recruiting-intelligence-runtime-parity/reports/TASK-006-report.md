# TASK Report - TASK-006

## 1. TASK ID

TASK-006 — Restore Deterministic Aggregation and Persistence. Mode: `fix-check-failures`; failed review rounds 1 through 3 are preserved. The user authorized one exceptional fresh Fixer R3 and independent Reviewer R4 for R3-001. Repair R3 is complete, all required checks pass, review round 4 is pending, and TASK-007 remains blocked.

Baseline: `base_sha=98165e7cdac06c0aa2a0685ecf1ad7c00a3eb518`, `base_tree=dd2e6bb5a1ad698f88bf624615fb1403d8861106`.

## 2. Modified File List

- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/candidate_aggregation.go`
- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/candidate_aggregation_test.go`
- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/candidate_legacy.go`
- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/candidate_legacy_test.go`
- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/candidate_match.go`
- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/job_requirement.go`
- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/runtime_policy.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_candidate_match_test.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_persistence_test.go`
- `.spec/recruiting-intelligence-runtime-parity/reports/TASK-006-report.md`
- `.spec/recruiting-intelligence-runtime-parity/reports/TASK-006-evidence.json`

The root-owned `pipeline-state.json` differs from the TASK base tree because the coordinator opened TASK-006. The Developer did not edit it.

## 3. Change Summary by File

- `candidate_aggregation.go`: restores the `dev` enhanced aggregator with four fixed dimensions and the intentionally unallocated 0.05; exact missing/conflict, weak-evidence, knockout, cap, recommendation, must-have, signal, and Chinese-summary semantics; canonical requirement/result ordering; duplicate/missing/unknown-result rejection; and a score breakdown that records the profile, component results, dimensions, policy, input hash, scorer type, fallback state, and risk penalty.
- `candidate_aggregation_test.go`: covers exact dimension arithmetic, knockout score cap and forced recommendation, risk penalty and cap, every recommendation threshold, Chinese summary, deterministic JSON, canonical order, and rejection of model-created requirements.
- `candidate_legacy.go`: restores the dev legacy deterministic five-dimension scorer over the already-loaded privacy-bounded snapshot: skills 0.35, requirements 0.25, experience 0.20, education 0.10, and profile 0.10, with dev recommendation/summary/risk/evidence semantics and stable input hashing.
- `candidate_legacy_test.go`: proves the five exact dimensions and weights, deterministic repeated output, scorer provenance, positive scoring, and real persisted evidence IDs without a provider.
- `candidate_match.go`: exposes the already validated deterministic per-requirement path for deterministic-primary orchestration; no matcher or total-score model output is added.
- `job_requirement.go`: exposes the already validated heuristic job-requirement extractor for deterministic-primary orchestration; no API, table, Prompt schema, or migration is added.
- `runtime_policy.go`: adds candidate-match total/generation contexts with a deterministic 20% persistence reserve capped at two seconds, honoring a shorter parent deadline.
- `native_servers.go`: routes primary dependencies from the execution plan; lets deterministic primary generate and persist with no provider; switches eligible enhanced failures to the legacy scorer only when fallbacks are enabled; executes the opposite scorer as a read-only shadow and returns a privacy-safe internal delta; and applies one total deadline to source read, generation/shadow, and transaction save. A bound structured runtime never returns to the old aggregate-LLM path.
- `native_candidate_match_test.go`: additionally covers the restored legacy primary, nil-provider save, enhanced-to-legacy fallback, enhanced-primary/deterministic-shadow, deterministic-primary/enhanced-shadow, no shadow persistence, source-read timeout, generation timeout, parent cancellation, and persistence reserve.
- `native_persistence_test.go`: uses the real exported `persistence.NativeStore` against in-memory SQLite from an external test package. It proves version increment, latest demotion, `agent_run_id`, evidence mapping, historical non-backfill, and complete transaction rollback when evidence insertion fails.
- TASK report/evidence: records scope, contracts, checks, knowledge impact, privacy, risks, and pending independent review.

## 4. Scope Check Result

PASS. `check-task-scope.sh TASK-006` classified the Harness as current and identified eight pre-report baseline differences: the seven TASK-006 files above plus root-owned `pipeline-state.json`. Every difference is allowed, with no forbidden or out-of-scope file. No production persistence file, Proto, schema/migration, dependency, shared configuration, public API, auth, frontend, deployment, CI, root docs, or knowledge file was edited.

Preflight confirmed `NativeStore.SaveRecruitingCandidateMatchDraft` already performs version selection, old-latest demotion, evaluation creation, and every evidence insert inside one GORM transaction. TASK-006 reuses it without persistence production changes.

## 5. SPEC Comparison Result

PASS against FR-006 through FR-010 and AC-004 through AC-008. New evaluations use the internal structured requirement profile and per-requirement results, never accept a model-provided overall score or recommendation, aggregate deterministically, preserve real allowlisted evidence IDs, and call the existing transactional persistence adapter exactly once after successful generation. Fallback-disabled extraction/matcher/schema failure returns before save. Historical rows are read only by the existing version computation/latest switch and are not backfilled.

## 6. SDD Comparison Result

PASS against SDD sections 3, 4, 6.3 through 6.6, 7 through 9, 11, and 13. The restored formulas match `dev:logic-grpc-service/service/candidate_match_score_aggregator.go`: must-have 0.40, core skills 0.25, experience 0.20, growth 0.10; missing/conflict penalty 5, weak-evidence penalty 2, knockout additional penalty 10, total penalty cap 20; knockout score cap 40; and 85/70/50/30 recommendation thresholds with must-have ratios. The existing persistence transaction is reused without schema changes.

## 7. Acceptance Comparison Result

PASS:

- fixed normalized inputs produce stable requirement order, results, dimensions, risk signals, recommendation, Chinese summary, and JSON breakdown;
- a knockout missing/conflict caps the total at 40 and forces `strong_not_recommend`;
- risk penalties and recommendation thresholds exactly follow `dev`;
- model output has no overall-score/recommendation field in the per-requirement schema, and unknown/model-created requirements are rejected;
- orchestration retains persisted source table/ID pairs and caps evidence at 240 runes;
- `agent_run_id`, version increment, latest demotion, new latest, evidence rows, historical content, and full rollback are proven against the real `NativeStore`;
- fallback-disabled invalid main-chain output creates no draft or evaluation.

## 8. Test Commands and Results

- `GOWORK=off go test ./internal/application/recruiting_intelligence -run 'TestCandidateMatchAggregator' -count=10` in AI Agent — PASS, exit `0`.
- `GOWORK=off go test ./internal/interfaces/grpc -run 'TestEvaluateCandidateMatchStructuredPipeline|TestEvaluateCandidateMatchFallbackDisabled|TestEvaluateCandidateMatchDeterministicPrimary|TestNativeStoreCandidateMatchPersistence' -count=10` in AI Agent — PASS, exit `0`.
- `GOWORK=off go test ./...` in AI Agent — PASS, exit `0`.
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree dd2e6bb5a1ad698f88bf624615fb1403d8861106` — PASS, exit `0`; mechanical result `update_required`, classified below as TASK-007 candidates/unchanged verdicts.
- `bash .spec/recruiting-intelligence-runtime-parity/scripts/check-task-scope.sh TASK-006` — PASS, exit `0`.
- `bash .spec/recruiting-intelligence-runtime-parity/scripts/agent-check.sh` — PASS, exit `0`; includes feature/JSON/whitespace/gofmt, full AI Agent, and Commons AI tests.
- `git diff --check` — PASS, exit `0`.
- Repair round 1: `GOWORK=off go test ./internal/application/recruiting_intelligence -run 'TestScoreCandidateMatchLegacy|TestCandidateMatchAggregator' -count=10` — PASS, exit `0`.
- Repair round 1: `GOWORK=off go test ./internal/interfaces/grpc -run 'TestEvaluateCandidateMatch|TestCandidateMatchShadow|TestNativeStoreCandidateMatchPersistence' -count=10` — PASS, exit `0`.
- Repair round 1: `GOWORK=off go test ./...` in AI Agent — PASS, exit `0`.
- Repair round 1: impact detection, `git diff --name-only`, TASK-006 scope, `git diff --check`, and `agent-check.sh` — PASS, exit `0`.

Two early Go test invocations were accidentally run from repository root, which has no root `go.mod`; they exited non-zero with the expected “cannot find main module” message. They were invocation errors, not test failures. The same focused commands were immediately rerun from `smart-recruit-ai-agent-service` and passed, including `-count=10`.

## 9. Knowledge Impact

Result: `candidate_required`; detector exit code `0`; coverage gap `false`. `agent-runtime`, `ai-configuration-governance`, and `resume-intelligence` remain TASK-007 candidates because candidate matching is now fully wired through database Prompts, deterministic aggregation, and versioned persistence while current knowledge describes the older unavailable runtime. `resume-sensitive-data` is `UNCHANGED`: the active prohibition remains correct and the implementation reuses the TASK-005 redacted/bounded evidence index. Service ownership, public contracts, Proto, local development, Agent Skill/retrieval, MCP, memory/context, embedding, and system topology documents are `UNCHANGED`. Required source references were checked against current native runtime/store, generated Proto, and service-boundary files. No `.knowledge` file was edited.

## 10. Risks

- The existing transaction computes `MAX(version)+1` without redesigning concurrent writer serialization; the SPEC explicitly excludes that redesign.
- Structured production compositions use deterministic aggregation. The legacy aggregate adapter remains for compatibility only when structured Prompt/provider ports are not bound; TASK-007 integration validation should continue to prove production binding.
- Evidence uses the persisted resume-profile child rows currently present in `RecruitingMatchSource`. No scope-external persistence expansion was made to add raw resume or candidate-profile fields.

Privacy check: PASS. Tests and artifacts contain synthetic bounded fixtures only, and no real candidate data, complete Prompt, raw model response, credential, or secret.

## 11. Review Round 1 History

Independent reviewer `/root/task006_reviewer_r1` returned `verdict: 不通过`:

- R1-001: deterministic primary and enhanced fallback were not the dev legacy five-dimension deterministic scorer.
- R1-002: a global provider dependency blocked deterministic-primary evaluation from a loaded snapshot.
- R1-003: `plan.Shadow()` was not consumed, so neither opposite-scorer comparison ran.
- R1-004: the configured deadline covered generation only, with source read/save outside it and no persistence reserve.

This failed review is preserved; it is not overwritten by repair results.

## 12. Repair Summary

### Failed Check

Independent self-review round 1 failed R1-001 through R1-004.

### Root Cause

- The deterministic plan reused the enhanced requirement-profile aggregator instead of porting the dev legacy scorer.
- Native orchestration gated all generation on `provider != nil` rather than the plan's primary mode.
- The adapter ignored the plan's shadow mode entirely.
- Candidate match created a timeout only inside structured generation, after source loading, and saved with the caller context.

### Files Changed

- Added `candidate_legacy.go` and `candidate_legacy_test.go`.
- Updated `runtime_policy.go`, `native_servers.go`, and `native_candidate_match_test.go`.
- Updated this report and matching evidence only; no pipeline state edit was made by the Fixer.

### Fix Summary

- Restored dev-equivalent five-dimensional legacy deterministic scoring and mapped only bounded, allow-listed evidence with real source IDs.
- Made primary dependency selection plan-aware; pure deterministic primary requires only the generation store/snapshot.
- Enhanced primary switches to legacy only for eligible failures with `fallbacks=true`; fallback-disabled failures remain non-success/no-save.
- Both shadow directions execute the opposite scorer read-only, return only safe scalar deltas internally, never overwrite the primary draft, and never create an evaluation.
- Source read, generation/shadow, and save now share one feature-total deadline while generation ends early to reserve a bounded transaction window. Parent timeout/cancellation is checked before save.

### Re-run Commands and Results

Focused tests repeated ten times, AI Agent full tests, knowledge impact detection, scope validation, `agent-check.sh`, and `git diff --check` all pass with exit code `0`.

### Remaining Risks

- Shadow execution is deliberately best-effort: a shadow failure returns a safe unsuccessful comparison and cannot fail or replace the primary evaluation.
- The old aggregate-LLM adapter remains only for backward-compatible generic-only compositions where structured ports are absent; any bound structured runtime is prohibited by routing from reaching it.
- Existing concurrent version allocation remains unchanged and out of scope.

## 13. Follow-up Items

- Independent fresh round-2 Reviewer must verify R1-001 through R1-004 and produce the exact verdict line required by the Harness.
- TASK-007 owns final integration observability and knowledge updates.

## 14. Whether the Next TASK Could Start After Repair Round 1

No. At that point repair round 1 checks passed, but TASK-006 remained pending independent round-2 review. The later round-2 failure is preserved below.

## 15. Independent Review Round 2 History

Independent reviewer `/root/task006_reviewer_r2` returned `verdict: 不通过`:

- R2-001: the purported dev legacy scorer substituted the bounded evidence aggregate for dev's distinct candidate-profile, application-education, and raw-resume inputs; it therefore missed the exact six-field/`IsComplete` profile formula, nil-profile score 35, `resume_text_missing`, and other dev semantics.
- R2-002: a generic aggregate-LLM route remained reachable when enhanced structured dependencies were absent, allowing model-owned overall score/recommendation.
- R2-003: shadow coverage did not prove provider, Prompt, schema, and timeout failures were isolated from the primary result, single save, and evaluation count.

This failed review is preserved. R2-001 is the same underlying parity issue as R1-001. Preflight correctly hard-stopped until the user expressly authorized the required production source mapping; a second hard stop was raised when the stale scope-out gRPC test still asserted the forbidden aggregate model total.

## 16. Repair Summary — Round 2

### Failed Check

Independent self-review round 2 failed R2-001 through R2-003.

### Root Cause

- `RecruitingMatchSource` did not carry the candidate profile's six dev fields plus `IsComplete`, `candidate_profiles.education`, or the exact application resume's `parsed_text`.
- The missing-structured-dependency branch still called the old generic candidate aggregate prompt adapter.
- Shadow success was covered in both directions, but enhanced-shadow failure modes were not proven through the public orchestration/save boundary.

### Authorized Scope Expansions

- At `2026-07-15T09:35:14Z`, the user authorized TASK-006 to modify `internal/infrastructure/persistence/native_store.go` and `native_recruiting_generation_test.go` only to load and test real dev legacy inputs. Schema, Proto, dependencies, and public APIs remained forbidden.
- At `2026-07-15T09:52:15Z`, the user additionally authorized `internal/interfaces/grpc/native_recruiting_generation_test.go` only to replace its stale aggregate-LLM assertions with deterministic/no-generic-call semantics.

### Files Changed

- `candidate_legacy.go` / `candidate_legacy_test.go`: port the dev five-dimension scorer inputs and formulas exactly, including profile nil=35, six profile fields plus `IsComplete`, application education, raw-resume presence risk, experience cap, and golden fixtures.
- `native_store.go` / `native_recruiting_generation_test.go`: load full `candidate_profiles` data and the exact `applications.resume_id -> resumes.parsed_text`; prove complete and missing mappings against the real store.
- `native_servers.go`: carry the real internal source fields, build trusted evidence from them, remove every candidate aggregate DTO/prompt/generic completion adapter, and route missing enhanced dependencies only through policy-controlled deterministic fallback or failure.
- `native_candidate_match_test.go`: prove Prompt/provider/schema/timeout enhanced-shadow failure isolation, unchanged deterministic primary, exactly one save, no extra evaluation, and fallback-policy behavior without generic aggregate calls.
- `native_recruiting_generation_test.go`: replace the obsolete model-total assertion with deterministic scorer and zero generic provider-call assertions.
- This report and evidence: preserve failed review history, authorizations, truthful checks, and pending round-3 status. The Fixer did not edit `pipeline-state.json` or `task-scope.json`.

### Fix Summary

- Production source loading now reads the candidate profile fields and completeness directly from `candidate_profiles`, and reads parsed text by the application's exact `resume_id`; alternate resumes for the same user cannot leak into scoring.
- The deterministic scorer now matches dev field-by-field instead of treating the whole evidence index as a substitute for profile, education, or resume-presence inputs.
- Candidate matching contains no aggregate candidate DTO, aggregate prompt renderer, generic completion adapter, or route that accepts a model-provided overall score/recommendation.
- Enhanced dependency absence follows the immutable fallback policy: deterministic fallback when enabled, explicit failure/no-save when disabled.
- Enhanced shadow Prompt/provider/schema/timeout failures are best-effort and cannot replace or fail the deterministic primary; orchestration persists the primary once and creates no shadow evaluation. Existing bidirectional success coverage still proves enhanced-primary/deterministic-shadow and deterministic-primary/enhanced-shadow execution.

### Re-run Commands and Results

- `GOWORK=off go test ./internal/application/recruiting_intelligence -run 'TestScoreCandidateMatchLegacy|TestCandidateMatchAggregator' -count=10` — PASS.
- `GOWORK=off go test ./internal/interfaces/grpc -run 'TestEvaluateCandidateMatch|TestCandidateMatchShadow|TestNativeStoreCandidateMatchPersistence' -count=10` — PASS.
- `GOWORK=off go test ./internal/infrastructure/persistence -run 'TestNativeStore(GetRecruitingMatchSource|SaveRecruitingCandidateMatch)' -count=10` — PASS.
- `GOWORK=off go test ./...` in AI Agent — PASS.
- Knowledge impact, scope, `agent-check.sh`, evidence validation, and `git diff --check` results are recorded in the matching evidence.

### Remaining Risks

- Existing concurrent version allocation remains unchanged and out of scope.
- Shadow output remains intentionally non-persistent and best-effort; TASK-007 owns final privacy-safe observability.
- The mechanically routed knowledge documents remain TASK-007 candidates or unchanged; TASK-006 made no `.knowledge` edits.

## 17. Whether the Next TASK Can Start After Repair Round 2

No. Repair round 2 checks passed, but independent round 3 returned `verdict: 不通过`; `max_review_rounds=3` is exhausted.

## 18. Independent Review Round 3 — Hard Stop

Independent reviewer `/root/task006_reviewer_r3` returned `verdict: 不通过`:

- `R3-001` High: the legacy skills dimension extracts tokens from the full bounded evidence snippet (`Name + Evidence + Level + Category`) while dev uses only `ResumeSkill.Name`; evidence text can therefore create false skill matches.
- The legacy resume scoring text is reconstructed from an EvidenceIndex capped at 100 items and 240 runes per item, while dev scores all complete education, experience, project, and skill child fields. Tokens after the cap can change requirements, experience, overall score, and input hash.

All other R1/R2 findings, production source mapping, aggregate-LLM removal, shadow isolation, deadline reserve, transaction semantics, scope, privacy, and dynamic checks passed. This remaining finding is the same underlying dev-parity issue as R1-001/R2-001. Per the Harness maximum-review rule, TASK-006 is hard-stopped and TASK-007 cannot start without explicit user direction to reopen the failed TASK.

## 19. Authorized Review Extension

At `2026-07-15T11:30:24Z`, the user explicitly reopened TASK-006 and authorized exactly one fresh Fixer R3 plus one independent Reviewer R4 for R3-001. Code scope is unchanged. This is the third TASK-006 authorization, after the production source-loading and stale gRPC-test authorizations recorded above. `pipeline-state.json` remains root-owned and was not edited by this Fixer.

## 20. Repair Summary — R3

### Failed Check

Independent review round 3 failed R3-001: legacy skill matching consumed complete bounded skill snippets, and legacy scoring/hash input was reconstructed from the 100-unit/240-rune EvidenceIndex rather than dev's complete source rows.

### Root Cause

One bounded structure was incorrectly used for two distinct purposes. Dev uses only `ResumeSkill.Name` for the skills dimension and builds requirements/experience/InputHash text from every complete source field in a fixed order. The bounded/redacted EvidenceIndex is only appropriate for trusted matching evidence and persistence.

### Files Changed

- `candidate_legacy.go`: accepts complete in-memory `ResumeText` and `SkillNames`; skills tokenization uses only names, while requirements, experience, and InputHash use the full text. Evidence persistence still consumes the bounded/redacted index.
- `candidate_legacy_test.go`: adds dev-difference goldens for Evidence not promoting a skill, tokens beyond 240 runes, exact dimensions/overall/recommendation/risks/summary/hash, and deterministic repetition.
- `native_servers.go`: constructs complete legacy scoring text in exact dev order from parsed text; resume-profile header; candidate profile; all education, experience, project, and skill child fields. Skill names are separately collected and sorted like dev. Raw text remains request-local and is never logged or persisted.
- `native_candidate_match_test.go`: proves native source order, the token in record 101, the token after rune 240, name-only skill matching, exact complete-input hash, and independently bounded persisted evidence.
- This report and matching evidence record Repair R3 truthfully. The Fixer did not edit `pipeline-state.json` or `task-scope.json`.

### Fix Summary

- `ResumeSkill{Name: "Go", Evidence: "Docker"}` can satisfy a Docker requirements token through dev's complete resume text but cannot satisfy the Docker skills dimension.
- Complete, untruncated source rows now determine requirements, experience, overall score, recommendation, risks, summary, and InputHash exactly as in dev; the EvidenceIndex's 100-item and 240-rune bounds cannot alter those results.
- Persisted evidence remains allow-listed, contact-redacted, capped at 100 units, and capped at 240 runes per snippet. Complete raw scoring text is not added to logs, reports, evidence JSON, or database evidence rows.

### Re-run Commands and Results

- `GOWORK=off go test ./internal/application/recruiting_intelligence -run 'TestScoreCandidateMatchLegacy|TestCandidateMatchAggregator' -count=10` — PASS.
- `GOWORK=off go test ./internal/interfaces/grpc -run 'TestEvaluateCandidateMatch|TestCandidateMatchShadow|TestDeterministicCandidateMatchUsesCompleteDevScoringInput|TestNativeStoreCandidateMatchPersistence' -count=10` — PASS.
- `GOWORK=off go test ./internal/infrastructure/persistence -run 'TestNativeStore(GetRecruitingMatchSource|SaveRecruitingCandidateMatch)' -count=10` — PASS.
- `GOWORK=off go test ./...` in AI Agent — PASS.
- Knowledge impact detection — PASS; existing candidate/unchanged verdicts remain accurate and no `.knowledge` file was edited.
- Final scope, agent-check, evidence validator, and diff checks are recorded in machine-readable evidence.

### Remaining Risks

- Existing concurrent version allocation remains unchanged and out of scope.
- Complete scoring text is intentionally request-local; future diagnostics must preserve the current prohibition on logging or artifacting it.
- TASK-007 knowledge candidates remain deferred and cannot start before independent review round 4 passes.

## 21. Review Round 4 Status

Fresh replacement Reviewer R4 `/root/task006_reviewer_r4_retry` completed a read-only review and returned exact `verdict: 通过`. The original R4 execution context produced no verdict and modified no files, so it was replaced without adding another review round.

- R3-001 is closed: skill evidence cannot promote a skill, full dev-ordered scoring text is unbounded in request memory, and persisted evidence remains independently bounded/redacted.
- All R1/R2 findings, source mapping, aggregate-LLM removal, routing, shadow failure isolation, deadlines, persistence semantics, scope, privacy, and knowledge impact were reverified.
- Three focused suites at `-count=10`, AI Agent full tests, scope, `agent-check.sh`, evidence validation, knowledge impact, and diff checks passed.

## 22. Whether the Next TASK Can Start After Repair R3

Yes. Repair R3 and all required checks passed, and independent review round 4 returned exact `verdict: 通过`. `next_task_ready=true` and TASK-007 may start.
