# Self Review Prompt

Use `spec-harness`.

Mode: `self-review`

Feature: `hr-agent-resumable-stream`

Task: `<TASK-ID>`

Read:

- `AGENTS.md`
- `.agents/skills/spec-harness/SKILL.md`
- `.spec/hr-agent-resumable-stream/hr-agent-resumable-stream-SPEC.md`
- `.spec/hr-agent-resumable-stream/hr-agent-resumable-stream-SDD.md`
- `.spec/hr-agent-resumable-stream/TASKS.md`
- `.spec/hr-agent-resumable-stream/AGENT_RULES.md`
- `.spec/hr-agent-resumable-stream/task-scope.json`
- `.spec/hr-agent-resumable-stream/acceptance/<TASK-ID>.md`
- `.spec/hr-agent-resumable-stream/reports/<TASK-ID>-report.md`
- Current `git diff`

Review without modifying files. Check scope, SPEC alignment, SDD alignment, acceptance coverage, tests, compatibility, and knowledge impact reporting.

End with exactly one verdict:

```text
verdict: 通过
```

or:

```text
verdict: 不通过
```
