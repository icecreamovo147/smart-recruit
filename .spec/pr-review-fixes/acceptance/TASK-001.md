# Acceptance - TASK-001

## TASK Summary

Enable CI for pull requests targeting the `dev` branch.

## SPEC References

- FR-001
- AC-001

## SDD References

- Section 3: CI
- Section 7: Configuration Design

## Acceptance Criteria

- `.github/workflows/ci.yml` includes `dev` and `main` under `pull_request.branches`.
- Existing workflow jobs and commands remain unchanged unless required for YAML correctness.
- No business code changes are made.

## Required Checks

```bash
git diff --name-only
bash .spec/pr-review-fixes/scripts/check-task-scope.sh TASK-001
bash .spec/pr-review-fixes/scripts/agent-check.sh
```

## Manual Verification, if needed

Review the YAML diff.

## Out-of-Scope

- Branch protection configuration outside the repository.
- CI job redesign.
