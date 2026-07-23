# HR Agent Trace Optimization SPEC

## 1. Background

HR AI chat exposes an "执行轨迹" entry in `hr-frontend/src/components/chat/ConversationHeader.vue` that opens `AgentTracePanel.vue` inside `AIChatView.vue`. The panel currently loads session-level agent runs and legacy tool traces through `getAgentRuns(sessionId)` and `getToolTraces(sessionId)`, then renders a linear drawer containing run summaries, final answers, step timelines, raw input/output previews, MCP policy decisions, and legacy trace details.

The current implementation is useful for engineering diagnostics, but the information hierarchy is dense for HR users and operational users. The previous analysis identified ten optimization areas:

1. Add a top-level overview.
2. Split information into clearer layers.
3. Add filtering and search.
4. Support real-time execution status.
5. Improve JSON display.
6. Emphasize failures, risks, and policy decisions.
7. Improve layout and entry affordance.
8. Reduce rendering and parsing cost.
9. Add pagination or lazy loading for long histories.
10. Add test coverage.

## 2. Goals

- Make the execution trace panel understandable at a glance for HR and operations users while keeping diagnostic detail available for developers.
- Preserve all existing trace information, including durable agent runs and legacy tool traces.
- Improve navigation for long sessions through search, filters, collapsible sections, and clearer visual hierarchy.
- Surface execution health, risk, MCP policy decisions, and failure information before raw timeline details.
- Reuse existing HR frontend architecture, Element Plus components, Vue 3 Composition API patterns, and existing API helpers.
- Define a path for optional real-time and pagination enhancements without requiring a breaking API change in the first implementation task.
- Ensure future implementation is testable with focused frontend unit tests and type checks.

## 3. Non-Goals

- Do not redesign the entire HR AI chat page.
- Do not change AI run execution semantics, tool execution behavior, MCP policy behavior, authorization rules, or quota behavior.
- Do not change database schemas.
- Do not add new third-party dependencies unless separately approved.
- Do not expose sensitive raw data that backend desensitization already masks.
- Do not replace the existing durable run lifecycle implementation.
- Do not remove legacy tool trace compatibility until a separate migration decision is made.

## 4. User-Facing Behavior

### 4.1 Panel Entry

- The chat header keeps an "执行轨迹" entry.
- The entry should provide a stronger affordance than plain text, such as an icon and optional status badge when trace data indicates failures, warnings, or active execution.
- Opening the panel must still work for sessions with or without an active application binding.

### 4.2 Top Overview

When trace data exists, the top of the panel must show an overview before timeline details. The overview must include, when derivable from current data:

- latest or selected run status;
- model name;
- intent;
- runtime or agent type;
- total run count;
- step count;
- tool step count;
- failure count;
- risk count;
- MCP policy warning or denial count;
- total or current run duration;
- whether manual confirmation is required.

If a value cannot be derived, the UI must show a neutral "未记录" or omit the metric rather than inventing data.

### 4.3 Information Layers

The panel must separate at least these layers:

- overview and abnormal summary;
- structured run plan and final answer;
- execution steps;
- raw input/output data;
- legacy tool trace details.

The separation may use tabs, segmented controls, collapsible sections, or a similarly clear Element Plus pattern. Legacy trace details should be visually secondary and collapsed by default when durable agent runs exist.

### 4.4 Search and Filtering

The panel must allow users to reduce visible trace noise through:

- keyword search across run title, model name, step title, tool name, error text, policy reason, input preview, and output preview;
- status filters for all, failed, warnings, succeeded, and active/running states;
- type filters for plan, tool, evidence, memory, prompt/skill, model, fallback/recovery, and legacy trace;
- a quick way to focus failures and policy/risk issues.

Filtering must not mutate source data. Empty filtered results must show a clear empty state distinct from "no trace exists".

### 4.5 Real-Time Status

The panel should support active execution visibility using existing durable run APIs where possible:

- `GET /api/v1/hr/ai/sessions/:session_id/active-run`;
- `GET /api/v1/hr/ai/runs/:run_id/events` through the existing frontend `subscribeAgentRunEvents` helper.

Real-time support must be non-destructive. Closing the panel or aborting a subscription must not cancel a backend run. If real-time subscription fails, the panel must retain the last loaded durable data and present a recoverable warning or refresh action.

### 4.6 JSON and Raw Data Display

Raw input/output must be easier to inspect than the current plain text preview:

- JSON should be formatted consistently.
- Invalid JSON or non-JSON strings must remain visible.
- Long values must be collapsed by default with clear expand/collapse controls.
- Users should be able to copy visible or full desensitized content.
- The UI should distinguish "business summary" from "raw data".
- Sensitive fields already masked by backend desensitization must remain masked in displayed and copied content.

### 4.7 Failure, Risk, and Policy Emphasis

The panel must aggregate and highlight:

- run-level errors;
- failed steps;
- partial or fallback outcomes;
- risk flags and risk checks;
- MCP policy decisions including deny, invalid args, confirmation required, and rate limited;
- unavailable tool risks and human confirmation requirements from decision entries.

The abnormal summary must link or scroll to the relevant run or step when technically feasible.

### 4.8 Layout Responsiveness

- Desktop panel width should be wider than the current fixed `560px` when viewport space allows.
- Mobile viewport behavior must avoid horizontal clipping and should use a near-full-width or full-screen drawer.
- Text, tags, JSON, and action controls must not overlap.
- Dense trace information should remain scannable without nested card-heavy layouts.

### 4.9 Performance

- Expensive JSON parsing, formatting, policy parsing, and derived counters must be computed once per loaded dataset or cached through computed view models.
- Template expressions should avoid repeatedly parsing the same run or step JSON.
- Large datasets must remain usable with collapse defaults and optional lazy rendering or pagination.

### 4.10 Pagination and Long History

The feature should define a path for long histories:

- frontend-side lazy rendering or section-level collapse is acceptable without backend API changes;
- backend pagination for `agent-runs` and `tool-traces` may be introduced only as a backward-compatible extension with default behavior preserved;
- any protobuf, gateway, or public API changes require explicit task scope and confirmation.

## 5. Functional Requirements

- FR-001: The panel must continue to load durable agent runs and legacy tool traces for the selected HR-owned session.
- FR-002: The panel must show an overview before detailed trace content when runs or traces exist.
- FR-003: Overview metrics must be derived only from returned run, step, plan, decision, and trace fields.
- FR-004: The panel must provide a clear abnormal summary for failures, partial/fallback states, risk flags, and MCP policy warnings or denials.
- FR-005: The panel must provide search and filter controls without changing source data.
- FR-006: The panel must separate structured overview, execution steps, raw data, and legacy traces into distinct navigational layers.
- FR-007: Legacy traces must remain accessible and must not be removed.
- FR-008: Raw input and output display must support formatted JSON, invalid JSON fallback display, collapse/expand, and copy behavior.
- FR-009: The panel must preserve markdown rendering and sanitization behavior for final answers.
- FR-010: The panel should use active-run and run event APIs to represent active execution when available and feasible within task scope.
- FR-011: Real-time subscription abort must not call cancel APIs or cancel backend execution.
- FR-012: Failed data refresh must clear stale data only when it would otherwise misrepresent another session, and must provide a retry path.
- FR-013: The entry button should provide a clearer visual indication of trace availability or severity when the data needed to determine that state is already available.
- FR-014: Implementation must keep existing auth and permission behavior unchanged.
- FR-015: Implementation must include focused tests for derived view-model logic and critical UI states.

## 6. Non-Functional Requirements

- NFR-001: The UI must follow existing HR frontend Vue 3, TypeScript, and Element Plus patterns.
- NFR-002: No new dependency may be added without explicit approval.
- NFR-003: The panel must remain responsive on desktop and mobile widths.
- NFR-004: Long text and JSON must not overflow, overlap, or force unreadable layout shifts.
- NFR-005: Derived data computation must avoid repeated JSON parsing in templates.
- NFR-006: TypeScript must avoid casual `any`; use existing types or narrowly defined interfaces.
- NFR-007: The implementation must pass `pnpm --filter hr-frontend typecheck`.
- NFR-008: Targeted frontend tests must run through the existing Vitest setup.

## 7. Compatibility Requirements

- CR-001: Existing endpoints `/api/v1/hr/ai/sessions/:session_id/agent-runs` and `/api/v1/hr/ai/sessions/:session_id/tool-traces` must continue to work for current callers.
- CR-002: Existing response shapes in `hr-frontend/src/types/ai.ts` must remain supported unless a task explicitly scopes and confirms a type/API extension.
- CR-003: The existing durable run event helper in `hr-frontend/src/api/agentRun.ts` must not be changed in a way that breaks `useHrAgentRun`.
- CR-004: The existing final answer markdown sanitization must not be weakened.
- CR-005: Legacy trace display must remain accessible for sessions without durable agent runs.
- CR-006: Backend desensitization in `logic-grpc-service/service/ai_service.go` must remain authoritative for sensitive fields.
- CR-007: No database migration is allowed for this feature.

## 8. Observability and Debug Requirements

- ODR-001: Existing `debugLog.trace` logging for load start, success, and failure should remain.
- ODR-002: Real-time subscription states should be observable through `debugLog.trace` or existing user-visible status where useful.
- ODR-003: Refresh and retry actions should not spam error messages during expected aborts or panel close events.
- ODR-004: Tests should cover derived abnormal counts and active/failed/empty states so regressions are visible.

## 9. Error Handling and Fallback Requirements

- EFR-001: If `getAgentRuns` fails but `getToolTraces` succeeds, the panel may continue showing legacy trace data with a non-blocking warning, matching the current compatibility intent.
- EFR-002: If all trace loading fails, the panel must show a clear error and retry affordance.
- EFR-003: Invalid JSON must be displayed as raw text rather than causing rendering failure.
- EFR-004: Missing optional fields must render as neutral unknown values, not as crashes or misleading success states.
- EFR-005: Real-time event subscription failure must not erase already loaded durable data.
- EFR-006: Filtering to zero results must show a filter-empty state with a reset option.

## 10. Security and Safety Requirements

- SSR-001: Do not bypass backend session ownership checks.
- SSR-002: Do not add client-side behavior that exposes unmasked sensitive data.
- SSR-003: Copied raw content must use the same desensitized content shown in the UI.
- SSR-004: Markdown rendering must remain sanitized through the existing DOMPurify approach or an equally strict local equivalent.
- SSR-005: No auth, authorization, permission, or quota logic changes are allowed without separate confirmation.
- SSR-006: No database schema or migration changes are allowed.

## 11. Acceptance Criteria

- AC-001: Opening execution trace for a session with durable runs shows a top overview with status, model, intent, run/step/tool counts, and abnormal counts.
- AC-002: A run containing failed steps or policy denial displays those issues in an abnormal summary before the detailed timeline.
- AC-003: Search filters visible steps/traces by keyword and can be reset to show all data.
- AC-004: Status/type filters can isolate failed steps, evidence steps, tool steps, and legacy traces.
- AC-005: Structured run information, execution steps, raw JSON, and legacy traces are separated into clear sections or tabs.
- AC-006: Legacy-only sessions still show legacy trace details.
- AC-007: Raw JSON is formatted when valid, falls back to raw text when invalid, supports expand/collapse, and supports copying desensitized content.
- AC-008: Closing a panel with active subscription does not cancel the backend run.
- AC-009: Subscription failure preserves already loaded data and shows a recoverable warning or refresh action.
- AC-010: The drawer layout is responsive and avoids text/control overlap at desktop and mobile widths.
- AC-011: View-model or helper tests cover overview counts, filtering, JSON formatting fallback, abnormal detection, and legacy compatibility.
- AC-012: `pnpm --filter hr-frontend typecheck` passes after implementation tasks.

## 12. Out of Scope

- Changing AI execution, planning, tool, MCP, quota, or auth semantics.
- Removing legacy trace APIs or database tables.
- Introducing a new design system.
- Adding a third-party JSON viewer package without approval.
- Changing generated protobuf files unless a later confirmed task explicitly scopes a backward-compatible API extension.
- Creating a new HR left-menu page for execution trace.
- Persisting user-specific trace panel filter preferences.

## 13. Assumptions Requiring Confirmation

- ARC-001: The feature name `hr-agent-trace-optimization` is acceptable.
- ARC-002: The first implementation pass should prioritize frontend-only improvements before optional backend pagination.
- ARC-003: A custom lightweight JSON viewer built from existing Vue/Element Plus primitives is preferred over adding a dependency.
- ARC-004: Optional backward-compatible query parameters for pagination are acceptable only after explicit confirmation in a later task.
- ARC-005: Real-time panel behavior can reuse existing durable run event APIs and does not need a new streaming endpoint.

## 14. Open Questions

- OQ-001: Should the panel show all runs by default or default to the latest run with older runs collapsed?
- OQ-002: Should active execution status appear only inside the open panel, or should the header entry badge update while the panel is closed?
- OQ-003: Which user role is the primary target for raw JSON inspection: HR operator, recruiting admin, or developer/support?
- OQ-004: Should copy actions include full raw JSON by default or only visible/collapsed-safe content?
- OQ-005: Should backend pagination be part of this feature's first harness package or deferred into a separate feature after frontend UX is complete?
