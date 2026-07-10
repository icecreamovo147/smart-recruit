# Implement TASK Prompt

Use `spec-harness`.

Mode: implement-task
Feature: pr-review-fixes
Task: <TASK-ID>

Required reading:

- `.spec/pr-review-fixes/pr-review-fixes-SPEC.md`
- `.spec/pr-review-fixes/pr-review-fixes-SDD.md`
- `.spec/pr-review-fixes/TASKS.md`
- `.spec/pr-review-fixes/AGENT_RULES.md`
- `.spec/pr-review-fixes/task-scope.json`
- `.spec/pr-review-fixes/acceptance/<TASK-ID>.md`

Rules:

- Execute only `<TASK-ID>`.
- Modify only files allowed by `task-scope.json`.
- Stop on Hard Stop conditions.
- Run required checks after implementation.
- Create or update `.spec/pr-review-fixes/reports/<TASK-ID>-report.md`.
- Do not continue to the next TASK without user confirmation.
