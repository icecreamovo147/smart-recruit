# Acceptance - TASK-004

## TASK Summary

Validate and mask LLM and embedding provider extra headers.

## SPEC References

- FR-009
- FR-010
- FR-011
- AC-006
- AC-007

## SDD References

- Section 3: Extra Headers
- Section 9: Error Handling and Fallback Design
- Section 10: Observability and Debug Output Design

## Acceptance Criteria

- Provider create/update rejects malformed `extra_headers_json`.
- Accepted headers are canonical JSON objects with string values.
- Header names are validated.
- Provider list/detail responses expose masked header values only.
- Invalid legacy data is not returned raw.
- Provider connection test behavior fails clearly when headers are invalid.

## Required Checks

```bash
cd logic-grpc-service && go test ./service
git diff --name-only
bash .spec/pr-review-fixes/scripts/check-task-scope.sh TASK-004
bash .spec/pr-review-fixes/scripts/agent-check.sh
```

## Manual Verification, if needed

Review test cases to ensure raw header values are absent from responses and logs.

## Out-of-Scope

- Frontend redesign.
- Secret vault integration.
- Protobuf changes.
