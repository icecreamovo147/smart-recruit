---
name: batch-agent-coordinator
description: Thin Claude adapter for canonical harness-pipeline over a .spec feature.
---

# batch-agent-coordinator

Use this adapter only when the user explicitly asks for serial multi-TASK orchestration over an existing `.spec/<feature-name>` contract.

Required input:

- `Feature: <feature-name>`

Optional input:

- `Start Task: <TASK-ID>`
- `Max Review Rounds: <n>`
- `Skip Human Confirm: true|false`

Canonical mapping:

```text
harness-pipeline
feature_name: <feature-name>
start_task: <TASK-ID, optional>
max_review_rounds: <n, optional>
skip_human_confirm: <boolean, optional>
```

Required behavior:

1. Read `AGENTS.md`.
2. Read `.agents/skills/harness-pipeline/SKILL.md`.
3. Read `.agents/skills/spec-harness/SKILL.md` because pipeline delegates single-TASK modes to it.
4. Validate the target `.spec/<feature-name>/` preflight before orchestration.
5. Let canonical `harness-pipeline` own TASK ordering, review/fix rounds, runtime state, evidence, completion rules, and final summary.

Do not redefine the TASK loop, review verdict, repair limit, branch policy, merge behavior, or pipeline state transitions in this adapter.

