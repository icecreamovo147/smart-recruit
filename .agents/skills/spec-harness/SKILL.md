---
name: spec-harness
description: Use this skill to create, verify, execute, review, and repair a SPEC + SDD + Harness workflow for one feature under .spec/<feature-name>.
---

# spec-harness

Use this skill for SPEC + SDD + Harness work under `.spec/<feature-name>`.

This skill supports four modes:

- `prepare-harness`
- `implement-task`
- `self-review`
- `fix-check-failures`

## Common Rules

- Work from the current repository files only.
- Do not infer missing SPEC, SDD, TASK, or acceptance requirements.
- Do not modify existing SPEC or SDD files.
- Do not modify business code except during `implement-task` or `fix-check-failures`, and only within the current TASK scope.
- Do not execute more than one TASK at a time.
- Do not continue to the next TASK without user confirmation.

## prepare-harness

Purpose: generate TASKS and Harness files for one feature from existing SPEC and SDD.

Must read:

- `.spec/<feature-name>/<feature-name>-SPEC.md`
- `.spec/<feature-name>/<feature-name>-SDD.md`

Allowed to create:

- `.spec/<feature-name>/TASKS.md`
- `.spec/<feature-name>/AGENT_RULES.md`
- `.spec/<feature-name>/task-scope.json`
- `.spec/<feature-name>/acceptance/`
- `.spec/<feature-name>/prompts/`
- `.spec/<feature-name>/scripts/`
- `.spec/<feature-name>/reports/`

Forbidden:

- Do not modify business code.
- Do not modify existing SPEC or SDD.
- Do not implement functionality.

Output:

- Created files and directories.
- Any detected SPEC and SDD conflicts.
- Any assumptions that need user confirmation.
- Instructions for running the first TASK.

## implement-task

Purpose: execute exactly one specified TASK.

Must read:

- `.spec/<feature-name>/<feature-name>-SPEC.md`
- `.spec/<feature-name>/<feature-name>-SDD.md`
- `.spec/<feature-name>/TASKS.md`
- `.spec/<feature-name>/AGENT_RULES.md`
- `.spec/<feature-name>/task-scope.json`
- `.spec/<feature-name>/acceptance/<TASK-ID>.md`

Restrictions:

- Execute only the specified TASK.
- Do not execute later TASKs.
- Do not modify files outside the TASK scope.
- If the TASK requires a wider scope, stop and request confirmation.
- Run Harness checks after implementation.
- Generate a TASK completion report.
- Stop after the report.

Required checks after implementation:

- `git diff --name-only`
- `bash .spec/<feature-name>/scripts/check-task-scope.sh <TASK-ID>`
- `bash .spec/<feature-name>/scripts/agent-check.sh`

Completion report must include:

- TASK ID
- Modified file list
- Change summary for each file
- Whether the changes exceed task scope
- SPEC comparison result
- SDD comparison result
- Acceptance comparison result
- Test commands and results
- Risks
- Whether the next TASK can start

## self-review

Purpose: review the current TASK diff.

Review against:

- SPEC
- SDD
- TASKS
- `task-scope.json`
- acceptance
- `AGENT_RULES.md`

Default behavior:

- Do not modify code.
- Report findings ordered by severity.
- Identify scope violations, missing tests, incomplete acceptance criteria, and behavior not covered by SPEC or SDD.

## fix-check-failures

Purpose: fix only failures from the previous Harness or test check.

Restrictions:

- Fix only the failed items.
- Do not expand scope.
- Do not add new functionality.
- Do not refactor unrelated code.
- If two consecutive repair attempts fail, stop and explain the root cause.

Required after fixes:

- Re-run the failed check.
- Re-run Harness checks when applicable.
- Update or create the TASK completion report with the repair summary.

## Hard Stop Conditions

Stop and request confirmation if any of the following occur:

- Need to modify `package.json` or a lockfile.
- Need to modify a shared module.
- Need to modify shared types.
- Need to modify global configuration.
- Need to change public API behavior.
- TASK scope is unclear.
- SPEC, SDD, and TASKS conflict.
- Required behavior is not present in SPEC or SDD.
