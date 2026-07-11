# TASKS - hr-agent-trace-optimization

Execute tasks in order. Each TASK must pass its acceptance file and Harness checks before the next TASK starts. Any TASK marked `requiresHumanConfirmation: true` must receive explicit user confirmation before implementation begins.

## Task Overview

| TASK | Title | Status | Scope | Acceptance |
|------|-------|--------|-------|------------|
| TASK-HATO-001 | Trace View Model And Helper Tests | pending | frontend pure derivation helpers | acceptance/TASK-HATO-001.md |
| TASK-HATO-002 | Overview, Issue Summary, And Responsive Shell | pending | trace panel overview and layout | acceptance/TASK-HATO-002.md |
| TASK-HATO-003 | Search, Filters, And Layered Navigation | pending | trace panel filtering and sections | acceptance/TASK-HATO-003.md |
| TASK-HATO-004 | JSON Raw Data Viewer | pending | local JSON display/copy component | acceptance/TASK-HATO-004.md |
| TASK-HATO-005 | Read-Only Real-Time Trace Status | pending | active-run and SSE observability | acceptance/TASK-HATO-005.md |
| TASK-HATO-006 | Optional Paginated Trace Retrieval | pending | backward-compatible trace API extension | acceptance/TASK-HATO-006.md |

## TASK-HATO-001 - Trace View Model And Helper Tests

### Goal

Create tested pure frontend helpers that normalize durable agent runs and legacy traces into overview metrics, issue summaries, searchable text, status/type classifications, policy decisions, and JSON display metadata.

### Scope

- Add a local trace view-model helper module under the HR frontend.
- Add focused Vitest tests for helper behavior.
- Do not change the visible panel UI in this task except where required to export or consume no-op helper types.

### Allowed Files

- `hr-frontend/src/components/agent-trace/agentTraceViewModel.ts`
- `hr-frontend/src/components/agent-trace/agentTraceViewModel.test.ts`
- `.spec/hr-agent-trace-optimization/reports/TASK-HATO-001-report.md`
- `.spec/hr-agent-trace-optimization/reports/TASK-HATO-001-evidence.json`

### Forbidden Files

- `package.json`
- `pnpm-lock.yaml`
- `*/package.json`
- `*/pnpm-lock.yaml`
- `**/*.proto`
- `db.sql`
- `logic-grpc-service/**`
- `web-gin-service/**`
- `user-frontend/**`
- `interviewer-frontend/**`

### Dependencies

- SPEC and SDD must exist.
- No prior TASK dependency.

### Acceptance Criteria

- View-model helpers derive overview metrics from durable runs and legacy traces.
- Helpers classify failed, warning, success, active, evidence, tool, and legacy items.
- Helpers parse MCP policy decisions and risk flags.
- Helpers produce searchable text without mutating source data.
- Helpers format valid JSON and preserve invalid JSON as raw text.
- Tests cover durable-run, failed-step, policy-warning, invalid-JSON, and legacy-only scenarios.

### Required Tests

- `pnpm --filter hr-frontend typecheck`
- `pnpm --filter hr-frontend test`

### Risks

- Label or status logic can diverge from the existing panel if helpers are not aligned with current mappings.

### Notes

- Keep this task pure and reusable so later UI tasks can consume the same derivation layer.

## TASK-HATO-002 - Overview, Issue Summary, And Responsive Shell

### Goal

Update the execution trace drawer to show a top overview, abnormal summary, and responsive shell before detailed timeline content.

### Scope

- Consume TASK-HATO-001 view-model helpers in `AgentTracePanel.vue`.
- Add overview and issue summary subcomponents if useful.
- Widen and responsive-tune the drawer.
- Preserve current run, final answer, step, and legacy trace display while adding the new summary layer.

### Allowed Files

- `hr-frontend/src/components/AgentTracePanel.vue`
- `hr-frontend/src/components/agent-trace/agentTraceViewModel.ts`
- `hr-frontend/src/components/agent-trace/agentTraceViewModel.test.ts`
- `hr-frontend/src/components/agent-trace/TraceOverview.vue`
- `hr-frontend/src/components/agent-trace/TraceIssueSummary.vue`
- `hr-frontend/src/components/AgentTracePanel.test.ts`
- `.spec/hr-agent-trace-optimization/reports/TASK-HATO-002-report.md`
- `.spec/hr-agent-trace-optimization/reports/TASK-HATO-002-evidence.json`

### Forbidden Files

- `package.json`
- `pnpm-lock.yaml`
- `*/package.json`
- `*/pnpm-lock.yaml`
- `**/*.proto`
- `db.sql`
- `logic-grpc-service/**`
- `web-gin-service/**`
- `user-frontend/**`
- `interviewer-frontend/**`

### Dependencies

- TASK-HATO-001 must be complete.

### Acceptance Criteria

- Sessions with trace data show overview metrics before timeline details.
- Failed runs, failed steps, policy issues, and risk flags appear in an abnormal summary.
- Existing final answer markdown sanitization remains intact.
- Legacy-only sessions still show trace data.
- Drawer width is responsive and avoids obvious clipping on desktop and mobile widths.

### Required Tests

- `pnpm --filter hr-frontend typecheck`
- `pnpm --filter hr-frontend test`

### Risks

- The existing large component may become harder to maintain if summary UI is added without removing duplicated parsing logic.

### Notes

- Do not add search/filter/layer navigation in this task beyond what is needed for summary rendering.

## TASK-HATO-003 - Search, Filters, And Layered Navigation

### Goal

Add keyword search, status/type filters, issue-only focus, and clear section or tab navigation for overview, steps, raw data, and legacy traces.

### Scope

- Add trace filter state and UI controls.
- Add layered navigation using Element Plus components or existing local patterns.
- Add filtered empty state and reset behavior.
- Keep raw JSON display functionally equivalent until TASK-HATO-004.

### Allowed Files

- `hr-frontend/src/components/AgentTracePanel.vue`
- `hr-frontend/src/components/agent-trace/agentTraceViewModel.ts`
- `hr-frontend/src/components/agent-trace/agentTraceViewModel.test.ts`
- `hr-frontend/src/components/agent-trace/TraceFilterBar.vue`
- `hr-frontend/src/components/agent-trace/TraceRunSection.vue`
- `hr-frontend/src/components/agent-trace/TraceLegacySection.vue`
- `hr-frontend/src/components/agent-trace/TraceOverview.vue`
- `hr-frontend/src/components/agent-trace/TraceIssueSummary.vue`
- `hr-frontend/src/components/AgentTracePanel.test.ts`
- `.spec/hr-agent-trace-optimization/reports/TASK-HATO-003-report.md`
- `.spec/hr-agent-trace-optimization/reports/TASK-HATO-003-evidence.json`

### Forbidden Files

- `package.json`
- `pnpm-lock.yaml`
- `*/package.json`
- `*/pnpm-lock.yaml`
- `**/*.proto`
- `db.sql`
- `logic-grpc-service/**`
- `web-gin-service/**`
- `user-frontend/**`
- `interviewer-frontend/**`

### Dependencies

- TASK-HATO-001 must be complete.
- TASK-HATO-002 should be complete.

### Acceptance Criteria

- Keyword search matches run, step, trace, error, policy, input, and output text included in view-model searchable fields.
- Status filters can isolate failed, warning, succeeded, and active records.
- Type filters can isolate plan, tool, evidence, memory, prompt/skill, model, fallback/recovery, and legacy records where present.
- Issue-only mode focuses failures, risks, and policy issues.
- Filtering never mutates raw `runs` or `traces`.
- Filter-empty state differs from no-trace empty state and includes a reset path.
- Layered navigation separates overview/plan, steps, raw data, and legacy traces.

### Required Tests

- `pnpm --filter hr-frontend typecheck`
- `pnpm --filter hr-frontend test`

### Risks

- Filtering can hide the target of an issue-summary link if scroll-to-anchor is implemented too early.

### Notes

- If scroll-to-anchor is implemented, it must no-op safely when the target is hidden by filters or inactive navigation.

## TASK-HATO-004 - JSON Raw Data Viewer

### Goal

Replace ad hoc raw input/output previews with a local JSON/text viewer that formats valid JSON, preserves invalid content, supports expand/collapse, and copies desensitized content.

### Scope

- Add `TraceJsonBlock.vue` or equivalent local subcomponent.
- Use it for durable step input/output and legacy trace args/result content.
- Preserve existing backend-desensitized content exactly.
- Add tests for formatting, invalid JSON fallback, expand/collapse rendering, and copy affordance.

### Allowed Files

- `hr-frontend/src/components/AgentTracePanel.vue`
- `hr-frontend/src/components/agent-trace/TraceJsonBlock.vue`
- `hr-frontend/src/components/agent-trace/TraceJsonBlock.test.ts`
- `hr-frontend/src/components/agent-trace/agentTraceViewModel.ts`
- `hr-frontend/src/components/agent-trace/agentTraceViewModel.test.ts`
- `hr-frontend/src/components/agent-trace/TraceRunSection.vue`
- `hr-frontend/src/components/agent-trace/TraceLegacySection.vue`
- `hr-frontend/src/components/AgentTracePanel.test.ts`
- `.spec/hr-agent-trace-optimization/reports/TASK-HATO-004-report.md`
- `.spec/hr-agent-trace-optimization/reports/TASK-HATO-004-evidence.json`

### Forbidden Files

- `package.json`
- `pnpm-lock.yaml`
- `*/package.json`
- `*/pnpm-lock.yaml`
- `**/*.proto`
- `db.sql`
- `logic-grpc-service/**`
- `web-gin-service/**`
- `user-frontend/**`
- `interviewer-frontend/**`

### Dependencies

- TASK-HATO-001 must be complete.
- TASK-HATO-003 should be complete.

### Acceptance Criteria

- Valid JSON is pretty-formatted.
- Invalid JSON and non-JSON text remain visible.
- Long content is collapsed by default and can be expanded/collapsed.
- Copy action copies the full desensitized content available to the UI.
- Copy failure is handled without crashing.
- No new JSON viewer dependency is added.

### Required Tests

- `pnpm --filter hr-frontend typecheck`
- `pnpm --filter hr-frontend test`

### Risks

- Browser clipboard behavior may differ in test and local runtime; tests should mock clipboard behavior where needed.

### Notes

- Do not request or reconstruct unmasked sensitive data.

## TASK-HATO-005 - Read-Only Real-Time Trace Status

### Goal

Add read-only active execution visibility to the trace panel using existing active-run and SSE event APIs.

### Scope

- On panel open, check for active run via existing `getActiveAgentRun`.
- Subscribe via existing `subscribeAgentRunEvents` when there is an active run.
- Display live status and recoverable subscription warnings.
- Refresh persisted durable run data after terminal events when feasible.
- Abort subscriptions on close/unmount without canceling backend runs.

### Allowed Files

- `hr-frontend/src/components/AgentTracePanel.vue`
- `hr-frontend/src/components/AgentTracePanel.test.ts`
- `hr-frontend/src/components/agent-trace/agentTraceViewModel.ts`
- `hr-frontend/src/components/agent-trace/agentTraceViewModel.test.ts`
- `hr-frontend/src/components/agent-trace/TraceOverview.vue`
- `hr-frontend/src/components/agent-trace/TraceIssueSummary.vue`
- `hr-frontend/src/components/agent-trace/TraceRunSection.vue`
- `hr-frontend/src/components/agent-trace/TraceFilterBar.vue`
- `.spec/hr-agent-trace-optimization/reports/TASK-HATO-005-report.md`
- `.spec/hr-agent-trace-optimization/reports/TASK-HATO-005-evidence.json`

### Forbidden Files

- `package.json`
- `pnpm-lock.yaml`
- `*/package.json`
- `*/pnpm-lock.yaml`
- `**/*.proto`
- `db.sql`
- `logic-grpc-service/**`
- `web-gin-service/**`
- `hr-frontend/src/api/agentRun.ts`
- `hr-frontend/src/composables/useHrAgentRun.ts`
- `user-frontend/**`
- `interviewer-frontend/**`

### Dependencies

- TASK-HATO-001 must be complete.
- TASK-HATO-002 should be complete.

### Acceptance Criteria

- The panel checks active-run state without blocking historical trace display.
- Active status is visible when an active run exists.
- SSE events update live status or live process indicators.
- Subscription errors preserve loaded data and show a recoverable warning.
- Closing the drawer or unmounting aborts only the subscription.
- No code path in this task calls `cancelAgentRun`.

### Required Tests

- `pnpm --filter hr-frontend typecheck`
- `pnpm --filter hr-frontend test`

### Risks

- Duplicate runtime state may conflict with `useHrAgentRun`; keep the panel passive and read-only.

### Notes

- If existing API helpers are insufficient, stop and request confirmation rather than modifying shared run APIs in this task.

## TASK-HATO-006 - Optional Paginated Trace Retrieval

### Goal

Optionally add backward-compatible pagination or lazy retrieval support for long trace histories after explicit user confirmation.

### Scope

- Extend trace retrieval contracts only in a backward-compatible way.
- Preserve default behavior when clients omit pagination parameters.
- Update gateway, logic service, proto/types, frontend API helpers, and focused tests if confirmed.
- Do not change database schema.

### Allowed Files

- `hr-frontend/src/api/ai.ts`
- `hr-frontend/src/types/ai.ts`
- `hr-frontend/src/components/AgentTracePanel.vue`
- `hr-frontend/src/components/AgentTracePanel.test.ts`
- `hr-frontend/src/components/agent-trace/**`
- `logic-grpc-service/proto/recruitment.proto`
- `logic-grpc-service/recruitment/pb/**`
- `logic-grpc-service/service/ai_service.go`
- `logic-grpc-service/service/*ai*test*.go`
- `logic-grpc-service/repository/*tool*trace*.go`
- `logic-grpc-service/repository/*agent_run*.go`
- `logic-grpc-service/repository/*test*.go`
- `web-gin-service/proto/recruitment.proto`
- `web-gin-service/recruitment/pb/**`
- `web-gin-service/handler/hr/ai.go`
- `web-gin-service/handler/hr/*ai*test*.go`
- `web-gin-service/router/**`
- `.spec/hr-agent-trace-optimization/reports/TASK-HATO-006-report.md`
- `.spec/hr-agent-trace-optimization/reports/TASK-HATO-006-evidence.json`

### Forbidden Files

- `package.json`
- `pnpm-lock.yaml`
- `*/package.json`
- `*/pnpm-lock.yaml`
- `db.sql`
- `logic-grpc-service/migrations/**`
- `user-frontend/**`
- `interviewer-frontend/**`

### Dependencies

- TASK-HATO-003 should be complete.
- Explicit user confirmation is required before implementation.

### Acceptance Criteria

- Existing clients without pagination parameters receive compatible responses.
- New pagination parameters or cursor behavior are documented in code/tests.
- Frontend can request additional trace data or indicate long-history truncation.
- Backend tests cover default compatibility and paginated responses.
- Frontend typecheck and relevant frontend tests pass.
- Go tests for affected logic and gateway packages pass.

### Required Tests

- `pnpm --filter hr-frontend typecheck`
- `pnpm --filter hr-frontend test`
- `cd logic-grpc-service && go test ./service ./repository -count=1`
- `cd web-gin-service && go test ./handler/... ./router/... -count=1`

### Risks

- This task touches public API/protobuf and generated code. It must not start without explicit confirmation.

### Notes

- If pagination can be satisfied by frontend lazy rendering alone, do not modify backend contracts.

## Required Knowledge Review

Every non-trivial TASK must review `AGENTS.md`, this feature contract, `.knowledge/README.md`, and active knowledge routed by the TASK file scope. Likely relevant active documents include:

- `.knowledge/architecture/frontend-apps.md`
- `.knowledge/architecture/agent-runtime.md`
- `.knowledge/architecture/api-contracts-and-gateway.md`
- `.knowledge/architecture/service-boundaries.md`
- `.knowledge/runbooks/frontend-validation.md`
- `.knowledge/runbooks/protobuf-and-migration-change.md`
- `.knowledge/pitfalls/protobuf-synchronization.md`
- `.knowledge/pitfalls/resume-sensitive-data.md`
- `.knowledge/pitfalls/auth-permission-alignment.md`

If a TASK discovers knowledge drift and its scope does not allow editing `.knowledge`, report the drift in the TASK report instead of modifying out-of-scope files.
