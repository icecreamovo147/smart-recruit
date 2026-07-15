# Recruiting Intelligence Runtime Parity SPEC

## 1. Background

The microservice runtime restored the public resume-profile and candidate-match RPCs, versioned persistence, and authorization, but replaced the original `dev` structured recruiting pipeline with hard-coded prompts sent through the generic HR Markdown completion path. As a result, active database prompts are not used, resume output is weakly validated, job requirements are not structured, and candidate matching asks the model to produce an aggregate score that is incompatible with the active `candidate_match_evaluator` prompt schema.

## 2. Goals

- Restore database-backed runtime use of `resume_profile_extractor`, `job_requirement_extractor`, and `candidate_match_evaluator` system prompts.
- Preserve true System/User message roles for structured generation.
- Restore strict resume and job-requirement validation with configurable deterministic fallbacks.
- Restore deterministic-first, per-requirement candidate matching and deterministic aggregate scoring.
- Preserve existing external APIs, authorization, tables, versioning, and `agent_run_id` behavior.
- Produce privacy-safe diagnostics, TASK evidence, and updated project knowledge.

## 3. Non-Goals

- No new frontend page, button, HTTP route, RPC, or protobuf field.
- No database schema or migration change.
- No package dependency, `go.mod`, `go.sum`, package manifest, or lockfile change.
- No Prompt Management UI behavior change.
- No historical profile or evaluation backfill.
- No Recruitment/AI Agent data-ownership redesign.

## 4. User-Facing Behavior

- Clicking “重新解析画像” uses the active `resume_profile_extractor/system` template and returns a newly versioned profile on success.
- A model response such as `"skills":["Go"]` is rejected as invalid schema. When fallbacks are enabled, the heuristic extractor is used; when disabled, no new profile is saved.
- Candidate match evaluation first derives structured job requirements, evaluates each requirement against candidate evidence, and deterministically aggregates the result.
- Existing response shapes and frontend behavior remain unchanged.

## 5. Functional Requirements

### FR-001 Structured completion

- The runtime shall provide an internal structured-completion operation accepting distinct System and User messages.
- Structured completion shall reuse the existing model selection, retry, circuit-breaker, concurrency, and timeout behavior.
- Structured completion shall not inject the generic HR data-analysis or Markdown system prompt.
- Existing generic completion behavior shall remain compatible.

### FR-002 Prompt selection

- Each structured recruiting operation shall load the active `system` prompt for its exact `agent_type` on every request.
- Supported agent types are `resume_profile_extractor`, `job_requirement_extractor`, and `candidate_match_evaluator`.
- Prompt IDs shall not be hard-coded.
- Missing, empty, wrong-role, or unavailable prompts shall be classified errors and shall not silently fall back to hard-coded LLM prompts.

### FR-003 Runtime policy

- The AI Agent service shall consume the existing configuration fields `structured_resume_parse`, `candidate_match`, `candidate_match_semantic`, `candidate_match_shadow`, `fallbacks`, `resume_parse_timeout`, and `candidate_match_timeout`.
- No shared configuration schema or default shall change.
- Runtime policy shall be request-safe and shall not be mutated temporarily by concurrent requests.

### FR-004 Resume profile extraction

- The database prompt shall be the System message; parsed resume text and minimal metadata shall be the User message.
- Output shall be one JSON object with the configured schema.
- Validation shall reject unknown or malformed top-level structures, missing required arrays, invalid child items, negative experience, invalid email when present, and invalid date values.
- A successful profile shall contain a name or at least one material education, experience, project, or skill item.
- Child collections shall be normalized, deduplicated where appropriate, and emitted in stable order.
- LLM and heuristic parser metadata shall remain request-scoped.

### FR-005 Resume fallback

- With `fallbacks=true`, Prompt, provider, empty-response, JSON, or schema failure shall invoke a deterministic heuristic extractor.
- With `fallbacks=false`, the error shall be returned and no new profile version shall be persisted.
- Both primary and fallback failure shall remain an explicit non-success.

### FR-006 Job requirement extraction

- The active `job_requirement_extractor/system` prompt shall receive job title, department, location, description, and requirements as the User message.
- Output shall use `job-requirement-profile-v1` with stable unique IDs, valid categories and priorities, positive weights, knockout flags, and aliases.
- The extractor shall reject empty requirement sets, invalid enums, duplicate IDs, and invalid weights.
- Small floating-point weight drift may be deterministically normalized; structurally invalid results shall not be repaired silently.
- With `fallbacks=true`, primary failure shall use a deterministic heuristic requirement extractor.
- The job-requirement profile is an internal evaluation input and shall not require a new table or public API.

### FR-007 Evidence index

- Evidence may come only from resume skills, experiences, projects, educations, resume text, and candidate profile fields already available to the runtime.
- Evidence shall retain source table and source ID where available.
- Snippets shall be bounded and sensitive contact data shall be redacted or excluded.
- Evidence ordering and deduplication shall be deterministic.

### FR-008 Per-requirement evaluation

- Every structured job requirement shall receive exactly one result.
- Deterministic matching shall run first; the LLM evaluator shall only run when direct deterministic evidence is insufficient.
- The active `candidate_match_evaluator/system` prompt shall evaluate one requirement at a time.
- Results shall validate status, score, confidence, risk, evidence source, and evidence completeness.
- Unknown or duplicate requirement IDs and model-created requirements shall be rejected.
- The model shall not decide the final overall score or recommendation.

### FR-009 Deterministic aggregation

- Aggregation shall restore the `dev` enhanced dimensions, knockout handling, capped risk penalty, recommendation thresholds, Chinese summary, strengths, and risks.
- Job requirement results and scoring policy shall be included in the existing score breakdown JSON.
- Given identical normalized inputs and component results, aggregation output shall be deterministic.
- With fallbacks enabled, enhanced-pipeline failure may use the restored legacy deterministic scorer. With fallbacks disabled, invalid primary output shall not be persisted.

### FR-010 Persistence and compatibility

- Existing profile/evaluation version increments, current/latest demotion, evidence persistence, and `agent_run_id` association shall remain transactional.
- A failed write shall roll back the complete new snapshot.
- Existing read and compare RPC behavior shall remain compatible.
- Historical rows shall not be rewritten.

## 6. Non-Functional Requirements

- No new dependencies.
- All new algorithms shall have deterministic unit tests.
- Request deadlines shall reserve time for persistence and respect configured feature timeouts.
- Structured runtime components shall be isolated from transport concerns and safe for concurrent use.

## 7. Compatibility Requirements

- Preserve all current HTTP and gRPC request/response contracts.
- Preserve current permissions, staff/application/resume/job authorization, and quota middleware.
- Preserve current database tables and migrations.
- Preserve generic HR and candidate chat behavior.

## 8. Observability and Debug Requirements

- Record request/resource IDs, stage, prompt ID/name/version, model name, parser/scorer version, duration, status, fallback type, and bounded counts.
- Do not record full prompts, resume text, candidate profile text, complete model output, secrets, or unredacted evidence.
- Persist parser/scorer/fallback provenance in existing metadata fields where representable without schema changes.

## 9. Error Handling and Fallback Requirements

- Errors shall distinguish configuration/prompt, provider, empty response, JSON syntax, schema, domain validation, timeout, and persistence failures.
- Fallback shall be controlled only by the existing runtime policy.
- Fallback success shall be observable and shall never be mislabeled as primary LLM success.
- Fallback-disabled failures shall not create misleading success snapshots.

## 10. Security and Safety Requirements

- Existing authorization is invariant.
- Candidate-sensitive text may be sent only to the configured model for the scoped operation.
- Logs, reports, evidence files, and knowledge documents shall contain no raw candidate data or credentials.
- Evidence source tables shall be allow-listed.

## 11. Acceptance Criteria

- AC-001 All three active database prompt types are loaded and used as System messages.
- AC-002 Generic Markdown system instructions are absent from structured recruiting calls.
- AC-003 `skills` string arrays fail LLM schema validation and follow the configured fallback behavior.
- AC-004 Job requirements are validated, stable, and available to the matcher without new schema or API changes.
- AC-005 Deterministic matches avoid LLM calls; missing evidence may use per-requirement LLM evaluation.
- AC-006 Final score/recommendation are deterministic and preserve knockout/risk rules.
- AC-007 Version/current/latest/agent-run/evidence persistence remains transactional and compatible.
- AC-008 Fallback-disabled invalid primary results do not create new snapshots.
- AC-009 Full AI Agent and Commons AI tests pass, along with Harness and evidence validators.
- AC-010 Logs and generated reports contain no raw resume or candidate-sensitive content.

## 12. Out of Scope

- Standalone job requirement extraction API or UI.
- Prompt-template schema redesign or additional prompt versions.
- Provider-native JSON Schema/response-format negotiation that requires new provider contracts.
- Concurrent version-write conflict redesign beyond preserving current behavior.

## 13. Assumptions Requiring Confirmation

- Confirmed: shared `smart-recruit-commons/ai` may be modified only to add internal structured System/User completion while preserving legacy behavior.
- Confirmed: no external API, dependency, Proto, or migration changes.

## 14. Open Questions

None. Implementation decisions are fixed by this SPEC and the SDD.
