# HR Agent Trace Optimization SDD

## 1. Existing Architecture Summary

### 1.1 Frontend Entry and Container

The HR AI chat page uses `hr-frontend/src/views/hr/AIChatView.vue`. Its header is rendered through `hr-frontend/src/components/chat/ConversationHeader.vue`, which emits `show-trace` from an `el-button` labeled "执行轨迹". `AIChatView.vue` owns `tracePanelVisible` and renders:

```vue
<AgentTracePanel
  v-model:visible="tracePanelVisible"
  :session-id="currentSession?.id ?? null"
/>
```

### 1.2 Current Trace Panel

`hr-frontend/src/components/AgentTracePanel.vue` is a single-file Vue component that:

- imports `getAgentRuns` and `getToolTraces` from `hr-frontend/src/api/ai.ts`;
- stores durable runs and legacy traces in local refs;
- loads both datasets when the drawer opens or the session changes;
- translates known agent, runtime, intent, tool, data, risk, output, decision, status, step, and MCP policy keys into Chinese labels;
- parses plan JSON and step output JSON inside helper functions;
- renders a fixed-width `el-drawer` at `560px`;
- renders durable runs first and legacy traces afterward;
- renders final answer markdown with MarkdownIt and DOMPurify sanitization;
- formats JSON with `JSON.parse` and `JSON.stringify` where possible;
- collapses long input/output previews with local expanded `Set`s.

### 1.3 Current Types and APIs

`hr-frontend/src/types/ai.ts` defines:

- `ToolTraceItem`;
- `AgentRunStepItem`;
- `AgentRunRecruitingPlan`;
- `AgentRunDecision`;
- `AgentRunPlanJSON`;
- `AgentRunItem`.

`hr-frontend/src/api/ai.ts` exposes:

- `getToolTraces(sessionId)`;
- `getAgentRuns(sessionId)`.

`hr-frontend/src/api/agentRun.ts` separately exposes active durable run lifecycle helpers:

- `getActiveAgentRun(sessionId)`;
- `subscribeAgentRunEvents(runId, afterSeq, handlers, options)`;
- `getAgentRun(runId)`;
- `cancelAgentRun(runId)`;
- `confirmAgentRun(runId)`.

The composable `hr-frontend/src/composables/useHrAgentRun.ts` already manages durable run subscription, reconnect, hydrate, cancel, confirm, and settlement semantics for chat execution.

### 1.4 Backend Support

The web gateway routes include:

- `GET /api/v1/hr/ai/sessions/:session_id/tool-traces`;
- `GET /api/v1/hr/ai/sessions/:session_id/agent-runs`;
- `GET /api/v1/hr/ai/sessions/:session_id/active-run`;
- `GET /api/v1/hr/ai/runs/:run_id/events`.

`logic-grpc-service/service/ai_service.go` verifies session ownership and returns:

- legacy tool traces capped at 500 records;
- durable agent runs capped at 100 records;
- desensitized arguments and result content for tool traces and run steps.

## 2. Problem Analysis

- The panel has useful data but lacks a clear first-read summary.
- Durable run details, final answer, step timeline, raw JSON, and legacy trace details are displayed in one linear scroll path.
- There is no filtering, searching, or quick isolation of failures and risk signals.
- Real-time durable run APIs exist, but the panel only reloads on open/session changes.
- JSON display repeats parse/format work and is difficult to scan for large objects.
- Failure, risk, and MCP policy issues are visible only where they occur, not aggregated.
- The fixed drawer width is too narrow for trace-heavy JSON and too rigid for responsive use.
- Template expressions repeatedly call parsing helpers such as `runPlan(run)`, `recruitingPlan(run)`, `policyDecisionFromJson(...)`, and `formatJson(...)`.
- Backend caps are hard-coded and no UI behavior explains or lazily handles long histories.
- Test coverage is focused on adjacent AI run flow and reducer behavior, not this trace panel's derived display logic.

## 3. Proposed Design

### 3.1 Component Decomposition

Refactor the current single-component logic into small, local units while preserving the public `AgentTracePanel` props/events:

- `AgentTracePanel.vue`: drawer shell, data loading, active subscription lifecycle, high-level layout.
- `agentTraceViewModel.ts`: pure derivation helpers for runs, steps, legacy traces, overview metrics, abnormal summaries, filters, JSON preview state, and policy extraction.
- Optional local subcomponents under `hr-frontend/src/components/agent-trace/`:
  - `TraceOverview.vue`;
  - `TraceIssueSummary.vue`;
  - `TraceFilterBar.vue`;
  - `TraceRunSection.vue`;
  - `TraceJsonBlock.vue`;
  - `TraceLegacySection.vue`.

The decomposition should stay inside HR frontend component boundaries and avoid shared module changes unless explicitly scoped.

### 3.2 View Model

Create a normalized view model from loaded `runs` and `traces`:

- `TraceRunVM`
  - original run reference;
  - parsed plan;
  - parsed recruiting plan;
  - selected skills/memories;
  - decision entries;
  - risk flags;
  - status tag type;
  - derived duration;
  - step VMs;
  - issue summaries.
- `TraceStepVM`
  - original step reference;
  - title;
  - localized type/status;
  - status category;
  - parsed policy decision;
  - evidence summary;
  - normalized searchable text;
  - formatted input/output display metadata.
- `TraceLegacyVM`
  - original trace reference;
  - status category;
  - parsed policy decision;
  - normalized searchable text;
  - formatted args/result display metadata.
- `TraceOverviewVM`
  - latest run status;
  - model name;
  - intent;
  - runtime;
  - run count;
  - step count;
  - tool count;
  - failure count;
  - warning/risk count;
  - policy issue count;
  - active subscription state.

The view model must be computed from raw data and filter state. JSON parsing and formatting should happen in the derivation layer rather than repeatedly in the template.

### 3.3 Layout

Use a wider responsive drawer:

- desktop: `min(860px, 92vw)` or equivalent;
- mobile: near-full-width or full-screen using Element Plus drawer size responsive binding/CSS.

Inside the drawer:

1. Header zone: overview metrics and refresh/retry/live state.
2. Abnormal summary: failures, warnings, MCP policy issues, risk flags, confirmation requirements.
3. Filter/search toolbar.
4. Layered content:
   - Overview/Plan;
   - Steps;
   - Raw Data;
   - Legacy Traces.

Tabs or segmented controls are both acceptable. If tabs are used, abnormal summary should remain visible above tabs.

### 3.4 Filtering

Maintain local filter state:

- `keyword`;
- `statusFilter`: `all | failed | warning | succeeded | active`;
- `typeFilter`: `all | plan | tool | evidence | memory | prompt | model | fallback | legacy`;
- `issueOnly`: boolean.

Filtering should be applied to the view model and should not mutate raw data. A reset action clears all filters.

### 3.5 Real-Time Integration

Initial real-time design should reuse existing API helpers:

1. On panel open with `sessionId`, load durable run and legacy trace data as today.
2. Query `getActiveAgentRun(sessionId, { silentError: true })`.
3. If an active run exists and is not already represented as terminal, subscribe through `subscribeAgentRunEvents(runId, lastEventSeq)`.
4. Use incoming events to update lightweight live state:
   - status;
   - last event sequence;
   - process text or active event line;
   - tool started/finished indicators where available.
5. On terminal event or stream close, refresh durable `getAgentRuns` to align the panel with persisted run/step data.
6. On drawer close/unmount, abort only the subscription fetch.

This design intentionally does not call `cancelAgentRun` from the trace panel.

### 3.6 JSON Viewer

Implement a lightweight local `TraceJsonBlock` without new dependencies:

- accepts `content`, `label`, `copyLabel`, and collapsed size;
- attempts JSON parse and pretty format;
- stores parse status;
- renders valid JSON in a `pre` block with stable wrapping;
- renders invalid JSON as raw text;
- supports "展开全部/收起";
- supports "复制" using the Clipboard API with fallback error message;
- uses desensitized content exactly as received from backend.

Syntax highlighting is optional and should not introduce dependency risk.

### 3.7 Abnormal Summary

Issue derivation should classify:

- run errors as `error`;
- failed steps as `error`;
- partial/fallback/canceled as `warning` unless canceled is shown as neutral info by product decision;
- MCP deny/invalid_args as `error`;
- MCP confirmation_required/rate_limited as `warning`;
- risk flags and decision warning booleans as `warning`;
- unavailable tool risk and human confirmation requirement as `warning`.

Each issue item should include label, severity, source run/step/trace id, and optional anchor key. If scroll-to-anchor is implemented, it must gracefully no-op if the element is not mounted because of filtering or inactive tab.

## 4. Data Structure Changes

### 4.1 Frontend-Only Types

Add local frontend view-model interfaces. Suggested location:

```text
hr-frontend/src/components/agent-trace/agentTraceViewModel.ts
```

or, if the project prefers utility placement:

```text
hr-frontend/src/utils/agentTraceViewModel.ts
```

These are internal UI types and should not change backend contracts.

### 4.2 Existing API Types

No required change to `hr-frontend/src/types/ai.ts` for the first frontend-only implementation. If pagination or additional backend metadata is later introduced, type extensions must be explicit and backward-compatible.

### 4.3 Backend Data

No database schema changes. Existing backend data remains authoritative.

## 5. API and Interface Changes

### 5.1 Required First Pass

No required public API change. Use existing:

- `getAgentRuns`;
- `getToolTraces`;
- `getActiveAgentRun`;
- `subscribeAgentRunEvents`.

### 5.2 Optional Later API Extension

Pagination may be added later as backward-compatible optional query parameters:

- `page`;
- `page_size`;
- `cursor`;
- `status`;
- `type`;
- response `total` or `next_cursor`.

Because this affects public API/protobuf/gateway behavior, it is a Hard Stop for implementation unless the task explicitly scopes the change and receives confirmation.

## 6. Algorithm or Workflow Changes

### 6.1 Loading Workflow

Current:

```text
drawer opens -> Promise.all(getAgentRuns, getToolTraces) -> render raw data
```

Proposed:

```text
drawer opens
  -> load durable runs and legacy traces
  -> build view model
  -> fetch active run silently
  -> subscribe if active
  -> update live state from events
  -> refresh durable runs on terminal event
```

### 6.2 Derivation Workflow

Raw data should flow through pure helper functions:

```text
AgentRunItem[] + ToolTraceItem[] + live state
  -> parsed/cached run VMs
  -> overview VM
  -> issue summary VM
  -> filtered visible VM
```

Helper functions must treat parse failures as display states, not thrown exceptions.

### 6.3 Filter Workflow

```text
filter state changes
  -> recompute visible VMs from cached searchable text
  -> show content or filter-empty state
```

### 6.4 Copy Workflow

```text
user clicks copy
  -> copy formatted full desensitized content
  -> show success/failure feedback
```

Copy must not request raw unmasked data from backend.

## 7. Configuration Design

- No environment variables are required.
- No package manifest changes are required.
- UI constants such as drawer width, preview length, and collapse thresholds should live in the trace panel module or view-model file.
- Debug logging should continue to use `debugLog.trace`.

## 8. Compatibility Strategy

- Preserve `AgentTracePanel` public props and `update:visible` event.
- Keep legacy trace rendering available.
- Keep final answer markdown sanitization.
- Keep `getAgentRuns` fallback behavior where durable run load failure does not necessarily prevent legacy trace display.
- Avoid changes to `useHrAgentRun` unless a later task explicitly scopes shared behavior changes.
- Use optional real-time status as additive UI state; persisted durable run data remains the source of truth for historical steps.
- Any backend pagination should default to current behavior when query params are absent.

## 9. Error Handling and Fallback Design

- Loading with no `sessionId`: clear data and render empty state.
- Durable run request failure: show legacy traces if available, with non-blocking warning.
- Legacy trace request failure: show durable runs if available, with non-blocking warning.
- Both requests fail: show error state with retry.
- JSON parse failure: render raw content and mark as plain text.
- Clipboard failure: show Element Plus warning/error without crashing.
- Active-run request failure: do not block historical trace display.
- SSE subscription failure: keep current loaded data, show live-state warning, allow manual refresh.
- Drawer close/unmount: abort subscription only.

## 10. Observability and Debug Output Design

- Keep existing `loadTraces_started`, `loadTraces_finished`, and `loadTraces_failed`.
- Add scoped debug events only where useful:
  - `activeRun_check_started`;
  - `activeRun_check_finished`;
  - `traceSubscription_started`;
  - `traceSubscription_event`;
  - `traceSubscription_failed`;
  - `traceSubscription_closed`;
  - `traceFilter_changed` only if not too noisy, or omit user keystroke logs for privacy.
- Do not log raw JSON input/output content.
- Log counts and IDs, not sensitive payloads.

## 11. Testing Strategy

### 11.1 Unit Tests

Add focused tests for view-model helpers:

- overview count derivation;
- failed/warning/success classification;
- MCP policy issue classification;
- risk flag aggregation;
- keyword search matching;
- status/type filter behavior;
- valid JSON formatting;
- invalid JSON fallback;
- legacy-only compatibility;
- active/live state merge behavior if implemented.

### 11.2 Component Tests

Add or extend Vue Test Utils tests for:

- empty trace state;
- durable run with overview;
- failed step summary;
- filter-empty state;
- legacy trace section;
- JSON expand/collapse/copy button rendering;
- retry action rendering after load failure.

### 11.3 Required Commands

Implementation tasks should run:

```bash
pnpm --filter hr-frontend typecheck
pnpm --filter hr-frontend test
```

If a task only touches pure helper tests, targeted Vitest invocation is acceptable in addition to the full frontend test command when runtime is a concern.

## 12. Migration Risks

- Splitting a large component may introduce state synchronization bugs between raw data, view-model data, filters, and live subscription state.
- Real-time subscription may duplicate state already managed by `useHrAgentRun` if not carefully scoped.
- Making legacy traces secondary could reduce visibility for old sessions unless the empty/legacy-only path is tested.
- Optional pagination requires protobuf/gateway/frontend alignment and should be isolated into a separate confirmed task.
- Copy-to-clipboard behavior may behave differently under insecure origins or browser permission restrictions.
- Wider drawer and tabs may need visual verification across desktop/mobile layouts.

## 13. Implementation Boundaries

Recommended task boundaries after SPEC/SDD approval:

1. Frontend view-model helpers and tests.
2. Panel overview, abnormal summary, and responsive layout.
3. Search/filter and layered sections.
4. JSON block component and raw data tab/section.
5. Real-time active-run integration.
6. Optional backend pagination and API type extension, only after confirmation.

Implementation should initially stay within:

- `hr-frontend/src/components/AgentTracePanel.vue`;
- `hr-frontend/src/components/agent-trace/**` if subcomponents are created;
- `hr-frontend/src/utils/agentTraceViewModel.ts` or local equivalent;
- tests near the helper/component.

Do not modify these without explicit task scope:

- `package.json`;
- lockfiles;
- generated protobuf files;
- database schema;
- auth or permission middleware;
- global app config;
- shared agent run composable behavior used by chat execution.

## 14. Alternatives Considered

- Keep one component and only add styles: rejected because repeated parsing and dense template logic would remain hard to test and maintain.
- Add a third-party JSON viewer: rejected for first pass because dependencies require approval and a lightweight local viewer satisfies current requirements.
- Replace existing trace APIs with a new aggregate endpoint: rejected for first pass because current APIs already provide enough data for the top UX improvements.
- Build a separate full-screen trace route: rejected because the current user request targets the existing HR execution trace interface and the drawer entry already exists.
- Use `useHrAgentRun` directly inside the trace panel: possible, but risky if it couples passive observability with execution controls. Prefer a small read-only subscription path unless a later task proves reuse is safer.

## 15. Assumptions Requiring Confirmation

- ARC-001: Frontend-only improvements should be implemented before backend pagination.
- ARC-002: The execution trace panel is a diagnostic/observability surface and must not provide cancel/confirm controls unless separately requested.
- ARC-003: The first version can use local component state for filters and does not need persistent user preferences.
- ARC-004: Real-time display can be best-effort and may refresh persisted runs on completion rather than fully reconstructing every step from SSE events.
- ARC-005: Optional backend pagination is acceptable as a later task only if public API/protobuf scope is confirmed.

## 16. Open Questions

- OQ-001: Should the default selected layer be "概览" or "步骤" after opening the panel?
- OQ-002: Should older runs be collapsed by default when a session has multiple runs?
- OQ-003: Should live state appear in the header button badge while the panel is closed?
- OQ-004: Should support/developer-only raw JSON affordances be hidden for ordinary HR users, or is the current HR AI permission enough?
- OQ-005: Should optional pagination be included in the same feature harness after `prepare-harness`, or split into a backend-focused feature?
