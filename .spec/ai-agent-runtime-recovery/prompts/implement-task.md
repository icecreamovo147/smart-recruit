# Implement TASK Prompt - ai-agent-runtime-recovery

Use spec-harness.

Mode: implement-task
Feature: ai-agent-runtime-recovery
Task: <TASK-ID>

## Required Reading

Read these files before editing:

- `AGENTS.md`
- `.agents/skills/spec-harness/SKILL.md`
- `.spec/ai-agent-runtime-recovery/ai-agent-runtime-recovery-SPEC.md`
- `.spec/ai-agent-runtime-recovery/ai-agent-runtime-recovery-SDD.md`
- `.spec/ai-agent-runtime-recovery/TASKS.md`
- `.spec/ai-agent-runtime-recovery/AGENT_RULES.md`
- `.spec/ai-agent-runtime-recovery/task-scope.json`
- `.spec/ai-agent-runtime-recovery/acceptance/<TASK-ID>.md`
- `.knowledge/README.md`
- `.knowledge/manifest.yaml`
- `.knowledge/INDEX.md`
- routed active knowledge documents for the TASK scope

## Instructions

1. Classify the feature with `node .agents/skills/spec-harness/scripts/validate-feature.mjs --feature .spec/ai-agent-runtime-recovery`.
2. Establish a reliable `TASK_BASE_TREE`.
3. Check whether `<TASK-ID>` requires human confirmation in `task-scope.json`; stop until confirmation is recorded when required.
4. Inspect relevant current code and `origin/dev` reference files named by the TASK.
5. Implement exactly one TASK.
6. Do not modify files outside the TASK scope.
7. Do not modify SPEC or SDD.
8. Stop on Hard Stop conditions from `AGENT_RULES.md`.
9. Prefer subagents for focused implementation investigation when useful, but the controlling agent must enforce scope and run checks.
10. Run all required checks from the TASK acceptance file.
11. Run:

```bash
git diff --name-only
bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh <TASK-ID>
bash .spec/ai-agent-runtime-recovery/scripts/agent-check.sh
```

12. Create/update:

- `.spec/ai-agent-runtime-recovery/reports/<TASK-ID>-report.md`
- `.spec/ai-agent-runtime-recovery/reports/<TASK-ID>-evidence.json`

The report and evidence must include knowledge impact and must not claim failed or skipped checks passed.
