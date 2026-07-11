Use spec-harness.

Mode: fix-check-failures
Feature: hr-agent-trace-optimization
Task: <TASK-ID>

Fix only failures from the previous Harness, test, or self-review output. Do not add new functionality and do not expand TASK scope.

Read:

1. `AGENTS.md`
2. `.agents/skills/spec-harness/SKILL.md`
3. `.spec/hr-agent-trace-optimization/hr-agent-trace-optimization-SPEC.md`
4. `.spec/hr-agent-trace-optimization/hr-agent-trace-optimization-SDD.md`
5. `.spec/hr-agent-trace-optimization/TASKS.md`
6. `.spec/hr-agent-trace-optimization/AGENT_RULES.md`
7. `.spec/hr-agent-trace-optimization/task-scope.json`
8. `.spec/hr-agent-trace-optimization/acceptance/<TASK-ID>.md`
9. `.spec/hr-agent-trace-optimization/reports/<TASK-ID>-report.md`
10. The failed Harness, test, or review output.

After fixes, re-run:

```bash
git diff --name-only
bash .spec/hr-agent-trace-optimization/scripts/check-task-scope.sh <TASK-ID>
bash .spec/hr-agent-trace-optimization/scripts/agent-check.sh <TASK-ID>
```

Also re-run the originally failed command. Update the TASK report and evidence.
