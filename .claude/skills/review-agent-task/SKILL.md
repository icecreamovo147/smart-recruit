---
name: review-agent-task
description: Thin Claude adapter for read-only spec-harness self-review on one .spec feature TASK.
---

# review-agent-task

Use this adapter only to review one canonical `.spec/<feature-name>` TASK.

Required input:

- `Feature: <feature-name>`
- `Task: <TASK-ID>`

Canonical mapping:

```text
spec-harness
Mode: self-review
Feature: <feature-name>
Task: <TASK-ID>
```

Required behavior:

1. Read `AGENTS.md`.
2. Read `.agents/skills/spec-harness/SKILL.md`.
3. Read the feature SPEC, SDD, TASKS, AGENT_RULES, task-scope, acceptance file, report, evidence, and current diff.
4. Do not modify files.
5. Prefer an independent Claude subagent or fresh context when available.
6. If review is not independent, disclose `reviewer_type: self-review`.
7. End with exactly one canonical verdict:

```text
verdict: 通过
```

or:

```text
verdict: 不通过
```

Do not emit provider-private status labels as the authoritative verdict.
