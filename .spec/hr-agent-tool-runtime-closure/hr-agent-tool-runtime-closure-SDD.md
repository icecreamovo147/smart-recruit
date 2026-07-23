# HR Agent Tool Runtime Closure — SDD

## Existing architecture and problem analysis

`native_servers.go` loads the default HR Agent, Prompt, Agent Skills, pre-context MCP/application tools, and model-native builtin tool schemas. `hr_tools.Executor` delegates Recruitment-owned reads to gRPC clients. `smart-recruit-commons/ai.Client` owns the model/tool loop. Durable Runs, Run Steps, and Tool Traces are persisted by the native store.

The current path conflates missing configuration with permission to use defaults; treats absent MCP selection as an unrestricted set; represents business failures as successful Tool content; and only deterministically pre-executes live tools for complete-only providers. Governance version evidence is incomplete and streaming model deltas are categorized as process events.

## Design

### FLOW-001 — Governance resolution

Resolve an explicit runtime allowlist from the selected Agent. Preserve a narrow application-snapshot fallback only when no persisted Agent exists and the request has an application context. Existing Agents with no enabled concrete binding produce an empty allowlist. MCP selection uses set intersection with explicit request keys; empty keys produce no calls.

### FLOW-002 — Tool result contract

Executor argument, authorization, unsupported, not-found, and downstream failures return classified non-nil errors. Callbacks persist `error`; only `success` rows with structurally non-error content are useful. User-facing errors remain bounded and do not disclose internal payloads.

### FLOW-003 — Required live-data execution

Run the deterministic `RecruitingPlanner` before provider dispatch. Extend it with explicit application-listing and candidate-lookup intents and a deterministic required-evidence-group representation. For required live-data intents, select the minimal authorized read Tool for each group and pre-execute it before both model-native and complete-only provider paths. Candidate comparison requires both job and job-application/candidate groups; interview preparation and offer support require both candidate/application detail and job detail; analytics requires each metric family explicitly requested by the query. The provider receives successful results as Tool/context messages. If any required group is unsatisfied, short-circuit to a clarification, deterministic limitation, empty state, or error output as applicable. A provider free-text answer is accepted only after all groups pass. Proposal/action tools are never auto-executed.

### FLOW-004 — Iteration control

Extend the internal shared AI client with a per-call tool-loop options value containing `MaxRounds`. Preserve existing methods as compatibility wrappers. `NativeStore.ChatWithRecruitingTools` maps the validated Agent `max_iterations` to that option. Clamp to `[1, system ceiling]`; absent values retain the current configured default.

### FLOW-005 — Prompt and Skill compilation

Prompt binding validation is shared by create/update and runtime. Render only allowlisted variables with stable string values. Detect unresolved template expressions before model invocation. Agent Skill selection continues to use policy ranking, but enrichment requires an exact valid current version and records its ID. Skill body and flow compilation remains instruction-only and cannot expand the Tool allowlist.

### FLOW-006 — Durable governance and events

Load effective Agent identity when creating the Run and populate `AgentID`, type, and name. Store Prompt version and Skill version metadata in privacy-safe governance/process evidence. Map `generating` events carrying text delta to `assistant.delta`; persist final answer as state, not a second synthetic full delta.

### FLOW-007 — Bounded aggregation

Keep existing Recruitment RPCs. Process at most 100 jobs, 10 pages per job with the existing 100-row page size, 5,000 aggregate application rows, and four concurrent job fetches. Stop promptly on context cancellation. Sort by job ID then application ID before pagination/formatting. Partial job failures return successful rows with `partial=true`, `failed_job_count`, and bounded safe warnings; final facts use only those rows. If all attempted jobs fail, return a non-nil aggregate error and no useful result.

### FLOW-008 — Application-analysis message validity

`CreateApplicationAnalysisSession` persists and returns a canonical user message requesting resume-to-job match analysis. The HR frontend uses the returned message and has a deterministic named fallback for compatibility, shared by route-entry and in-chat candidate analysis. The gateway trims and rejects blank durable Run messages, and the native AI service repeats this validation before governance loading or Run creation. The Anthropic adapter separates System content from conversational messages, then returns a local validation error when no non-System message remains; initializing an empty JSON array alone is insufficient because the provider requires a meaningful conversation turn.

## Data, API, and configuration impact

- Database/schema: none.
- Proto/public HTTP/gRPC: none.
- Dependencies/lockfiles: none.
- Shared internal Go API: an additive per-call tool-loop option in `smart-recruit-commons/ai`, gated by user confirmation.
- Frontend API payloads: no new public endpoint; selector filtering uses existing Prompt fields.

## Observability and privacy

Persist effective IDs/versions and statuses, never Prompt/Skill bodies in Run metadata. Tool failures are error Trace/Step rows. No raw candidate resume or unredacted MCP argument is added to logs.

## Test strategy

- Unit: allowlist resolution, typed Tool errors, planner gates, Prompt rendering/validation, current Skill version, aggregation bounds/order.
- Integration: fake model-native provider skipping Tool; disabled Tool; empty/error/real job inventory; MCP empty selection; durable Run/Trace/Step/event persistence.
- Shared AI: per-call max-round override and compatibility wrapper.
- Frontend: Prompt selector filters active compatible system templates and typecheck.
- Cumulative: full AI Agent service tests, shared AI tests, HR typecheck/tests, Harness validators, knowledge validators.
- Regression: analysis-session message seeding, route-entry payload construction, blank HTTP/gRPC Run rejection, and Anthropic system-only request rejection before HTTP dispatch.

## Risk and boundaries

Fail-closed changes can reveal previously hidden misconfiguration; explicit deterministic error messages are required. Pre-execution applies only to read-only data tools selected by planner intent. Shared module changes stop at the TASK-002 human gate. Any discovered need for Proto, schema, dependency, global config, or wider security-policy change is a `contract_gap`/L2 amendment, not an in-task improvisation.

## Alternatives rejected

- Prompt-only “do not fabricate”: cannot enforce truth.
- Default Tool fallback for configured Agents: violates admin governance.
- Execute every MCP binding: unsafe and side-effect prone.
- Add HR-wide application RPC now: public contract expansion is outside this repair.
