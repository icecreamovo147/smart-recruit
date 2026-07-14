Use spec-harness.

Mode: implement-task
Feature: legacydomain-retirement
Task: <TASK-ID>

Read before editing:

- AGENTS.md
- .agents/skills/spec-harness/SKILL.md
- .spec/legacydomain-retirement/legacydomain-retirement-SPEC.md
- .spec/legacydomain-retirement/legacydomain-retirement-SDD.md
- .spec/legacydomain-retirement/TASKS.md
- .spec/legacydomain-retirement/AGENT_RULES.md
- .spec/legacydomain-retirement/task-scope.json
- .spec/legacydomain-retirement/acceptance/<TASK-ID>.md
- .knowledge/README.md and selected active knowledge per manifest routes

Implement exactly one TASK. Do not execute later TASKs. Obey allowed files in task-scope.json. Stop on Hard Stop conditions. Every TASK requires knowledge impact reporting.

After implementation run:

```bash
git diff --name-only
bash .spec/legacydomain-retirement/scripts/check-task-scope.sh <TASK-ID>
bash .spec/legacydomain-retirement/scripts/agent-check.sh
```

Run TASK-specific checks listed in the acceptance file. Create or update:

- .spec/legacydomain-retirement/reports/<TASK-ID>-report.md
- .spec/legacydomain-retirement/reports/<TASK-ID>-evidence.json

Report real failures and skipped checks honestly.
