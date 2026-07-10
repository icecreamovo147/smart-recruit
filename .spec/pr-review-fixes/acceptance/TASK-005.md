# Acceptance - TASK-005

## TASK Summary

Clamp management list `page_size` values to a maximum of 100.

## SPEC References

- FR-012
- AC-008

## SDD References

- Section 3: Pagination
- Section 6: Algorithm or Workflow Changes

## Acceptance Criteria

- A shared helper normalizes management pagination.
- `page <= 0` becomes `1`.
- `page_size <= 0` becomes `20`.
- `page_size > 100` becomes `100`.
- LLM, embedding, MCP, prompt, skill, and agent config management list methods use the helper.
- Public candidate/job pagination behavior is unchanged.

## Required Checks

```bash
cd logic-grpc-service && go test ./service
git diff --name-only
bash .spec/pr-review-fixes/scripts/check-task-scope.sh TASK-005
bash .spec/pr-review-fixes/scripts/agent-check.sh
```

## Manual Verification, if needed

Review touched services to confirm only management list APIs were changed.

## Out-of-Scope

- Export APIs.
- Frontend pagination redesign.
- Public candidate/job listing behavior.
