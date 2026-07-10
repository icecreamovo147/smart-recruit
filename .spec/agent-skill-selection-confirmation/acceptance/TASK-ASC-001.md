# Acceptance - TASK-ASC-001

## TASK Summary

Create backend policy helpers for deciding whether automatic Agent Skill matches require user confirmation.

## SPEC References

- FR-001
- FR-002
- FR-005
- AC-004

## SDD References

- Section 3 Proposed Design
- Section 6 Algorithm or Workflow Changes
- Section 11 Testing Strategy

## Acceptance Criteria

- Manual `agent_skill_ids` are recognized as explicit user choices.
- Multiple automatic candidates can produce a confirmation-required decision.
- Zero or one candidate can bypass confirmation.
- Policy is generic and does not hard-code Skill names.
- Tests cover the policy behavior.

## Required Checks

- `git diff --name-only`
- `bash .spec/agent-skill-selection-confirmation/scripts/check-task-scope.sh TASK-ASC-001`
- `bash .spec/agent-skill-selection-confirmation/scripts/agent-check.sh`
- `cd logic-grpc-service && go test ./service -run 'Test.*AgentSkill.*'`

## Manual Verification, if needed

None for this task.

## Out-of-Scope

- Stream payload changes.
- Frontend UI changes.
- Proto changes.
