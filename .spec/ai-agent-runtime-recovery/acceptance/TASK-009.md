# Acceptance - TASK-009

## TASK Summary

Verify and adjust frontend API/type/view compatibility for restored AI Agent backend behavior.

## SPEC References

- FR-010 Compatibility of public contracts
- User-Facing Behavior
- Acceptance Criteria 1 through 12, 15

## SDD References

- API and Interface Changes
- Compatibility Strategy
- Testing Strategy frontend section

## Acceptance Criteria

- HR AI Chat UI handles restored stream event types, context usage, candidate options, and Agent Skill selection payloads.
- HR durable run UI remains compatible with restored run event metadata.
- Application Intelligence UI handles generated profile/evaluation responses.
- MCP, Embedding, and Semantic Debug admin views handle real runtime success/failure states.
- Candidate AI API/types remain compatible with stream and non-stream behavior.
- No unrelated UI redesign or global frontend config change is introduced.

## Required Checks

- `pnpm --filter hr-frontend typecheck` if HR frontend is touched.
- `pnpm --filter user-frontend typecheck` if user frontend is touched.
- Focused Vitest for touched parsing/composable behavior where existing test patterns are present.
- `git diff --name-only`.
- `bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh TASK-009`.
- `bash .spec/ai-agent-runtime-recovery/scripts/agent-check.sh`.

## Manual Verification, if needed

Document any unavailable local UI smoke and the backend/test substitute used.

## Out-of-Scope

- Backend source changes.
- Package dependency changes.
- New UX features beyond compatibility.
