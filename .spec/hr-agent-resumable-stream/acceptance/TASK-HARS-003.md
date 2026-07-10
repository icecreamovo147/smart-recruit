# Acceptance - TASK-HARS-003

## TASK Summary

Define the logic-grpc contract for create, get, active lookup, event subscription, cancel, and confirm operations.

## SPEC References

- FR-001
- FR-007 through FR-011
- FR-017
- AC-003
- AC-008

## SDD References

- Section 5: API and Interface Changes
- Section 8: Compatibility Strategy
- Section 12: Migration Risks

## Acceptance Criteria

- Adds logic-grpc RPC definitions for create, get, active lookup, subscribe events, cancel, and confirm operations.
- Defines messages for run snapshot, event payload, status, confirmation payload, and result metadata.
- Mirrors proto changes between logic and gateway proto trees.
- Regenerates generated Go code in both services.
- Keeps numeric field tags stable and non-overlapping.
- Preserves existing `ChatStream` compatibility.

## Required Checks

- `git diff --name-only`
- `bash .spec/hr-agent-resumable-stream/scripts/check-task-scope.sh TASK-HARS-003`
- `bash .spec/hr-agent-resumable-stream/scripts/agent-check.sh`
- Proto generation command output or explanation of the repo's generated-code workflow.

## Manual Verification, if needed

Confirm public contract changes were explicitly approved before editing.

## Out-of-Scope

- Database schema.
- Frontend files.
- Business behavior beyond compile alignment.
- Legacy RPC removal.
