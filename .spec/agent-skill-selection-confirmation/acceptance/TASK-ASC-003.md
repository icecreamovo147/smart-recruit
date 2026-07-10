# Acceptance - TASK-ASC-003

## TASK Summary

Implement HR chat confirmation UI and confirmation resubmission behavior.

## SPEC References

- FR-006
- FR-007
- FR-008
- User-Facing Behavior

## SDD References

- Section 3 Proposed Design
- Section 6 Algorithm or Workflow Changes
- Section 13 Implementation Boundaries

## Acceptance Criteria

- Selection-required event renders an inline assistant-side confirmation UI.
- UI supports selecting one, multiple, or none.
- Confirmation resubmits the original message with selected IDs.
- Manual composer-selected Skills bypass the confirmation UI.
- Loading/streaming state is cleared while waiting for user confirmation.
- UI remains compact and responsive.

## Required Checks

- `git diff --name-only`
- `bash .spec/agent-skill-selection-confirmation/scripts/check-task-scope.sh TASK-ASC-003`
- `bash .spec/agent-skill-selection-confirmation/scripts/agent-check.sh`
- `pnpm --filter hr-frontend typecheck`

## Manual Verification, if needed

Use local HR chat to simulate a selection-required event if automated mocking is not available.

## Out-of-Scope

- Backend selection policy changes.
- Proto changes.
