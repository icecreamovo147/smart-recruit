# TASK Report - TASK-007

## 1. TASK ID

TASK-007 — Integration, Observability, and Knowledge (`self-review`; fresh independent Reviewer R5 returned `verdict: 通过`; TASK completed).

Harness classification: `current`. Reliable baseline: `base_sha=98165e7cdac06c0aa2a0685ecf1ad7c00a3eb518`, `base_tree=a3c78ee11f198247c20cf107b3790fd7c5ce61ce`. Human confirmation is not required. The Developer did not modify root-owned `pipeline-state.json`.

## 2. Modified File List

- `.knowledge/architecture/agent-runtime.md`
- `.knowledge/architecture/service-boundaries.md`
- `.knowledge/domains/ai-configuration-governance.md`
- `.knowledge/domains/resume-intelligence.md`
- `.knowledge/runbooks/debug-resume-intelligence.md`
- `.knowledge/pitfalls/resume-sensitive-data.md`
- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/structured_runtime.go`
- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/structured_runtime_observability_test.go`
- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/resume_profile.go`
- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/job_requirement.go`
- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/candidate_match.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/recruiting_observability.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/recruiting_observability_test.go`
- `.spec/recruiting-intelligence-runtime-parity/reports/compatibility-and-risk.md`
- `.spec/recruiting-intelligence-runtime-parity/reports/TASK-007-report.md`
- `.spec/recruiting-intelligence-runtime-parity/reports/TASK-007-evidence.json`
- `.spec/recruiting-intelligence-runtime-parity/pipeline-state.json` differs from the base only because the root coordinator opened TASK-007; it was not edited by the Developer.

## 3. Change Summary by File

- `structured_runtime.go`: expands the immutable observer allow-list with request/internal resource IDs, operation/stage terminality, fixed categories, parser/scorer versions, and bounded counts. Every non-empty request ID now becomes a process-random-key, domain-separated HMAC-SHA256 correlation token; an unexported normalized marker makes Runtime/native/Zap boundaries idempotent without trusting a caller-supplied token shape. Prompt/model identities use bounded non-reversible correlation aligned with their 256/128-byte contracts, invalid UTF-8/control text fails closed, and only exact internal parser/scorer versions remain readable.
- `resume_profile.go`, `job_requirement.go`, `candidate_match.go`: emit safe primary/fallback/error outcomes without logging source or response bodies.
- `recruiting_observability.go`: defensively normalizes a directly supplied event, then maps only the expanded SPEC §8 privacy-safe fields to one fixed Zap message.
- `native_servers.go`: installs one deferred finalizer at the start of both public write-or-read methods, before validation. Nil/invalid input, missing dependencies, application and AI authorization, not-found/error, capability and structured-path decisions, compatibility reads, generation/aggregation/timeout/fallback/persistence, and success all produce exactly one terminal event. Typed authorization/configuration errors now survive resume access resolution, so missing/nil application-owner dependencies use the fixed `configuration_failure` terminal category without response-message classification. Persistence failure can no longer follow a misleading overall-success event.
- observability tests: prove all three exact agent types load role `system`, all component stages are non-terminal, the complete generation matrix and exhaustive real early returns emit exactly one terminal, and persistence success precedes overall success. Privacy tests cover safe-alphabet phone/body/API-key/JWT/UUID/ULID/normal IDs, Unicode/control/overlong input, same-process stability, distinct-input separation, repeated-normalization idempotence, Prompt 256-byte/model 128-byte UTF-8 and punctuation, and absence of every raw marker from observations/logs.
- six knowledge files: retain the bounded TASK-007 update; the three R1-identified documents now cite the exact runtime, extractor/evaluator/aggregator, observer, persistence, and regression-test source refs. The debug runbook includes executable metadata-only SQL, focused commands/expectations, port/provider skip decisions, and a synthetic-only live-smoke procedure.
- `compatibility-and-risk.md`: records `dev`/current parity, live-smoke limitation, compatibility invariants, residual risks, and operational rollback.
- TASK report/evidence: records truthful commands, scope, privacy, knowledge impact, live skip, and the independent passing review.

## 4. Scope Check Result

PASS against `a3c78ee11f198247c20cf107b3790fd7c5ce61ce`. All Developer changes match TASK-007 allowed files; forbidden and out-of-scope lists are empty. Only the six authorized knowledge files changed. Commons, Proto, schema, migrations, dependencies/lockfiles, public APIs, shared configuration definitions, gateway, frontend, deployment, root docs, feature contracts, task scope, and scripts are unchanged by TASK-007.

## 5. SPEC Comparison Result

PASS. Reviewer R5 independently confirmed typed application-owner configuration errors are preserved across resume access resolution; missing client and nil owner response each emit exactly one `configuration_failure/error` terminal. Genuine owner RPC error remains `source_failure`, forbidden remains `authorization_failure`, not-found remains `not_found`, and bad request remains `domain_validation_failure`.

## 6. SDD Comparison Result

PASS against SDD sections 3, 6, 7, 8, 9, 10, 11, 12, and 13. Observability is transport-neutral in the application port and adapted to Zap only at the interface boundary. Prompt selection remains uncached/per-request, runtime policy immutable, fallback explicit, evidence bounded, aggregation deterministic, and persistence/public contracts unchanged.

## 7. Acceptance Comparison Result

- Active Prompt metadata: local read-only MySQL evidence shows version `2`, active, role `system` for all three exact agent types; no Prompt content or secret was selected.
- Per-request use: a repeated test invokes each of the three agent types twice and proves six exact `agent_type + system` loads and six separate System/User provider calls.
- Safe logs: runtime/Zap tests prove that phone/body/API-key/JWT/UUID/ULID/normal external IDs never appear raw and retain bounded same-process correlation. Prompt/model contract-length UTF-8 and punctuation retain non-empty bounded correlation; invalid/control text fails closed; artifacts are rescanned after final report/evidence updates.
- Knowledge: only six allowed files changed; validator/reference checks pass; detector reports `update_required` with all updated/unchanged verdicts and no stale/conflict/coverage gap.
- Smoke: fake-provider native resume and candidate-match orchestration plus real persistence/rollback tests pass. Live API/provider execution is explicitly skipped because required services were not listening and no safely synthetic authorized database fixture was available.
- Compatibility/risk report: present and truthful.

## 8. Test Commands and Results

- `GOWORK=off go test ./internal/application/recruiting_intelligence -run 'TestRuntimeLoadsEveryRecruitingSystemPromptOnEveryRequest|TestResumeFallbackObservationContainsClassificationNotProviderBody' -count=10` — PASS.
- Native fake-provider resume/match/privacy smoke at `-count=3` — PASS.
- Real `NativeStore` resume version/rollback and candidate version/latest/evidence/Agent-run tests at `-count=3` — PASS.
- `GOWORK=off go test ./ai -run 'TestGenerateStructured' -count=3` in Commons — PASS.
- `GOWORK=off go test ./...` in AI Agent — PASS.
- R4 focused terminal suite, including both newly added resume owner-configuration boundaries and the full existing terminal matrices, at `-count=10` — PASS.
- Real `NativeStore` resume/candidate persistence tests at `-count=10` — PASS.
- Commons structured completion at `-count=10` — PASS.
- Knowledge validator tests, formal validation, reference checks, impact detection, TASK scope, `git diff --check`, and `agent-check.sh` — PASS.
- Corrected privacy artifact/forbidden-log-field scan — PASS. An earlier combined regex had a shell-quoting error and was discarded; an initial timing wrapper used unsupported macOS `%3N` after the underlying suite passed, so the full suite was rerun with a portable timer. Neither invalid wrapper result is presented as evidence.

## 9. Knowledge Impact

Result: `update_required`; detector exit code `0`; coverage gap `false`. The five R3-affected documents updated in this repair are `agent-runtime`, `ai-configuration-governance`, `resume-intelligence`, `debug-resume-intelligence`, and `resume-sensitive-data`; `service-boundaries` remains an earlier TASK-007 update and is still verified. The remaining declared or mechanically routed active documents are `UNCHANGED`: system overview, local development, semantic retrieval, memory/context, Agent retrieval, Agent Skill, embedding fallback, MCP governance/audit, knowledge coverage, API/gateway contracts, protobuf/migration runbook, and protobuf synchronization.

## 10. Privacy Result

PASS, including independent review. Runtime and Zap tests prove external IDs and configured malicious identities are never reversible, HMAC correlation is stable/distinct/idempotent/bounded, and contract-length UTF-8 identity remains correlatable. Artifact scans found no raw candidate data, Prompt body, model output, evidence body, credential, token, DSN, or presigned URL.

## 11. Risks and Follow-up Items

- Live provider/API smoke remains skipped for the explicit local service/fixture reason; this is not presented as a live PASS.
- Existing concurrent version allocation, conservative CJK validation, heuristic fidelity, process-lived configuration fingerprints, and request-local complete legacy scoring text remain documented residual risks.
- Root must run the final all-TASK evidence and pipeline-state completion audit.

## 12. Whether the Next TASK Can Start

Yes for pipeline finalization. TASK-007 is the final TASK and independent Reviewer R5 returned `verdict: 通过`; no later implementation TASK remains.

## 13. Independent Review Round 1 History

Independent Reviewer R1 returned `verdict: 不通过`:

- `R1-H1` High: the nine-field Observation omitted SPEC §8 request/resource IDs, parser/scorer version, and bounded counts; generation-stage success could be mistaken for overall success because source/aggregation/timeout/persistence had no complete terminal model, including persistence failures after generation.
- `R1-M1` Medium: `resume-intelligence.md`, `debug-resume-intelligence.md`, and `resume-sensitive-data.md` claimed verified/UPDATED structured runtime behavior without direct implementation and test `source_refs`.
- `R1-M2` Medium: the debug runbook lacked the exact metadata-only active/latest SQL, focused commands and expectations, safe synthetic smoke procedure, and explicit port/provider skip decision commands.

This failed verdict is preserved. No R1 finding was removed or relabeled.

## 14. Repair Summary — R1

### Failed Check

Independent self-review round 1 failed R1-H1, R1-M1, and R1-M2.

### Root Cause

Observability stopped at component generation boundaries and its event schema did not encode stage versus terminal semantics. Knowledge prose was updated without adding direct refs for each new claim, and the runbook described intent rather than reproducible privacy-safe commands.

### Files Changed

- `structured_runtime.go`, `structured_runtime_observability_test.go`
- `native_servers.go`, `recruiting_observability.go`, `recruiting_observability_test.go`
- `.knowledge/domains/resume-intelligence.md`
- `.knowledge/runbooks/debug-resume-intelligence.md`
- `.knowledge/pitfalls/resume-sensitive-data.md`
- TASK-007 report/evidence

### Fix Summary

- Added only metadata-safe fields and fixed classifications; raw errors/bodies remain structurally impossible.
- Made all component events `terminal=false`; native resume/match operations now emit one and only one terminal after source, generation/aggregation, and persistence resolution.
- Added primary/fallback/source/provider/schema/aggregation/timeout/persistence regression paths, persistence-failure no-success proof, terminal cardinality checks, versions/counts, and expanded allow-list/privacy assertions.
- Added direct implementation/test source refs to exactly the three R1 documents and replaced the runbook gap with metadata-only SQL and safe executable procedures.

### Re-run Commands and Results

- Focused observability/orchestration matrix at `-count=10` — PASS.
- Full `GOWORK=off go test ./...` in AI Agent — PASS.
- Commons structured completion at `-count=10` — PASS.
- Knowledge validation, reference validation, and impact detection — PASS; `update_required`, no stale/conflict/coverage gap.
- Scope, agent-check, evidence validation, privacy scan, and diff checks passed for the R1 repair; at that point independent review round 2 remained pending.

### Remaining Risks

Live provider/API smoke remained truthfully skipped under the documented service/provider/fixture gates. At the end of R1 repair, independent review round 2 and root-owned pipeline finalization were still required; round 2 subsequently failed as recorded below.

## 15. Independent Review Round 2 History

Independent Reviewer R2 returned `verdict: 不通过` with one High finding:

- `R2-H1` High: external `X-Request-ID` was propagated and logged verbatim; the observation boundary did not prove request IDs opaque, and operation/stage/category/outcome/agent/fallback plus Prompt/model/parser/scorer identity remained caller-controlled free strings. The privacy test put its marker under an unrelated context key instead of the real request-ID path. R1-M1 and R1-M2 were confirmed closed.

This failed verdict is preserved. No R1/R2 finding was removed or relabeled.

## 16. Repair Summary — R2

### Failed Check

Independent self-review round 2 failed R2-H1.

### Root Cause

The event type was an allow-list of field names, but values crossed application/native observer and Zap boundaries without validation. Gateway and platform metadata intentionally propagate external `X-Request-ID`, so the recruiting boundary could receive email, body, control-character, or oversized text. Enum-like fields and identity/version fields also lacked fail-closed value policies.

### Files Changed

- `structured_runtime.go`, `structured_runtime_observability_test.go`
- `native_servers.go`, `recruiting_observability.go`, `recruiting_observability_test.go`
- `.knowledge/architecture/agent-runtime.md`
- `.knowledge/domains/resume-intelligence.md`
- `.knowledge/runbooks/debug-resume-intelligence.md`
- `.knowledge/pitfalls/resume-sensitive-data.md`
- TASK-007 report/evidence

### Fix Summary

- Added one deterministic `NormalizeObservation` policy and invoked it before Runtime/native observers and again at the Zap adapter.
- Request IDs accept only 8–64 ASCII safe-ID characters; invalid values become empty. Fixed classifications use explicit whitelists and unknown values become fixed `unknown`. Prompt/model/parser/scorer identity/version uses bounded safe metadata forms; invalid text becomes empty. Negative IDs/versions/durations/counts are clamped and counts/duration are capped.
- Injected email, resume body, raw output, evidence, API-key, control-character, and oversized markers directly into the real propagated request ID and every Observation string field. Logs contain no marker, while valid database Prompt names, model identifiers, versions, and classifications remain unchanged.
- Preserved the complete terminal matrix and updated routed knowledge claims to describe the actual boundary.

### Re-run Commands and Results

- Focused observation/log boundary plus terminal matrix at `-count=10` — PASS.
- Full `GOWORK=off go test ./...` in AI Agent — PASS.
- Commons structured completion at `-count=10` — PASS.
- Knowledge validator tests, formal validation, references, and impact detection — PASS; `update_required`, no reported stale/conflict/coverage gap.
- Scope, agent-check, evidence/privacy, and diff checks — recorded after report/evidence finalization below.

### Remaining Risks

Live provider/API smoke remains truthfully skipped under the documented service/provider/fixture gates. Independent review round 3 subsequently failed as recorded below.

## 17. Independent Review Round 3 — Final Configured Round

Independent Reviewer R3 returned `verdict: 不通过`:

- `R3-H1` High: safe-alphabet sensitive request IDs such as phone/body/token-shaped values can still be logged verbatim, while the 80-byte ASCII Prompt/model identity policy can drop legitimate UTF-8 and schema-length persisted values.
- `R3-H2` High: validation, missing dependency, authorization, disabled capability, compatibility-read, and other native resume/match early-return paths are not all covered by method-boundary terminal finalization and can emit zero terminal events.

All independently rerun tests, scope, Harness, knowledge, evidence-pending, privacy scan, and diff checks passed, but those checks do not invalidate the two behavioral findings. The live provider/API smoke skip remains truthful and the fake-provider plus real-persistence evidence is accepted only for that skip.

The configured `max_review_rounds=3` is exhausted. No further Fixer or Reviewer was started; TASK-007 and the pipeline are blocked pending explicit user direction.

## 18. Repair Summary — R3 (User-Authorized One-Time Extension)

### Failed Check

Independent review round 3 failed R3-H1 and R3-H2. The user explicitly authorized one Fixer R3 and one independent Reviewer R4 beyond the ordinary three-round limit.

### Root Cause

R3-H1: the prior boundary treated a bounded safe alphabet as a trust signal, so low-entropy or secret-shaped request IDs could remain reversible, while an 80-byte ASCII identity filter discarded valid persisted Prompt/model metadata. R3-H2: terminal events were manually emitted inside generation branches rather than owned by the two public method boundaries, leaving validation, dependency, authorization, disabled, and compatibility returns with no terminal.

### Files Changed

- `structured_runtime.go`, `structured_runtime_observability_test.go`
- `resume_profile.go`, `job_requirement.go`, `candidate_match.go`
- `native_servers.go`, `recruiting_observability_test.go`
- `.knowledge/architecture/agent-runtime.md`
- `.knowledge/domains/ai-configuration-governance.md`
- `.knowledge/domains/resume-intelligence.md`
- `.knowledge/runbooks/debug-resume-intelligence.md`
- `.knowledge/pitfalls/resume-sensitive-data.md`
- TASK-007 report/evidence and compatibility/risk report

### Fix Summary

- Replaced reversible request-ID handling with process-random-key HMAC-SHA256, fixed domain labels, and bounded prefixes. Random-key initialization fails closed. The unexported normalized state keeps all three normalization layers idempotent without accepting an externally forged token shape.
- Represented Prompt/model identity as bounded non-reversible correlations aligned with the persisted 256/128-byte contracts. Valid UTF-8/Unicode/punctuation remains stably correlatable; control/invalid UTF-8 fails closed; overlong input uses a separate correlation domain. Parser/scorer text is readable only for exact internal constants.
- Installed one deferred finalizer before validation in each public method and removed manual terminal emission. Every actual return path sets a fixed category; intermediate success remains non-terminal and overall success follows persistence or a successful compatibility read.
- Made component outcome observation nil-safe so missing structured runtime dependencies return a classified non-success rather than panic before normal completion.

### Re-run Commands and Results

- Focused HMAC/privacy, early-return finalizer, existing generation matrix, persistence-failure, and deadline suite at `-count=10` — PASS.
- Full AI Agent `GOWORK=off go test ./...` — PASS.
- Real recruiting persistence tests at `-count=10` — PASS.
- Commons structured completion at `-count=10` — PASS.
- Knowledge validator tests, formal validation, references, and impact detection — PASS; `update_required`, no stale/conflict/coverage gap.
- TASK scope, `agent-check.sh`, feature pending-state validation, gofmt, and `git diff --check` — PASS before final artifact scan.

### Remaining Risks

The live provider/API smoke remains truthfully skipped under the documented service/provider/synthetic-fixture gates. HMAC correlation is intentionally process-local, so it supports within-process incident correlation rather than cross-restart tracking. Independent Reviewer R4 subsequently failed as recorded below.

## 19. Independent Review Round 4 — User-Authorized Extension

Independent Reviewer R4 returned `verdict: 不通过` with one High finding:

- `R4-H1`: `ParseResumeProfile` converts the typed application-owner authorization/configuration error into a normal response before terminal classification. Missing or nil owner dependencies therefore emit exactly one terminal but incorrectly use `source_failure` instead of `configuration_failure`; both cases are absent from the exhaustive repeated matrix. The candidate-match path already classifies the same condition correctly.

R4 independently confirmed R3-H1 closed and reran the HMAC/privacy/identity and terminal suites at `-count=10`, AI Agent full tests, persistence and Commons tests, knowledge/Harness/evidence/privacy/diff checks—all PASS. Those green checks do not invalidate the uncovered classification defect.

The one-time Fixer R3 / Reviewer R4 extension is exhausted. No additional Fixer or Reviewer was started; TASK-007 and the pipeline are blocked pending explicit user direction.

## 20. Repair Summary — R4 (User-Authorized Final Extension)

### Failed Check

Independent review round 4 failed R4-H1. The user explicitly authorized one fresh Fixer R4 and one independent Reviewer R5; file scope remains unchanged.

### Root Cause

`resolveResumeProfileAccess` converted `recruitingAuthError` into a public `GetResumeProfileResponse` before returning to `ParseResumeProfile`. The parser finalizer therefore saw only an internal response code and classified missing/nil application-owner dependencies as generic `source_failure`. The exhaustive resume boundary table omitted both cases.

### Files Changed

- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/recruiting_observability_test.go`
- TASK-007 report/evidence and compatibility/risk report

### Fix Summary

- Preserved the typed authorization error alongside the unchanged public response across the internal resume access-resolution boundary.
- Added an explicit fixed terminal category to typed configuration errors and removed message-substring classification from terminal auth categorization.
- Kept `GetResumeProfile` response behavior unchanged while allowing `ParseResumeProfile` to classify the typed category.
- Added missing application-owner client and nil owner response to the exhaustive table; every case continues to assert exactly one terminal.

### Re-run Commands and Results

- Focused resume/candidate terminal matrices at `-count=10` — PASS.
- Full AI Agent `GOWORK=off go test ./...` — PASS.
- Real recruiting persistence tests at `-count=10` — PASS.
- Commons structured completion at `-count=10` — PASS.
- Knowledge, scope, Harness, pending-review evidence, privacy/forbidden-log, and diff checks — PASS as recorded after final artifact updates.

### Remaining Risks

The existing live-provider/API smoke skip and previously documented residual risks are unchanged. Independent Reviewer R5 subsequently passed as recorded below.

## 21. Independent Review Round 5 — Final Verdict

Fresh independent Reviewer R5 returned `verdict: 通过` with no Critical, High, Medium, or Low findings.

R5 independently verified:

- typed resume access errors survive the internal resolver boundary;
- missing and nil application-owner dependencies each emit exactly one `configuration_failure/error` terminal;
- genuine owner RPC error, forbidden, not-found, and bad-request classifications remain correct;
- `GetResumeProfile` public behavior is unchanged and Parse classification does not depend on response text;
- HMAC/privacy/identity and all terminal matrices pass at `-count=10`;
- AI Agent full tests, real persistence, Commons structured completion, knowledge, scope, Harness, evidence, privacy, and diff checks pass;
- the live provider/API smoke skip is truthful, with fake-provider orchestration and real persistence evidence accepted for the documented skip.

TASK-007 is complete and ready for root-owned pipeline finalization.
