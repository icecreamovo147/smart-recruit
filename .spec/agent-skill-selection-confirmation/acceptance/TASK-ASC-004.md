# Acceptance - TASK-ASC-004

## TASK Summary

Keep execution trace and retry behavior consistent after confirmed Skill selection.

## SPEC References

- FR-008
- FR-009
- Observability and Debug Requirements

## SDD References

- Section 10 Observability and Debug Output Design
- Section 11 Testing Strategy

## Acceptance Criteria

- Confirmed Skill IDs/details are visible in execution trace after final execution.
- Retry preserves confirmed Skill choices.
- Selection-required pre-execution state is not shown as a completed AI answer.
- Trace UI remains compact for Skill reason metadata.

## Required Checks

- `git diff --name-only`
- `bash .spec/agent-skill-selection-confirmation/scripts/check-task-scope.sh TASK-ASC-004`
- `bash .spec/agent-skill-selection-confirmation/scripts/agent-check.sh`
- `cd logic-grpc-service && go test ./service -run 'Test.*AgentRun.*|Test.*AgentSkill.*'`
- `pnpm --filter hr-frontend typecheck`

## Manual Verification, if needed

Open HR chat execution trace after a confirmed flow and verify Skill display.

## Out-of-Scope

- Full trace panel redesign.
- Ranking changes.
