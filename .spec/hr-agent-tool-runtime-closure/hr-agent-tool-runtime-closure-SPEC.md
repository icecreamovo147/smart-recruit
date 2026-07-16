# HR Agent Tool Runtime Closure — SPEC

## Context

The HR AI data assistant now exposes and executes Recruitment-backed tools, but configuration and runtime behavior are not closed. Empty or disabled Agent bindings can still enable default tools, an empty MCP selection executes every bound MCP tool, model-native Tool Calling may answer live-data questions without a successful Tool call, and business failures encoded as JSON are audited as success. Agent `max_iterations`, published Agent Skill versions, Prompt variables, effective Run identity/version evidence, streaming event types, and large application aggregation also remain incomplete.

## Goals

- Make Agent tool governance fail closed and side-effect safe.
- Guarantee that live recruiting facts come from successful authorized Tool results.
- Make Agent, Prompt, and published Agent Skill configuration materially and audibly affect each new Run.
- Make Tool failures and streaming events truthful.
- Bound existing HR-wide application aggregation without changing public contracts.

## Non-goals

- No new database table or migration.
- No Proto, public HTTP/gRPC, dependency, lockfile, provider, or global configuration change.
- No arbitrary code execution from Agent Skill content.
- No direct AI Agent database reads of Recruitment-owned job/application data.
- No automatic execution of side-effecting MCP or status-change operations.

## User behavior and mandatory requirements

### FR-001 — Fail-closed builtin allowlist

For an existing enabled Agent, only explicitly enabled concrete builtin bindings that are implemented by the runtime may be exposed or executed. An empty, cleared, disabled-only, abstract-only, or unknown binding set means no builtin business tools. A missing Agent may use only an explicitly documented safe platform fallback, not the current broad recruiting default set.

### MCP-001 — Explicit MCP invocation

No MCP tool may execute merely because `skill_capability_keys` is empty. MCP execution requires an explicit selected capability or a separately governed model-planned invocation that passes policy and confirmation. This feature implements the explicit-selection path only.

### ERR-001 — Truthful Tool failures

Invalid arguments, unsupported tools, authorization denial, not-found/hidden data, and downstream non-success responses must produce a typed non-nil execution error. Tool Trace and Run Step status must be `error`; error payloads must not count as useful results or as facts for the final answer.

### FR-002 — Live-data truth gate

The deterministic planner classifies job inventory, application listing, candidate lookup, analytics, candidate comparison/match, interview preparation, offer support, and status-change proposal intents. Each intent has required evidence groups; every group must be satisfied by a matching successful authorized read Tool before the runtime accepts corresponding business facts. Job-list queries such as “现在有哪些岗位” must execute `get_job_list` or an authorized equivalent even when the provider returns free text without Tool calls. Missing parameters trigger a clarification; missing/disabled Tool, Tool failure, and empty result yield deterministic no-data/error semantics and must not let the model invent entities or counts.

Required evidence groups are:

- job inventory: one matching job list/search/detail Tool selected from query shape;
- application listing: one matching all/by-job/by-status application list Tool;
- candidate lookup: `search_candidates`, plus `get_candidate_detail` when a specific candidate/application detail is requested;
- analytics: every explicitly requested metric family (total/today, status summary, trend, or heat ranking) requires its matching metric Tool; job inventory is not evidence for application metrics;
- candidate comparison: job identity/detail plus applications/candidates for that job;
- candidate match evaluation: candidate/application identity plus persisted or newly evaluated match evidence;
- interview preparation and offer support: candidate/application detail plus job detail;
- status-change proposal: candidate/application identity may be read automatically, but proposal/action tools are never auto-executed and explicit confirmation remains mandatory.

### CFG-001 — Per-Agent iteration limit

The effective enabled Agent `max_iterations`, bounded by a system safety ceiling, must control that Run's model/tool loop. The setting must not remain prompt-only metadata. Implementing this internal cross-module option requires explicit human confirmation before TASK-002 writes `smart-recruit-commons/**`.

### FR-003 — Authorized domain reads

All builtin job/application/candidate facts continue to use the current HR identity and Recruitment gRPC owner contracts. Detail and aggregation operations must not cross HR scope.

### FR-004 — Prompt runtime semantics

Only active system Prompts compatible with `hr_recruiting_agent` (with read-only legacy alias compatibility) may be bound and executed. Runtime variables are rendered from an allowlist including `hr_id`, `session_id`, `application_id`, and `current_date`; unknown variables fail closed instead of reaching the model verbatim. The HR Agent admin selector must show only compatible active Prompts, and create/update validation must reject incompatible bindings.

### FR-005 — Published Agent Skill semantics

Only enabled eligible Agent Skills with a valid `current_version_id` pointing to a published/current version may enter runtime instructions. No arbitrary previous version or description fallback is allowed. Runtime evidence records `skill_id` and the exact `version_id`; new Runs observe version changes while existing evidence remains stable.

### OBS-001 — Effective governance evidence

Every durable Run records the actual effective Agent ID/type/name rather than ID zero and fixed labels. Process/audit evidence records Prompt ID/version, Agent Skill ID/version, selected Tool names, Tool result status, and selection mode without sensitive Prompt/Skill bodies or raw personal data.

### EVT-001 — Streaming event contract

Model text deltas are persisted as `assistant.delta`; planning/context/fallback status remains `process.delta`. Final snapshots must not cause consumers to replay a duplicate full delta.

### NFR-001 — Bounded aggregation

HR-wide application/candidate aggregation must process at most 100 HR jobs, 10 pages per job at the existing 100-row page size, and 5,000 returned application records, with at most four concurrent job fetches. It must honor context cancellation and sort deterministically by job ID then application ID before higher-level filtering/pagination. A partial job failure returns successful records with `partial=true`, `failed_job_count`, and bounded non-sensitive warnings; facts may use only successful records. If every attempted job fails, return a non-nil aggregate Tool error and no useful result. It must not add a new public Recruitment RPC in this feature.

### KNOW-001 — Knowledge alignment

Routed active knowledge for Agent runtime, AI governance, MCP governance, Agent Skill, and service boundaries must match the verified implementation and cite code/test evidence.

### FR-006 — Valid application-analysis message envelope

Creating an HR application-analysis session must seed and return one non-empty user message that explicitly requests resume-to-job match analysis. Both HR route-entry and in-chat candidate analysis must submit a non-empty message that the recruiting planner classifies as candidate match evaluation. Durable Run HTTP and gRPC entry points reject blank messages before dispatch. The Anthropic Messages adapter rejects system-only input locally and must never send `messages: null` or an empty message sequence to a provider.

## Errors and fallbacks

- Disabled/unbound/unavailable required Tool: deterministic limitation message with no fabricated records.
- Empty successful Tool result: deterministic empty-state response.
- Tool argument/permission/downstream failure: error Trace/Step and safe user-visible failure; never success fallback.
- Unknown Prompt variable or invalid current Skill version: omit/fail the affected governance input and record a governance error; do not silently substitute stale content.
- Model/provider failure after useful results: privacy-safe deterministic formatter may answer only from successful Tool results.

## Security and compatibility

The feature reduces authority: tools become explicit allowlists and MCP no longer executes implicitly. Existing persisted `hr_agent` Prompts remain readable as a legacy alias but new HR Prompts use `hr_recruiting_agent`. No schema or wire contract changes are allowed. Tool arguments and results remain subject to existing redaction/audit rules.

## Acceptance summary

- Empty/disabled bindings expose and execute zero business tools.
- Empty MCP selection executes zero MCP tools.
- A fake native Tool Calling provider that directly fabricates a job answer is rejected or forced through `get_job_list`.
- Error JSON cases become non-nil errors and error traces.
- Agent iteration limits change observed tool-loop termination.
- Invalid/missing current Skill versions do not enter the runtime; valid version IDs are recorded.
- Prompt variables render from the allowlist; invalid bindings are rejected in service and filtered in UI.
- durable Run identity and delta event types are accurate.
- bounded aggregation and cumulative HR assistant behavior tests pass.
- application-analysis route entry submits a planner-recognizable user message; blank durable Runs and system-only Anthropic requests fail locally without provider dispatch.

## Assumptions and open questions

All blocking assumptions are recorded as verified in `contract.json`. There are no open blocking product questions. TASK-002 has a required human gate for the shared-module write.
