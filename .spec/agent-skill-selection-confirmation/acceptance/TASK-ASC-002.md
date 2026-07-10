# Acceptance - TASK-ASC-002

## TASK Summary

Expose selection-required candidate metadata through the HR chat stream contract.

## SPEC References

- FR-002
- FR-003
- FR-004
- FR-006
- Compatibility Requirements

## SDD References

- Section 5 API and Interface Changes
- Section 8 Compatibility Strategy
- Section 12 Migration Risks

## Acceptance Criteria

- Stream clients can identify `agent_skill_selection_required`.
- Candidate payload includes safe Skill metadata and excludes Skill body content.
- HTTP gateway forwards the payload to SSE JSON.
- Frontend TypeScript payload types describe the new event.
- Proto copies and generated files are synchronized if proto is modified.

## Required Checks

- `git diff --name-only`
- `bash .spec/agent-skill-selection-confirmation/scripts/check-task-scope.sh TASK-ASC-002`
- `bash .spec/agent-skill-selection-confirmation/scripts/agent-check.sh`
- `cd logic-grpc-service && go test ./...`
- `cd web-gin-service && go test ./...`
- `pnpm --filter hr-frontend typecheck`

## Manual Verification, if needed

Confirm approval for proto/public API changes before implementing this TASK.

## Out-of-Scope

- Confirmation UI rendering.
- Ranking algorithm changes.
