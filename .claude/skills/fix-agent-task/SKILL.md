---
name: fix-agent-task
description: Thin Claude adapter for spec-harness fix-check-failures on one .spec feature TASK.
---

# fix-agent-task

Use this adapter only to repair failures from one canonical `.spec/<feature-name>` TASK.

Required input:

- `Feature: <feature-name>`
- `Task: <TASK-ID>`
- Previous Harness, test, or self-review failure output

Canonical mapping:

```text
spec-harness
Mode: fix-check-failures
Feature: <feature-name>
Task: <TASK-ID>
```

Required behavior:

1. Read `AGENTS.md`.
2. Read `.agents/skills/spec-harness/SKILL.md`.
3. Read the feature SPEC, SDD, TASKS, AGENT_RULES, task-scope, acceptance file, report, and failed output.
4. Fix only the listed failure items.
5. Modify only files allowed by the current TASK scope.
6. Re-run the failed check plus the required Harness checks.
7. Update the canonical report and evidence JSON.

Do not add unrelated functionality, broaden scope, or create provider-private completion state.

