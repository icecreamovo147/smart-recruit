Use spec-harness.

Mode: fix-check-failures
Feature: microservice-ddd-evolution
Task: <TASK-ID>

Fix only failures from the previous Harness check, test output, self-review, or user-provided failure log. Read:

- .spec/microservice-ddd-evolution/microservice-ddd-evolution-SPEC.md
- .spec/microservice-ddd-evolution/microservice-ddd-evolution-SDD.md
- .spec/microservice-ddd-evolution/TASKS.md
- .spec/microservice-ddd-evolution/AGENT_RULES.md
- .spec/microservice-ddd-evolution/task-scope.json
- .spec/microservice-ddd-evolution/acceptance/<TASK-ID>.md
- .spec/microservice-ddd-evolution/reports/<TASK-ID>-report.md

Do not expand scope. Do not add new functionality. Re-run:

```bash
git diff --name-only
bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh <TASK-ID>
bash .spec/microservice-ddd-evolution/scripts/agent-check.sh
```

Also re-run the originally failed command. Update the TASK report and evidence with repair details.
