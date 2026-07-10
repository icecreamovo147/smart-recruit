# Run SPEC+SDD Phase

Run an implementation/review loop for one recruitment mainline phase.

Usage:

```text
/run-phase recruitment-mainline-phase-1-pipeline-state
```

If no argument is provided, use:

```text
recruitment-mainline-phase-1-pipeline-state
```

## Orchestrator Instructions

You are the orchestrator. Use `phase-implementer` as Agent A and `phase-reviewer` as Agent B.

Let:

```text
PHASE_SLUG = $ARGUMENTS
```

If `$ARGUMENTS` is empty:

```text
PHASE_SLUG = recruitment-mainline-phase-1-pipeline-state
```

## Preflight

1. Confirm `.ai-guides/<PHASE_SLUG>/constitution.md`, `spec.md`, `plan.md`, and `tasks.md` exist.
2. Run `git status --short`.
3. If unrelated dirty files exist, stop and ask the user how to proceed. Do not mix phase work with unrelated local changes.
4. Create or switch to a local phase branch named:

   ```text
   codex/<PHASE_SLUG>
   ```

   If the branch already exists, switch to it. If it does not exist, create it from the current branch.
5. After switching branches, run `git status --short` again and confirm the worktree is clean or contains only expected phase artifacts.
6. Confirm the phase folder is the only SDD phase being implemented.

## Loop

Run at most 5 iterations.

For each iteration:

1. Invoke `phase-implementer` with:
   - `PHASE_SLUG`
   - current iteration number
   - previous reviewer `BLOCKING` findings, if any
2. After Agent A returns, run relevant verification commands based on changed files.
3. Save or summarize verification output for Agent B.
4. Invoke `phase-reviewer` with:
   - `PHASE_SLUG`
   - iteration number
   - git diff summary
   - verification output
5. If reviewer returns `DECISION: PASS`, stop the loop.
6. If reviewer returns `DECISION: NEEDS_WORK`, pass `NEXT_PROMPT_FOR_IMPLEMENTER` to the next implementer iteration.
7. If five iterations complete without PASS, stop and report `STOPPED`.

## Required Stop Conditions

Stop and ask the user before continuing if any of these happen:

- The worktree has unrelated uncommitted changes before branch creation.
- The target phase branch already exists but contains unexpected unrelated changes.
- A destructive database reset is needed.
- A dependency upgrade is needed.
- A migration strategy conflicts with the phase spec.
- A change would modify multiple future phase ownership areas.
- Tests cannot run because of external services and no safe local fallback exists.
- The reviewer finds the same blocking issue after two repair attempts.

## Final Output

Create or update:

```text
.ai-guides/<PHASE_SLUG>/delivery-report.md
.ai-guides/<PHASE_SLUG>/acceptance-review.md
```

Final response must include:

```text
PHASE_STATUS: PASS | STOPPED | BLOCKED
PHASE_SLUG: <phase-slug>
BRANCH: codex/<phase-slug>
ITERATIONS: <n>
VERIFICATION:
- ...
REVIEW_DECISION:
- ...
NEXT_STEP:
- ...
```
