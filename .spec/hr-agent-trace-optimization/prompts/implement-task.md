Use spec-harness.

Mode: implement-task
Feature: hr-agent-trace-optimization
Task: <TASK-ID>

Before editing:

1. Read `AGENTS.md`.
2. Read `.agents/skills/spec-harness/SKILL.md`.
3. Read `.spec/hr-agent-trace-optimization/hr-agent-trace-optimization-SPEC.md`.
4. Read `.spec/hr-agent-trace-optimization/hr-agent-trace-optimization-SDD.md`.
5. Read `.spec/hr-agent-trace-optimization/TASKS.md`.
6. Read `.spec/hr-agent-trace-optimization/AGENT_RULES.md`.
7. Read `.spec/hr-agent-trace-optimization/task-scope.json`.
8. Read `.spec/hr-agent-trace-optimization/acceptance/<TASK-ID>.md`.
9. Read this prompt.
10. Follow `.knowledge/README.md` and routed active knowledge for the TASK scope.

Implement exactly one TASK. Modify only files allowed by the TASK scope. Stop on Hard Stop conditions, missing confirmation, unsupported harness state, unreliable baseline, or any need to expand scope.

After implementation, run:

```bash
git diff --name-only
bash .spec/hr-agent-trace-optimization/scripts/check-task-scope.sh <TASK-ID>
bash .spec/hr-agent-trace-optimization/scripts/agent-check.sh <TASK-ID>
```

Also run required checks listed in the acceptance file. Create or update the TASK Markdown report and evidence JSON.
