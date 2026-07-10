# Run Phase Migration Gate

`/run-phase` is retained only as a compatibility entrypoint for historical `.ai-guides/<phase-slug>` references. It must not directly implement `.ai-guides` content.

Usage:

```text
/run-phase <phase-slug>
/run-phase <phase-slug> --feature <feature-name>
/run-phase <phase-slug> --feature <feature-name> --task <TASK-ID>
```

If no argument is provided, use the historical default phase slug only for discovery and migration guidance:

```text
recruitment-mainline-phase-1-pipeline-state
```

## Gate Rules

1. Treat `.ai-guides/<phase-slug>/constitution.md`, `spec.md`, `plan.md`, and `tasks.md` as legacy input, not executable authority.
2. If no explicit `.spec` mapping is provided, stop without editing files and output:
   - legacy phase path;
   - canonical authority: `AGENTS.md`, `.agents/skills/spec-harness/SKILL.md`, `.spec/<feature-name>/`;
   - recommended next mode: `spec-harness draft-spec-sdd`;
   - reason: legacy phase must be revalidated against current code before implementation.
3. If `--feature <feature-name>` is provided, run canonical preflight against `.spec/<feature-name>/`.
4. If preflight fails, stop and report the validation failure.
5. If preflight passes and `--task <TASK-ID>` is provided, invoke canonical `spec-harness implement-task` for that TASK.
6. If preflight passes and no task is provided, invoke canonical `harness-pipeline` only when the user explicitly requested pipeline execution.

## Required Output When Blocked

```text
PHASE_GATE: blocked
LEGACY_PHASE: <phase-slug>
LEGACY_PATH: .ai-guides/<phase-slug>
CANONICAL_ENTRY: spec-harness draft-spec-sdd
REASON: No explicit valid .spec mapping was provided.
```

Do not create branches, commit, merge, update `.ai-guides`, or write provider-private phase status from this command.

