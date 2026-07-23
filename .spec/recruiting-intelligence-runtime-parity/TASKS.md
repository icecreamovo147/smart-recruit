# TASKS - recruiting-intelligence-runtime-parity

TASKs execute strictly in order. Every TASK uses its own reliable `base_sha` and `base_tree`, independent Developer and Reviewer agents, scope/evidence validation, and no more than three review rounds.

| TASK | Title | Depends on | Human confirmation |
|---|---|---|---|
| TASK-001 | Structured Provider and Prompt Runtime | - | required; already granted for the listed shared AI files |
| TASK-002 | Runtime Policy Wiring | TASK-001 | no |
| TASK-003 | Restore Resume Profile Extraction | TASK-002 | no |
| TASK-004 | Restore Job Requirement Extraction | TASK-003 | no |
| TASK-005 | Restore Per-Requirement Candidate Matching | TASK-004 | no |
| TASK-006 | Restore Deterministic Aggregation and Persistence | TASK-005 | no |
| TASK-007 | Integration, Observability, and Knowledge | TASK-006 | no |

## TASK-001 - Structured Provider and Prompt Runtime

### Goal

Add a dedicated structured-generation path that sends explicit System and User messages and loads the active database prompt per request without adding the generic HR Markdown prompt.

### Work

- Add an internal structured completion entry point to `smart-recruit-commons/ai`, reusing its timeout, retry, concurrency, and circuit-breaker controls.
- Add an AI Agent internal prompt loader and structured provider port/adapter.
- Select one active system prompt by `agent_type + system`; preserve prompt name and version metadata for safe observability.
- Keep the change internal: no public HTTP/gRPC, Proto, dependency, or configuration-schema changes.

### Acceptance focus

- System/User roles reach the model unchanged and the generic recruiting Markdown system prompt is absent.
- Active prompt selection and version changes are tested.
- Provider errors remain classified and do not leak message bodies.
- The existing user grant is recorded in pipeline state and evidence exactly as required by `AGENT_RULES.md`.

## TASK-002 - Runtime Policy Wiring

### Goal

Inject the existing feature flags and timeouts into the recruiting-intelligence runtime without changing shared configuration definitions.

### Work

- Consume `structured_resume_parse`, `candidate_match`, `candidate_match_semantic`, `candidate_match_shadow`, `fallbacks`, `resume_parse_timeout`, and `candidate_match_timeout` from the existing service config.
- Define an immutable runtime policy with explicit defaults and pass it into the internal runtime.
- Test enabled/disabled and timeout mapping behavior.

## TASK-003 - Restore Resume Profile Extraction

### Goal

Restore strict database-prompt-driven resume profile extraction with configurable heuristic fallback and existing transactional version semantics.

### Work

- Use `resume_profile_extractor/system` via TASK-001.
- Strictly decode exactly one JSON value and validate field types, skill objects, arrays, dates, and a non-empty useful profile.
- Treat `skills: ["Go"]` as a schema error; use heuristic fallback only when policy enables it.
- Preserve input hash, parser version, version increment, current switching, and transactional storage.
- Never persist a new invalid profile when fallback is disabled.

## TASK-004 - Restore Job Requirement Extraction

### Goal

Restore an internal job requirement extractor used by matching, driven by `job_requirement_extractor/system`.

### Work

- Validate non-empty requirements, unique IDs, category, priority, weight, knockout, aliases, and count limits.
- Normalize only permitted floating-point weight drift deterministically.
- Use heuristic fallback only when policy enables it.
- Add no public endpoint, table, or Proto field.

## TASK-005 - Restore Per-Requirement Candidate Matching

### Goal

Build trusted candidate evidence and evaluate every job requirement deterministically first, invoking `candidate_match_evaluator/system` only for missing judgments.

### Work

- Build a bounded, redacted evidence index with real source IDs.
- Prefer deterministic evidence matches; call the structured matcher only when evidence is insufficient.
- Validate status, score, confidence, risk, and evidence provenance.
- On an individual model failure, retain deterministic results and apply the configured fallback policy.

## TASK-006 - Restore Deterministic Aggregation and Persistence

### Goal

Restore dev-equivalent deterministic scoring, knockout, risks, recommendation, Chinese summary, and existing evaluation persistence semantics.

### Work

- Aggregate requirement results into dimensions and total score; the model never selects the final total.
- Apply knockout caps, risk deductions, recommendations, and deterministic Chinese summaries.
- Preserve `agent_run_id`, evaluation version increment, latest switching, evidence truncation, and transactional writes.
- Do not rewrite historical results.

## TASK-007 - Integration, Observability, and Knowledge

### Goal

Prove end-to-end parity, add privacy-safe stage observability, update routed knowledge, and publish final compatibility/risk evidence.

### Work

- Verify all three active database prompts are actually loaded.
- Log only prompt identity/version, model, stage, fallback decision, outcome, and duration.
- Update only the knowledge files allowed by `task-scope.json`.
- Run local resume reparse and candidate match smoke tests, or record an explicit provider-dependent skip with truthful evidence.
- Produce the final compatibility and risk report used by the pipeline summary.

## Global hard stops

Stop and request user direction if any TASK requires Proto, database schema/migration, dependency/lockfile, public HTTP/gRPC, permission/security, CI/global configuration, root documentation, or out-of-scope changes; if contracts conflict; if a reliable baseline/evidence cannot be produced; if the same finding fails repair twice; or if a TASK exceeds three review rounds.
