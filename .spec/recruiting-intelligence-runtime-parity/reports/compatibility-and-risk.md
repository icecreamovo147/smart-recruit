# Recruiting Intelligence Runtime Parity — Compatibility and Risk

## Outcome

The current AI Agent implementation restores the `dev` structured recruiting pipeline without changing public HTTP/gRPC contracts, Proto, schema, dependencies, permissions, or historical rows. Fresh independent Reviewer R5 returned `verdict: 通过`; TASK-007 is complete and ready for root-owned pipeline finalization.

## Dev / Current Parity

| Capability | `dev` reference | Current implementation | Evidence |
|---|---|---|---|
| Structured model call | Distinct System/User messages | Dedicated Commons structured call; generic HR Markdown prompt is absent | Commons structured tests and AI Agent runtime tests |
| Prompt governance | Database-backed recruiting Prompts | Active `system` Prompt loaded on every request for all three exact agent types | Per-request Prompt test plus read-only local metadata check |
| Resume extraction | Strict LLM schema with heuristic fallback | Strict object/child/date/domain validation; policy-controlled deterministic fallback; invalid fallback-disabled output is not saved | Native resume orchestration and real persistence/rollback tests |
| Job requirement extraction | Internal validated profile | `job-requirement-profile-v1`, bounded/unique requirements, exact enums, stable IDs, normalized minor drift, deterministic fallback | Job requirement component tests |
| Candidate evidence | Bounded trusted sources | Allow-listed persisted source IDs, deterministic order, contact redaction, 100-unit/240-rune evaluator bounds | Candidate evidence and privacy tests |
| Requirement matching | Deterministic first, model for unresolved | Direct evidence avoids model calls; unresolved requirements use one `candidate_match_evaluator` call and validate provenance | Native structured match tests |
| Aggregate scoring | Deterministic enhanced and legacy policies | Model cannot set total/recommendation; knockout, risk cap, thresholds, Chinese summary, and `dev` legacy fallback semantics restored | Aggregator/legacy golden tests |
| Persistence | Versioned transactional snapshots | Profile/evaluation version switches, evidence, `agent_run_id`, rollback, and historical-row preservation retained | Real `NativeStore` tests |
| Observability | Privacy-safe stage diagnostics | Process-keyed HMAC correlation prevents reversible external/configured identity logging; fixed categories and one deferred method-boundary finalizer cover all early, generation, compatibility, persistence, and success returns; typed resume owner configuration failures retain `configuration_failure` across internal access resolution | R4 focused terminal matrix at `-count=10`; independent Reviewer R5 pending |

## Local Integration Evidence

- The local MySQL instance was reachable read-only. Metadata-only selection found one active version `2`, role `system` row for each of `resume_profile_extractor`, `job_requirement_extractor`, and `candidate_match_evaluator`. Prompt content and secrets were not selected.
- Fake-provider native resume reparse and structured candidate-match orchestration passed, including true System/User separation, deterministic no-model matching, fallback-disabled no-save, bounded evidence, deterministic aggregation, and Agent-run association.
- Real local SQLite `NativeStore` integration tests passed for resume version/children/rollback and candidate evaluation version/latest/evidence/Agent-run persistence.
- A live provider/API smoke was not run: AI Agent, Identity, Recruitment, and Gateway were not listening, and no explicitly synthetic authorized local resume/application fixture was available for safe provider submission. This is a provider/service-dependent `SKIP`, not a live `PASS`.

## Compatibility Invariants

- Public HTTP and generated gRPC request/response shapes are unchanged.
- Existing authorization, data-scope checks, quota, and ownership validation are unchanged.
- No Proto, migration, table, `db.sql`, dependency, `go.mod`, lockfile, shared configuration definition, frontend, gateway, deployment, or root documentation change is part of TASK-007.
- Generic HR/candidate chat completion remains compatible.
- Job requirements remain transient and internal; historical profile/evaluation snapshots are not rewritten.

## Residual Risks

- Live provider behavior still depends on local provider credentials, endpoint availability, model conformance, and a safely synthetic authorized fixture; this task does not provision them.
- Existing `MAX(version)+1` concurrent-writer allocation remains unchanged and was explicitly outside the feature scope.
- Heuristic fallback is intentionally lower fidelity and must remain observable as fallback, never primary LLM success.
- Static Simplified-Chinese validation uses a conservative marker policy and cannot perfectly classify every ambiguous CJK sentence without a new dependency.
- Structured model clients remain process-lived per complete configuration fingerprint; infrequent historical fingerprints may remain allocated until process restart.
- Complete legacy scoring text is request-local and unbounded for `dev` parity. Future diagnostics must keep it out of logs, reports, and persisted evidence.
- Recruiting HMAC correlation is intentionally process-keyed: the same raw identifier correlates within one process lifetime but changes after restart. Cross-restart request correlation must use an authorized internal trace mechanism, not reversible external header logging.

## Operational Rollback

1. Disable `structured_resume_parse` to stop structured resume reparsing, or disable `candidate_match` to stop new match evaluation. Use the existing configuration surface; do not change shared defaults for an incident workaround.
2. If enhanced semantic matching is the issue, disable `candidate_match_semantic`; with approved fallback policy enabled, the deterministic legacy scorer remains available. Disable `candidate_match_shadow` independently if shadow diagnostics are implicated.
3. Activate a previously known-good Prompt version through existing Prompt Management governance. Every later request reloads the active row; no process restart or Prompt cache invalidation is needed.
4. If a provider/model configuration is unhealthy, restore the known-good persisted configuration or stop the AI Agent service. Do not expose credentials in incident logs.
5. Existing versioned snapshots remain readable. Rollback does not delete or rewrite historical profile/evaluation rows; investigate any partial-write concern through transaction tests and metadata-only database checks.
6. Verify rollback with privacy-safe stage fields and focused fake-provider/real-persistence tests before re-enabling traffic.

## Privacy Rule

Do not attach Prompt bodies, User messages, resumes, job/candidate text, raw model responses, evidence snippets, provider error bodies, database snapshots, credentials, or candidate PII to logs, reports, evidence, or knowledge. External request IDs and configured Prompt/model names are correlatable only through bounded process-keyed HMAC tokens; numeric Prompt/version metadata, trusted internal parser/scorer versions, and synthetic fixtures are sufficient for runtime verification.
