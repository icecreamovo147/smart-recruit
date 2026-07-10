# Acceptance - TASK-ASC-004

## TASK Summary

Wire backend selection-required emission into AI execution and keep execution trace/retry behavior consistent after confirmed Skill selection.

## SPEC References

- FR-008
- FR-009
- Observability and Debug Requirements

## SDD References

- Section 10 Observability and Debug Output Design
- Section 11 Testing Strategy

## Acceptance Criteria

- Backend emits `agent_skill_selection_required` with candidate metadata when policy requires confirmation.
- Backend stops before Agent Skill prompt injection and model execution for selection-required responses.
- Confirmed resubmissions bypass confirmation, including selecting none.
- Confirmed Skill IDs/details are visible in execution trace after final execution.
- Retry preserves confirmed Skill choices.
- Selection-required pre-execution state is not shown as a completed AI answer.
- Trace UI remains compact for Skill reason metadata.

## Required Checks

- `git diff --name-only`
- `bash .spec/agent-skill-selection-confirmation/scripts/check-task-scope.sh TASK-ASC-004`
- `bash .spec/agent-skill-selection-confirmation/scripts/agent-check.sh`
- `cd logic-grpc-service && go test ./service -run 'Test.*AI.*|Test.*AgentRun.*|Test.*AgentSkill.*'`
- `pnpm --filter hr-frontend typecheck`

## Manual Verification, if needed

Open HR chat execution trace after a confirmed flow and verify Skill display.

## Out-of-Scope

- Full trace panel redesign.
- Ranking changes.
