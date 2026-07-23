Use spec-harness.

Mode: fix-check-failures
Feature: legacydomain-retirement
Task: <TASK-ID>

Fix only failures from the previous Harness check, test output, self-review, or user-provided failure log. Read:

- .spec/legacydomain-retirement/legacydomain-retirement-SPEC.md
- .spec/legacydomain-retirement/legacydomain-retirement-SDD.md
- .spec/legacydomain-retirement/TASKS.md
- .spec/legacydomain-retirement/AGENT_RULES.md
- .spec/legacydomain-retirement/task-scope.json
- .spec/legacydomain-retirement/acceptance/<TASK-ID>.md
- .spec/legacydomain-retirement/reports/<TASK-ID>-report.md

Do not expand scope. Do not add new functionality. Re-run:

```bash
git diff --name-only
bash .spec/legacydomain-retirement/scripts/check-task-scope.sh <TASK-ID>
bash .spec/legacydomain-retirement/scripts/agent-check.sh
```

Also re-run the originally failed command. Update the TASK report and evidence with repair details, including knowledge impact if affected.
