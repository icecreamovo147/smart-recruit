# Fix Check Failures TASK Prompt

Use `spec-harness`.

Mode: fix-check-failures
Feature: pr-review-fixes
Task: <TASK-ID>

Required reading:

- `.spec/pr-review-fixes/pr-review-fixes-SPEC.md`
- `.spec/pr-review-fixes/pr-review-fixes-SDD.md`
- `.spec/pr-review-fixes/TASKS.md`
- `.spec/pr-review-fixes/AGENT_RULES.md`
- `.spec/pr-review-fixes/task-scope.json`
- `.spec/pr-review-fixes/acceptance/<TASK-ID>.md`
- `.spec/pr-review-fixes/reports/<TASK-ID>-report.md`

Fix only failures from the previous Harness, test, or self-review output.

Rules:

- Do not expand TASK scope.
- Do not add unrelated functionality.
- Stop if a required fix needs files outside scope.
- Re-run failed checks and Harness checks.
- Update the TASK report with a repair summary.
