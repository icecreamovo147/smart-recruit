# Acceptance - TASK-HARS-004

## TASK Summary

Make Agent execution backend-owned and durable across browser disconnects.

## SPEC References

- FR-001 through FR-011
- FR-016 through FR-017
- AC-001 through AC-006
- AC-008

## SDD References

- Section 3: Proposed Design
- Section 6: Algorithm or Workflow Changes
- Section 9: Error Handling and Fallback Design

## Acceptance Criteria

- Create run is idempotent for HR user, session, and client request id.
- Run execution does not use the HTTP/SSE subscription context as its lifetime owner.
- Worker appends ordered events and updates snapshots while running.
- Explicit cancel transitions through cancel requested and reaches canceled when observed.
- Subscription disconnect does not mark the run canceled.
- Waiting confirmation is persisted and can resume the same logical run.
- Final completion persists exactly one assistant message for the run.
- Existing legacy stream behavior remains available.

## Required Checks

- `git diff --name-only`
- `bash .spec/hr-agent-resumable-stream/scripts/check-task-scope.sh TASK-HARS-004`
- `bash .spec/hr-agent-resumable-stream/scripts/agent-check.sh`
- Focused Go tests for create idempotency, cancellation, waiting confirmation, event append, and final persistence.

## Manual Verification, if needed

Confirm any worker startup, queue, timeout, or config change was explicitly approved.

## Out-of-Scope

- Gateway handlers.
- Frontend files.
- Migrations or `db.sql`.
- Legacy stream removal.
