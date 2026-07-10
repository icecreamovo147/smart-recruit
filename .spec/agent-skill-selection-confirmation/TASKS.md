# TASKS - agent-skill-selection-confirmation

## Task Overview

| TASK | Title | Status | Scope | Acceptance |
|------|-------|--------|-------|------------|
| TASK-ASC-001 | Backend selection confirmation policy | pending | logic service only | acceptance/TASK-ASC-001.md |
| TASK-ASC-002 | Stream/API payload contract | pending | proto + HTTP gateway + frontend types | acceptance/TASK-ASC-002.md |
| TASK-ASC-003 | HR chat confirmation UI | pending | HR frontend chat flow | acceptance/TASK-ASC-003.md |
| TASK-ASC-004 | Backend emission, trace, and retry consistency | pending | backend emission + trace/recorder/frontend retry | acceptance/TASK-ASC-004.md |
| TASK-ASC-005 | End-to-end validation and cleanup | pending | tests and harness verification | acceptance/TASK-ASC-005.md |

## TASK-ASC-001 - Backend selection confirmation policy

### Goal

Create a generic backend policy that can decide when multiple automatically matched Agent Skills require user confirmation before AI execution.

### Scope

Logic service Skill selection and AI orchestration helpers only. No proto or frontend changes in this task.

### Allowed Files

- `logic-grpc-service/service/agent_skill_selector.go`
- `logic-grpc-service/service/agent_skill_selector_test.go`
- `logic-grpc-service/service/ai_service.go`
- `logic-grpc-service/service/ai_service_test.go`
- `.spec/agent-skill-selection-confirmation/**`

### Forbidden Files

- `package.json`
- `pnpm-lock.yaml`
- `logic-grpc-service/proto/**`
- `web-gin-service/proto/**`
- generated `*.pb.go`
- frontend files

### Dependencies

None.

### Acceptance Criteria

- Policy distinguishes manual selections from automatic candidates.
- Multiple automatic candidates can be classified as confirmation-required.
- Manual `agent_skill_ids` bypass confirmation.
- Unit tests cover zero, one, multiple, and manual-selection cases.

### Required Tests

- `cd logic-grpc-service && go test ./service -run 'Test.*AgentSkill.*'`

### Risks

- Touching `ai_service.go` may affect both ADK and legacy runtime paths.

### Notes

Do not emit stream events in this task; only establish policy and helper shape.

## TASK-ASC-002 - Stream/API payload contract

### Goal

Expose a structured selection-required response from backend to HR frontend streaming clients.

### Scope

Proto/HTTP gateway/frontend type contract only.

### Allowed Files

- `logic-grpc-service/proto/recruitment.proto`
- `logic-grpc-service/recruitment/pb/recruitment.pb.go`
- `logic-grpc-service/recruitment/pb/recruitment_grpc.pb.go`
- `web-gin-service/proto/recruitment.proto`
- `web-gin-service/recruitment/pb/recruitment.pb.go`
- `web-gin-service/recruitment/pb/recruitment_grpc.pb.go`
- `web-gin-service/handler/hr/ai.go`
- `hr-frontend/src/api/ai.ts`
- `hr-frontend/src/types/ai.ts`
- `.spec/agent-skill-selection-confirmation/**`

### Forbidden Files

- `package.json`
- lockfiles
- database migrations
- unrelated handlers

### Dependencies

- TASK-ASC-001.

### Acceptance Criteria

- Stream response can carry `agent_skill_selection_required` with candidate metadata.
- HTTP gateway forwards the payload in SSE JSON.
- Frontend types compile with the new payload shape.
- Proto copies remain synchronized if proto is modified.

### Required Tests

- `cd logic-grpc-service && go test ./...`
- `cd web-gin-service && go test ./...`
- `pnpm --filter hr-frontend typecheck`

### Risks

- Public API/proto changes require human confirmation before implementation.

### Notes

If a no-proto transitional encoding is chosen, document why in the TASK report.

## TASK-ASC-003 - HR chat confirmation UI

### Goal

Render Skill selection confirmation in HR AI chat and submit confirmed choices.

### Scope

HR frontend chat components and tests.

### Allowed Files

- `hr-frontend/src/views/hr/AIChatView.vue`
- `hr-frontend/src/components/chat/**`
- `hr-frontend/src/types/ai.ts`
- `hr-frontend/src/api/ai.ts`
- `hr-frontend/src/**/*.test.ts`
- `.spec/agent-skill-selection-confirmation/**`

### Forbidden Files

- backend files
- package manifests
- lockfiles
- global styling unrelated to chat

### Dependencies

- TASK-ASC-002.

### Acceptance Criteria

- Selection-required event stops pending generation and shows a confirmation UI.
- User can confirm one, multiple, or none.
- Confirmed submission uses original message/session/model and selected IDs.
- Manual composer selections still execute directly.
- UI does not overflow or obscure chat content on desktop and mobile widths.

### Required Tests

- `pnpm --filter hr-frontend typecheck`
- Targeted Vitest tests if existing chat test harness supports this flow.

### Risks

- Incorrect state handling can duplicate messages or lose user input.

### Notes

Prefer reusing existing chat visual patterns over adding a modal.

## TASK-ASC-004 - Backend emission, trace, and retry consistency

### Goal

Wire the backend confirmation decision into AI execution, then ensure confirmed selections appear correctly in execution trace and retry flow.

### Scope

AI service orchestration, Agent run recorder, and HR retry/message metadata behavior.

### Allowed Files

- `logic-grpc-service/service/ai_service.go`
- `logic-grpc-service/service/ai_service_test.go`
- `logic-grpc-service/service/agent_run_recorder.go`
- `logic-grpc-service/service/agent_run_recorder_test.go`
- `logic-grpc-service/service/agent_skill_selector.go`
- `hr-frontend/src/views/hr/AIChatView.vue`
- `hr-frontend/src/components/AgentTracePanel.vue`
- `hr-frontend/src/types/ai.ts`
- `.spec/agent-skill-selection-confirmation/**`

### Forbidden Files

- proto files unless TASK-ASC-002 explicitly left required follow-up
- package manifests
- lockfiles
- database migrations

### Dependencies

- TASK-ASC-003.

### Acceptance Criteria

- Backend emits `agent_skill_selection_required` before prompt injection when the policy requires confirmation.
- Confirmed resubmissions bypass confirmation, including an explicitly empty Skill selection.
- Execution trace identifies confirmed Skill IDs/details after final execution.
- Selection-required pre-execution state does not appear as a completed AI answer.
- Retry preserves confirmed Skill choices.
- Trace UI remains compact for Skill reason metadata.

### Required Tests

- `cd logic-grpc-service && go test ./service -run 'Test.*AgentRun.*|Test.*AgentSkill.*'`
- `pnpm --filter hr-frontend typecheck`

### Risks

- AI execution must stop cleanly after emitting selection-required, without persisting a completed assistant answer.
- Trace records may be confusing if a run is started before confirmation.

### Notes

Do not redesign the trace panel beyond this feature's metadata display.

## TASK-ASC-005 - End-to-end validation and cleanup

### Goal

Run broad validation and make only feature-scoped cleanup required by failing checks.

### Scope

Tests, reports, and narrowly scoped fixes from prior task findings.

### Allowed Files

- `logic-grpc-service/service/**`
- `web-gin-service/handler/hr/ai.go`
- `hr-frontend/src/views/hr/AIChatView.vue`
- `hr-frontend/src/components/chat/**`
- `hr-frontend/src/components/AgentTracePanel.vue`
- `hr-frontend/src/types/ai.ts`
- `hr-frontend/src/api/ai.ts`
- `.spec/agent-skill-selection-confirmation/**`

### Forbidden Files

- package manifests
- lockfiles
- database migrations
- unrelated frontend/backend modules

### Dependencies

- TASK-ASC-001 through TASK-ASC-004.

### Acceptance Criteria

- Backend tests pass.
- HR frontend typecheck passes.
- Harness scope checks pass after unrelated dirty files are resolved or excluded.
- Final report lists remaining risks and manual verification steps.

### Required Tests

- `cd logic-grpc-service && go test ./...`
- `cd web-gin-service && go test ./...`
- `pnpm --filter hr-frontend typecheck`

### Risks

- Existing unrelated dirty worktree changes can cause scope checks to fail.

### Notes

Do not expand scope to fix unrelated pre-existing failures.
