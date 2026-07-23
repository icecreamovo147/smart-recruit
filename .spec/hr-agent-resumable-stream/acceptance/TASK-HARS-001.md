# Acceptance - TASK-HARS-001

## TASK Summary

Create the durable persistence foundation for resumable HR Agent runs.

## SPEC References

- FR-001 through FR-006
- FR-015 through FR-016
- AC-001 through AC-003
- AC-010

## SDD References

- Section 4: Data Structure Changes
- Section 12: Migration Risks
- Section 13: Implementation Boundaries

## Acceptance Criteria

- Adds forward and rollback migrations for durable Agent run events and refresh recovery fields.
- Keeps `db.sql` aligned with the migration result.
- Updates Go models for new or changed columns.
- Supports idempotent create lookup by owner/session/client request id.
- Supports active run lookup by chat session.
- Supports final assistant message association with a run.
- Adds indexes for event replay and active-run lookup.
- Rollback removes only this feature's schema changes.

## Required Checks

- `git diff --name-only`
- `bash .spec/hr-agent-resumable-stream/scripts/check-task-scope.sh TASK-HARS-001`
- `bash .spec/hr-agent-resumable-stream/scripts/agent-check.sh`
- Relevant logic service Go tests, or a written explanation if database-dependent tests cannot run locally.

## Manual Verification, if needed

Confirm the migration number does not collide with existing migrations and that rollback ordering is safe.

## Out-of-Scope

- Proto files.
- Frontend files.
- Gateway files.
- Runtime behavior.
- Package dependencies.
