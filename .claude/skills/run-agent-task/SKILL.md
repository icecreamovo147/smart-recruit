---
name: run-agent-task
description: Thin Claude adapter for spec-harness implement-task on one .spec feature TASK.
---

# run-agent-task

Use this adapter only to execute one canonical `.spec/<feature-name>` TASK.

Required input:

- `Feature: <feature-name>`
- `Task: <TASK-ID>`

Canonical mapping:

```text
spec-harness
Mode: implement-task
Feature: <feature-name>
Task: <TASK-ID>
```

Required behavior:

1. Read `AGENTS.md`.
2. Read `.agents/skills/spec-harness/SKILL.md`.
3. Read the feature SPEC, SDD, TASKS, AGENT_RULES, task-scope, acceptance file, and `prompts/implement-task.md`.
4. Modify only files allowed by the current TASK scope.
5. Run the required Harness checks and TASK-specific checks.
6. Write the canonical Markdown report and evidence JSON under `.spec/<feature-name>/reports/`.

Do not create provider-private task state, branch policy, completion rules, or review verdicts. Do not commit, push, merge, or update local Claude permissions.

