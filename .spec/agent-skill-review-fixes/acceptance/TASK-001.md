# Acceptance - TASK-001

## TASK Summary

Fix HR AI chat retry so `agent_skill_selection_required` renders the Skill confirmation card and preserves it.

## SPEC References

- FR-001 Retry Skill Confirmation
- AC-001
- AC-002

## SDD References

- Section 3, TASK-001 Retry Confirmation Handling
- Section 11, Testing Strategy

## Acceptance Criteria

- `retry()` handles `agent_skill_selection_required`.
- Confirmation UI is created through the same path as normal submit where possible.
- Retry returns before session message refresh when Skill confirmation is pending.
- Existing `submit()` and `submitConfirmedSkillSelection()` behavior remains unchanged.

## Required Checks

- `git diff --name-only`
- `bash .spec/agent-skill-review-fixes/scripts/check-task-scope.sh TASK-001`
- `bash .spec/agent-skill-review-fixes/scripts/agent-check.sh`
- `pnpm --filter hr-frontend typecheck`

## Manual Verification, if needed

- Simulate or trigger a retried HR AI stream that emits `agent_skill_selection_required`.
- Confirm the Skill confirmation card remains visible after the stream completes.

## Out-of-Scope

- Backend changes.
- Protobuf changes.
- UI redesign.
