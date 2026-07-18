Use spec-harness.

Mode: implement-task
Feature: microservice-ddd-evolution
Task: <TASK-ID>

Read before editing:

- AGENTS.md
- .agents/skills/spec-harness/SKILL.md
- .spec/microservice-ddd-evolution/microservice-ddd-evolution-SPEC.md
- .spec/microservice-ddd-evolution/microservice-ddd-evolution-SDD.md
- .spec/microservice-ddd-evolution/TASKS.md
- .spec/microservice-ddd-evolution/AGENT_RULES.md
- .spec/microservice-ddd-evolution/task-scope.json
- .spec/microservice-ddd-evolution/acceptance/<TASK-ID>.md
- .knowledge/README.md and selected active knowledge per manifest routes

Implement exactly one TASK. Do not execute later TASKs. Obey allowed files in task-scope.json. Stop on Hard Stop conditions. After implementation run:

```bash
git diff --name-only
bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh <TASK-ID>
bash .spec/microservice-ddd-evolution/scripts/agent-check.sh
```

Run TASK-specific checks listed in the acceptance file. Create or update:

- .spec/microservice-ddd-evolution/reports/<TASK-ID>-report.md
- .spec/microservice-ddd-evolution/reports/<TASK-ID>-evidence.json

Report real failures and skipped checks honestly.
