# AGENT RULES - agent-skill-review-fixes

## Feature Scope

This feature fixes three review findings:

- HR chat retry must support Agent Skill confirmation events.
- Single Agent Skill embedding regeneration must not report no-op as success.
- Semantic debug metadata must be request-local and Skill-phase accurate.

## Required Reading

Before each TASK, read:

- `.spec/agent-skill-review-fixes/agent-skill-review-fixes-SPEC.md`
- `.spec/agent-skill-review-fixes/agent-skill-review-fixes-SDD.md`
- `.spec/agent-skill-review-fixes/TASKS.md`
- `.spec/agent-skill-review-fixes/task-scope.json`
- `.spec/agent-skill-review-fixes/acceptance/<TASK-ID>.md`

## Scope Boundaries

- Execute only one TASK at a time.
- Modify only files allowed by `task-scope.json`.
- Do not refactor unrelated code.
- Do not mass-format files.
- Do not modify generated protobuf files unless explicitly confirmed.
- Do not modify package manifests, lockfiles, database migrations, auth, permissions, or CI.

## Testing Requirements

- Run `git diff --name-only`.
- Run `bash .spec/agent-skill-review-fixes/scripts/check-task-scope.sh <TASK-ID>`.
- Run `bash .spec/agent-skill-review-fixes/scripts/agent-check.sh`.
- Run TASK-specific checks listed in the acceptance file when feasible.

## Compatibility Rules

- Preserve existing public API shapes by default.
- Preserve existing HR chat submit and confirmed submit behavior.
- Preserve batch embedding backfill compatibility.
- Preserve semantic debug response fields.

## Logging and Debug Rules

- Keep existing logging style.
- Do not expose secret material in error messages.
- Preserve frontend debug logs unless directly superseded.

## Hard Stop Conditions

Stop and request confirmation if the TASK requires:

- Protobuf schema changes
- Database schema changes
- Auth, permission, or security changes
- Package manifest or lockfile changes
- Shared module changes outside the TASK scope
- Public API behavior changes beyond the SPEC/SDD

## Report Requirements

After implementation, update `.spec/agent-skill-review-fixes/reports/<TASK-ID>-report.md` with modified files, scope result, SPEC/SDD/acceptance comparison, tests, risks, and whether the next TASK can start.
