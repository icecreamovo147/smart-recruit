# Acceptance - TASK-003

## TASK Summary

Restore durable HR Agent Run execution, events, confirmation continuation, cancellation, and replay behavior.

## SPEC References

- FR-003 Durable Agent Run restoration
- Observability and Debug Requirements 2, 3
- Error Handling and Fallback Requirements 7, 8
- Acceptance Criteria 3

## SDD References

- Proposed Design 3.3
- Algorithm or Workflow Changes 6.2
- Error Handling and Fallback Design
- Testing Strategy

## Acceptance Criteria

- `CreateAgentRun` stores a durable payload and creates idempotent queued runs.
- Runs execute through the restored HR runtime rather than provider `complete()` only.
- Run events are persisted with replayable sequence numbers and meaningful event metadata.
- Confirmation moves waiting runs forward and continues execution.
- Cancellation reaches a terminal canceled state for queued/running work.
- Invalid transitions are rejected without corrupting persisted state.

## Required Checks

- Targeted AI Agent service tests for run create, execute, replay, confirm, cancel, idempotency, and invalid transitions.
- `GOWORK=off go test ./...` from `smart-recruit-ai-agent-service`.
- `git diff --name-only`.
- `bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh TASK-003`.
- `bash .spec/ai-agent-runtime-recovery/scripts/agent-check.sh`.

## Manual Verification, if needed

If live streaming smoke is unavailable, document skipped live run smoke and provide fake runtime/event replay test evidence.

## Out-of-Scope

- Frontend run UI changes.
- Protobuf or schema changes without confirmation.
- MCP and Embedding runtime.
