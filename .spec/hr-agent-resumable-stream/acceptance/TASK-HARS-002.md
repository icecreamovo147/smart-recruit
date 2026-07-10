# Acceptance - TASK-HARS-002

## TASK Summary

Implement backend state transition and event replay primitives after the schema exists.

## SPEC References

- FR-004 through FR-006
- FR-013
- AC-003
- AC-010

## SDD References

- Section 3: Proposed Design
- Section 6: Algorithm or Workflow Changes
- Section 11: Testing Strategy

## Acceptance Criteria

- Adds repository functions for appending run events with ordered sequence numbers.
- Adds replay query by `run_id` and `after_seq`.
- Adds snapshot/status update helpers needed by the run service.
- Defines transition validation for all required run states.
- Prevents illegal transitions in unit tests.
- Handles duplicate or stale event application safely.

## Required Checks

- `git diff --name-only`
- `bash .spec/hr-agent-resumable-stream/scripts/check-task-scope.sh TASK-HARS-002`
- `bash .spec/hr-agent-resumable-stream/scripts/agent-check.sh`
- Focused Go tests for repository/state behavior.

## Manual Verification, if needed

Confirm the event sequence allocation strategy is safe under concurrent append attempts.

## Out-of-Scope

- Schema migrations or `db.sql`.
- Proto files.
- Frontend or gateway files.
- Public API behavior.
