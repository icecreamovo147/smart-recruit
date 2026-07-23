# Acceptance - TASK-HARS-008

## TASK Summary

Wire and validate the complete refresh/reconnect behavior across frontend and backend.

## SPEC References

- FR-003
- FR-007 through FR-017
- AC-001 through AC-010

## SDD References

- Section 6: Algorithm or Workflow Changes
- Section 8: Compatibility Strategy
- Section 9: Error Handling and Fallback Design
- Section 11: Testing Strategy

## Acceptance Criteria

- On mount/refresh, the HR chat queries active run for the current session.
- Active run snapshot hydrates visible answer, process trace, status, confirmation, and result state.
- Subscription resumes from the last known sequence and replays missing events without duplication.
- Browser refresh or subscription disconnect does not cancel the backend run.
- Explicit cancel persists and renders canceled state.
- Waiting confirmation survives refresh before and after submission.
- Completion reconciles transient state with persisted chat history.
- Legacy stream compatibility is verified or documented as still available.
- Final report records knowledge impact and remaining rollout risks.

## Required Checks

- `git diff --name-only`
- `bash .spec/hr-agent-resumable-stream/scripts/check-task-scope.sh TASK-HARS-008`
- `bash .spec/hr-agent-resumable-stream/scripts/agent-check.sh`
- `pnpm --filter hr-frontend typecheck`
- `pnpm --filter hr-frontend test`
- Relevant Go tests from changed backend packages.

## Manual Verification, if needed

Exercise refresh while a run is active in local development or document why the environment cannot support it.

## Out-of-Scope

- Package manifests or lockfiles.
- Schema or proto files.
- Legacy stream removal.
- Unrelated frontend apps.
