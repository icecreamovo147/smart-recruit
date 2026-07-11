# Acceptance - TASK-HATO-005

## TASK Summary

Add read-only active execution visibility to the trace panel through existing active-run and SSE event APIs.

## SPEC References

- 4.5 Real-Time Status
- FR-010
- FR-011
- EFR-005
- ODR-002
- AC-008
- AC-009

## SDD References

- 3.5 Real-Time Integration
- 6.1 Loading Workflow
- 8 Compatibility Strategy
- 9 Error Handling and Fallback Design
- 10 Observability and Debug Output Design

## Acceptance Criteria

- Panel checks active-run state when opened with a valid session.
- Active run state is displayed without blocking historical trace display.
- SSE events update read-only live status or process indicators.
- Terminal events trigger a persisted trace refresh when feasible.
- Subscription errors preserve already loaded data and show a recoverable warning.
- Closing/unmounting aborts only the subscription.
- No code path calls `cancelAgentRun`.

## Required Checks

- `git diff --name-only`
- `bash .spec/hr-agent-trace-optimization/scripts/check-task-scope.sh TASK-HATO-005`
- `bash .spec/hr-agent-trace-optimization/scripts/agent-check.sh TASK-HATO-005`
- `pnpm --filter hr-frontend typecheck`
- `pnpm --filter hr-frontend test`

## Manual Verification, if needed

- Open the panel during a running HR Agent request and verify live status appears; close the panel and verify the run is not canceled.

## Out-of-Scope

- Adding cancel/confirm controls.
- Modifying `useHrAgentRun`.
- Modifying `hr-frontend/src/api/agentRun.ts`.
- Backend API changes.
