# Acceptance - TASK-HARS-007

## TASK Summary

Remove duplicated stream orchestration from the HR chat page by routing existing flows through the unified runtime.

## SPEC References

- FR-012 through FR-017
- AC-007
- AC-008

## SDD References

- Section 3: Proposed Design
- Section 8: Compatibility Strategy
- Section 13: Implementation Boundaries

## Acceptance Criteria

- Normal submit uses the unified HR Agent runtime.
- Retry uses the unified HR Agent runtime.
- Route-created candidate analysis uses the unified HR Agent runtime.
- Candidate option actions use the unified HR Agent runtime.
- Skill confirmation uses the unified HR Agent runtime.
- Existing UI behavior remains stable for chat messages, process trace, loading, errors, and action options.
- Old duplicated stream orchestration is removed or reduced to compatibility glue.
- Focused tests cover at least two formerly separate entry paths using the same runtime.

## Required Checks

- `git diff --name-only`
- `bash .spec/hr-agent-resumable-stream/scripts/check-task-scope.sh TASK-HARS-007`
- `bash .spec/hr-agent-resumable-stream/scripts/agent-check.sh`
- `pnpm --filter hr-frontend typecheck`
- `pnpm --filter hr-frontend test`

## Manual Verification, if needed

Confirm UX remains visually stable in the HR Agent chat page.

## Out-of-Scope

- Backend files.
- Proto or schema files.
- Unrelated HR pages.
- Candidate or interviewer frontend behavior.
