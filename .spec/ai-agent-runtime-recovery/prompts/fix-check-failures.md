# Fix Check Failures Prompt - ai-agent-runtime-recovery

Use spec-harness.

Mode: fix-check-failures
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
- the failed Harness, test, or self-review output

## Instructions

1. Fix only the failed items.
2. Do not expand TASK scope.
3. Do not add unrelated functionality.
4. Do not modify SPEC or SDD.
5. Stop if the fix requires files outside scope or a Hard Stop confirmation.
6. Prefer a focused subagent for failure investigation when useful, but the controlling agent must make final scope and evidence decisions.
7. Re-run the originally failed check.
8. Re-run:

```bash
git diff --name-only
bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh <TASK-ID>
bash .spec/ai-agent-runtime-recovery/scripts/agent-check.sh
```

9. Update `.spec/ai-agent-runtime-recovery/reports/<TASK-ID>-report.md` and evidence with repair summary and remaining risks.
