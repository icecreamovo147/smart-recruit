# Implement TASK Prompt

Use spec-harness.

Mode: implement-task
Feature: legacydomain-migration-finalization
Task: <TASK-ID>

Read:

- `AGENTS.md`
- `.agents/skills/spec-harness/SKILL.md`
- `.spec/legacydomain-migration-finalization/legacydomain-migration-finalization-SPEC.md`
- `.spec/legacydomain-migration-finalization/legacydomain-migration-finalization-SDD.md`
- `.spec/legacydomain-migration-finalization/TASKS.md`
- `.spec/legacydomain-migration-finalization/AGENT_RULES.md`
- `.spec/legacydomain-migration-finalization/task-scope.json`
- `.spec/legacydomain-migration-finalization/acceptance/<TASK-ID>.md`
- `.knowledge/README.md`

Obey the TASK scope exactly. Stop on Hard Stop conditions. Report `knowledge_impact`.
