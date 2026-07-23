# Self Review Prompt - ai-agent-runtime-recovery

Use spec-harness.

Mode: self-review
Feature: ai-agent-runtime-recovery
Task: <TASK-ID>

## Required Reading

Read:

- `AGENTS.md`
- `.agents/skills/spec-harness/SKILL.md`
- `.spec/ai-agent-runtime-recovery/ai-agent-runtime-recovery-SPEC.md`
- `.spec/ai-agent-runtime-recovery/ai-agent-runtime-recovery-SDD.md`
- `.spec/ai-agent-runtime-recovery/TASKS.md`
- `.spec/ai-agent-runtime-recovery/AGENT_RULES.md`
- `.spec/ai-agent-runtime-recovery/task-scope.json`
- `.spec/ai-agent-runtime-recovery/acceptance/<TASK-ID>.md`
- `.spec/ai-agent-runtime-recovery/reports/<TASK-ID>-report.md`

## Instructions

1. Review the current TASK diff without modifying files.
2. Prefer an independent read-only subagent/fresh context when available.
3. Compare the diff against SPEC, SDD, TASKS, acceptance, scope, knowledge impact, and reported tests.
4. Check for scope violations, missing tests, incomplete acceptance criteria, unsafe assumptions, hidden public API changes, weak fallback/error handling, and insufficient observability.
5. End with exactly one verdict:

```text
verdict: 通过
```

or:

```text
verdict: 不通过
```

If not passing, provide concrete required fixes.
