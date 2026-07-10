# Self Review TASK Prompt

Use `spec-harness`.

Mode: self-review
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

Review against:

- Scope compliance
- SPEC and SDD compliance
- Acceptance criteria
- Test results
- Compatibility risks
- Security risks

Do not modify files in this mode.

End with exactly one verdict:

- `verdict: 通过`
- `verdict: 不通过`
