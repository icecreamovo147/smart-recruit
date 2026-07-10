# Acceptance - TASK-HARS-005

## TASK Summary

Expose the durable run lifecycle to the HR frontend through authenticated gateway endpoints.

## SPEC References

- FR-007 through FR-010
- FR-017
- AC-003 through AC-005
- AC-008

## SDD References

- Section 5: API and Interface Changes
- Section 8: Compatibility Strategy
- Section 9: Error Handling and Fallback Design

## Acceptance Criteria

- Adds authenticated HR endpoints for create, get, active lookup, subscribe, cancel, and confirm.
- Forwards commands and queries to logic-grpc without gateway-local lifecycle state.
- SSE endpoint supports replay after `after_seq` or `Last-Event-ID`.
- HTTP disconnect ends only the SSE subscription.
- Ownership/auth checks prevent cross-user run access.
- Handler tests cover validation, auth failure, replay parameters, and disconnect behavior where practical.

## Required Checks

- `git diff --name-only`
- `bash .spec/hr-agent-resumable-stream/scripts/check-task-scope.sh TASK-HARS-005`
- `bash .spec/hr-agent-resumable-stream/scripts/agent-check.sh`
- Focused web-gin Go tests.

## Manual Verification, if needed

Confirm public HTTP API changes were explicitly approved before editing.

## Out-of-Scope

- Database schema.
- HR frontend files.
- Legacy stream endpoint removal.
- Gateway-local run state.
