# Prelaunch Security Fix Execution Contract

## Contents

1. Inputs and authority
2. Default full-plan scheduling
3. Baseline and drift
4. Runtime artifacts
5. Task evidence
6. Final outcomes

## 1. Inputs and authority

The authoritative input is a generated `*-remediation-plan.md`. Prefer a plan containing a fenced JSON object with:

```json
{
  "schema_version": 1,
  "artifact_type": "prelaunch-security-remediation-plan",
  "audit": {},
  "findings": [],
  "remediation": {"strategy": [], "tasks": []}
}
```

The parser computes the Markdown file SHA-256. The original plan stays immutable during a run. A plan change invalidates resume until the user chooses whether to start a new run.

A generated plan grants repository implementation authority only within each task's `allowed_paths`. It does not grant authority for production, external systems, secret rotation, destructive operations, commits, pushes, PRs, deployment, public contract changes, schema changes, shared security policy, global configuration, or scope expansion. Apply ordinary user instructions and repository rules in addition to the plan.

## 2. Default full-plan scheduling

An unqualified invocation means `run-plan`.

1. Validate all IDs, references, paths, dependencies, and cycles.
2. Consider only tasks in the requested selection; default to all tasks.
3. A task is ready when its state is `PENDING` or `READY` and every dependency is `PASSED` or `ALREADY_RESOLVED`.
4. Sort ready tasks by `P0`, `P1`, `P2`, `P3`, then original plan position.
5. Execute exactly one task at a time.
6. After `PASSED` or `ALREADY_RESOLVED`, select the next task automatically.
7. A blocked task blocks descendants, not unrelated ready tasks, unless the failure invalidates repository or plan integrity.
8. End when every selected task reaches a terminal state or the whole-run stop condition fires.

Narrow modes are allowed only when explicitly requested:

- `run-task REM-NNN`
- `run-priority P0` or a priority set
- `resume`
- `verify-only`

For task or priority selection, automatically include transitive dependencies. Report those prerequisites before execution; do not silently treat an omitted dependency as satisfied.

## 3. Baseline and drift

Record at initialization:

- plan path and SHA-256;
- audit branch and commit;
- current branch and HEAD;
- initial `git status --porcelain` paths;
- task order and task contracts;
- creation/update timestamps.

Before each task:

- re-hash the plan;
- compare current HEAD and task-relevant diff with the audit baseline;
- inspect overlap between pre-existing dirty paths and task allowed paths;
- revalidate the security finding.

Classify drift:

- `NONE`: evidence and scope remain valid.
- `SAFE`: unrelated changes; record and continue.
- `TASK_RELEVANT`: requires manual inspection before continuing.
- `STALE`: finding, path, contract, or architecture no longer matches; stop the task.

## 4. Runtime artifacts

Use:

```text
.security-review/fix-runs/<plan-id>/
├── plan.json
├── fix-state.json
├── execution-log.md
├── REM-001-report.md
└── final-fix-summary.md
```

`fix-state.json` is the runtime source of truth. Use schema version 1:

```json
{
  "schema_version": 1,
  "run_id": "...",
  "plan": {
    "path": "...",
    "sha256": "...",
    "audit_commit": "..."
  },
  "repository": {
    "branch": "...",
    "initial_head": "...",
    "initial_worktree_paths": []
  },
  "run_status": "IN_PROGRESS",
  "selection": {"mode": "all", "values": []},
  "tasks": [
    {
      "id": "REM-001",
      "priority": "P0",
      "dependencies": [],
      "status": "PENDING",
      "allowed_paths": [],
      "excluded_paths": [],
      "modified_paths": [],
      "notes": [],
      "checks": [],
      "updated_at": "..."
    }
  ],
  "events": []
}
```

Write state atomically with a temporary file and rename. Do not edit it by hand while a run is active.

## 5. Task evidence

Require these gates before `PASSED`:

- linked finding revalidated;
- root cause fixed;
- all acceptance criteria evaluated;
- original negative case no longer succeeds;
- normal behavior regression tested;
- required module checks pass;
- changed paths pass scope validation;
- `git diff --check` passes;
- security self-review has no new P0/P1 issue;
- confirmation/external evidence requirements satisfied.

If a check was not run, record `NOT_RUN` or `BLOCKED`; do not record `PASS`.

## 6. Final outcomes

- `REMEDIATION_COMPLETE_PENDING_REAUDIT`: every selected task is `PASSED`, `ALREADY_RESOLVED`, or explicitly `ACCEPTED_RISK`, and default full-plan selection included all tasks.
- `REMEDIATION_PARTIAL`: at least one task passed, but unresolved terminal tasks remain.
- `REMEDIATION_BLOCKED`: no meaningful safe progress can continue.
- `REMEDIATION_ABORTED`: the user stopped the run or an integrity condition forced termination.

Only `prelaunch-security-audit` can issue a new `GO`, `CONDITIONAL_GO`, or `NO_GO` verdict.
