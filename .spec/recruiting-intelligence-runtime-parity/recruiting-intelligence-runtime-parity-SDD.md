# Recruiting Intelligence Runtime Parity SDD

## 1. Existing Architecture Summary

The AI Agent service owns native recruiting-intelligence gRPC adapters and uses `NativeStore` for model configuration, active Prompt CRUD, recruiting source reads, and versioned profile/evaluation persistence. `smart-recruit-commons/ai.Client` owns provider adapters and resilience controls. Current resume and candidate-match generation is embedded in `native_servers.go`, uses hard-coded prompts, and calls generic `GenerateRecruitingReply`.

The `dev` reference used dedicated LLM and heuristic extractors, a job-requirement profile, a privacy-bounded evidence index, deterministic-first per-requirement matching, and a deterministic score aggregator.

## 2. Problem Analysis

- Prompt governance is disconnected from the runtime.
- Generic HR Markdown instructions conflict with structured JSON instructions.
- Resume JSON is weakly decoded and has no fallback.
- The active candidate evaluator prompt is per-requirement, while the current runtime expects aggregate JSON.
- Runtime feature flags and timeouts are loaded but not injected.
- Provenance and privacy-safe diagnostics are incomplete.

## 3. Proposed Design

Implement focused components while retaining the current external gRPC adapter and persistence contracts:

1. `StructuredCompletionProvider` accepts distinct system/user messages and returns content plus model provenance.
2. `RecruitingRuntimePolicy` is constructed once from existing service configuration.
3. `RecruitingPromptLoader` loads an active system Prompt per operation for every request.
4. `ResumeProfileExtractor` combines strict LLM extraction with optional heuristic fallback.
5. `JobRequirementExtractor` combines strict LLM extraction with optional heuristic fallback.
6. `EvidenceIndex` and `CandidateRequirementEvaluator` implement deterministic-first matching and optional LLM evaluation.
7. `CandidateMatchAggregator` calculates the final result.
8. The native gRPC adapter authorizes, resolves sources, calls these components, and persists through the existing store.

## 4. Data Structure Changes

Internal-only structures:

- Structured completion request/result with system prompt, user prompt, model option, content, and model name.
- Prompt descriptor with ID, name, version, role, type, and content.
- Runtime policy with feature flags, fallback flag, and timeouts.
- Resume extraction result with raw JSON and request-scoped provenance.
- Job requirement profile/items and extraction provenance.
- Evidence units/index and per-requirement results.
- Enhanced score breakdown and deterministic aggregation result.

`RecruitingMatchSource` may be extended internally with parsed resume and candidate-profile data already available in the shared database. No persisted schema changes are allowed.

## 5. API and Interface Changes

- Add an exported Commons AI client method for structured completion; it is an internal repository API and must preserve existing methods unchanged.
- Add optional/internal AI Agent provider and Prompt-loader ports.
- Extend `RuntimeDeps` with immutable recruiting policy options.
- Do not change protobuf, HTTP, or gRPC contracts.

## 6. Algorithm or Workflow Changes

### 6.1 Structured completion

Build exactly two messages: configured System content and scoped User content. Call the existing model client through its retry/circuit/concurrency wrapper. Return model content and provenance without injecting generic Markdown rules.

### 6.2 Resume profile workflow

Validate policy and source, load Prompt, calculate input hash, call LLM within the feature deadline while reserving persistence time, extract exactly one JSON object, decode with unknown-field rejection, normalize and validate domain fields, then persist. On eligible failure and enabled fallback, run the deterministic heuristic extractor and persist its provenance.

### 6.3 Job requirement workflow

Load the job Prompt, pass normalized job text, decode `job-requirement-profile-v1`, validate stable unique IDs/enums/weights/aliases, normalize minor weight drift, and compute an input hash. On eligible failure and enabled fallback, derive deterministic requirements from skills, education, experience, and textual requirements.

### 6.4 Evidence workflow

Build at most 100 evidence units. Bound snippets to 240 runes, redact contact data, allow-list source tables, normalize terms, retain source IDs, and produce stable ordering. Candidate phone/email are excluded.

### 6.5 Requirement evaluation workflow

For each requirement in stable order, run deterministic term/category matching. Only a deterministic `missing` result may invoke the LLM matcher. The LLM evaluates one requirement, and the service overwrites/validates its requirement ID. Invalid LLM results follow policy rather than creating new requirements.

### 6.6 Aggregation workflow

Restore the `dev` enhanced scoring semantics:

- must-have 0.40;
- core skills 0.25;
- experience 0.20;
- growth 0.10;
- missing/conflict and weak-evidence penalties capped at 20;
- missing knockout caps score at 40 and forces strong-not-recommend;
- recommendation thresholds follow the `dev` enhanced scorer.

The intentionally unallocated 0.05 is preserved for parity rather than silently changing historical score semantics. The score breakdown records requirements, results, dimensions, must-have summary, policy, input hash, scorer type, and fallback state.

## 7. Configuration Design

Consume existing `cfg.Agent.Features` fields in `cmd/ai-agent-service/main.go` and pass an immutable policy through `RuntimeDeps`. Defaults remain owned by `serviceconfig`; this feature shall not modify shared configuration definitions or config files.

Prompt selection uses current active-Prompt persistence semantics. Duplicate active rows are resolved by the existing latest-updated/latest-ID ordering; the selected prompt identity is logged. No cache is introduced.

## 8. Compatibility Strategy

- Keep existing gRPC adapter methods, response mappers, authorization calls, and store transactions.
- Preserve generic completion methods and tests.
- Keep legacy deterministic scoring as the enhanced-pipeline fallback when enabled.
- Do not rewrite historical snapshots; new calls naturally create new versions.

## 9. Error Handling and Fallback Design

Classify errors by stage. Provider/prompt/JSON/schema failures are fallback-eligible; authorization and persistence failures are not. With fallbacks disabled, return non-success and do not persist invalid data. With fallbacks enabled, record primary failure category and fallback provenance. If both paths fail, return a joined diagnostic without sensitive content.

For per-requirement matching, deterministic results are always available. LLM failure retains the deterministic missing result only when fallback policy allows it; otherwise the enhanced evaluation aborts before persistence.

## 10. Observability and Debug Output Design

Use structured fields for request and resource IDs, stage, prompt identity/version, model name, parser/scorer type, duration, counts, and fallback category. Never log message bodies, parsed resume text, full output, or unredacted evidence. Existing raw JSON profile persistence remains a business record, not a log.

## 11. Testing Strategy

- Commons AI unit tests for message roles and legacy compatibility.
- AI Agent unit tests for policy, Prompt loading, strict resume decoding, heuristic fallback, requirement validation/extraction, evidence privacy, per-requirement matching, and aggregation.
- Persistence tests for version/current/latest/evidence/rollback/agent-run behavior.
- Native gRPC tests for authorization and end-to-end orchestration with fake providers.
- Full `GOWORK=off go test ./...` in Commons and AI Agent modules.
- Feature, scope, evidence, pipeline-state, and knowledge validators.

## 12. Migration Risks

- New scores may differ from the current incorrect aggregate-LLM path; versioned evaluations preserve history.
- Heuristic fallback may produce lower-fidelity results and must be clearly marked.
- The `dev` enhanced weights sum to 0.95; parity explicitly preserves this.
- Live provider smoke tests depend on local credentials and may be skipped only with truthful evidence.

## 13. Implementation Boundaries

- TASK-001 alone may modify `smart-recruit-commons/ai`, under the recorded user authorization.
- No TASK may modify Proto, migrations, dependencies, lockfiles, permissions, public APIs, root docs, deployment, or frontend files.
- Job requirements remain internal and transient.
- Feature-owned docs remain under this feature directory; durable knowledge changes are limited to routed `.knowledge` files in TASK-007.

## 14. Alternatives Considered

- Expanding the hard-coded aggregate Prompt was rejected because it bypasses governance and conflicts with the active per-requirement schema.
- Leniently accepting string skills was rejected because it hides invalid model output.
- Adding job-requirement tables or APIs was rejected as unnecessary scope expansion.
- Duplicating provider clients in AI Agent was rejected in favor of the authorized Commons AI extension.

## 15. Assumptions Requiring Confirmation

The shared Commons AI authorization is confirmed by the user and must be copied into TASK-001 pipeline state and evidence.

## 16. Open Questions

None.
