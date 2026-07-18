---
name: harness-pipeline
description: Serially orchestrate all TASKs in a spec-harness feature, including typed Review routing, implementation repair, contract amendments, rolling-plan reconciliation, cumulative verification, and migration parity or retirement gates. Use only when the user explicitly requests multi-TASK pipeline execution.
---

# harness-pipeline

## Authority and prerequisites

This skill orchestrates `spec-harness`; it does not redefine TASK implementation, Review, Amendment, scope, evidence, or completion semantics.

Before execution, read the active `spec-harness` SKILL completely. For schema v2, also read its contract reference; for `behavior_preserving_migration`, read the migration reference.

Require:

- a valid `.spec/<feature>/` package;
- SPEC, SDD, TASKS, rules, scope, acceptance, prompts, scripts, and reports;
- an approved v2 contract and traceability for new features;
- a pinned baseline and behavior manifest for migrations;
- no unrelated dirty changes that overlap the feature;
- a reliable Git base tree for every TASK.

Input:

```text
Feature: <feature-name>
Start Task: <optional TASK-ID>
Max Review Rounds: <default 3>
Skip Human Confirm: <default false>
```

`skip_human_confirm=true` means block rather than bypass a human gate.

## Compatibility

- Resume existing schema v1 non-migration pipelines with their original state semantics and strengthened evidence validation.
- Do not silently upgrade schema v1 state or evidence.
- Do not resume schema-less pipelines for implementation.
- Require schema v2 for pending migration, cutover, or source-retirement work.

## Pipeline model

The v2 loop is:

```text
discovery / plan review
→ ready TASK
→ implement
→ typed review
   ├─ pass                   → complete TASK → reconcile plan
   ├─ implementation_defect  → fix → review
   ├─ contract_gap           → amendment_pending
   ├─ requirement_conflict   → amendment_pending + user decision
   └─ environment_blocker    → blocked at current completion level
→ cumulative final verification
→ completed only at the contract's required completion level
```

Do not send `contract_gap`, `requirement_conflict`, or `environment_blocker` into the ordinary fix loop.

## State ownership

`pipeline-state.json` is the runtime source of truth. `task-scope.json.tasks[*].lifecycle` is the contract-defined readiness state; it is not a substitute for runtime evidence.

Schema v2 state includes:

```json
{
  "schemaVersion": 2,
  "feature_name": "example",
  "contract_revision": 1,
  "current_task": "TASK-001",
  "current_phase": "implement",
  "completion_level": "code_complete",
  "completed_tasks": [],
  "failed_tasks": [],
  "blocked_tasks": [],
  "skipped_human_confirmation_tasks": [],
  "approved_exceptions": [],
  "open_change_requests": [],
  "needs_revalidation_tasks": [],
  "review_round": 0,
  "status": "in_progress",
  "task_runs": {}
}
```

Allowed status:

- `pending`
- `in_progress`
- `blocked`
- `completed`
- `completed_with_exceptions`

Allowed v2 phase:

- `pending`
- `discovery`
- `plan_review`
- `ready`
- `implement`
- `review`
- `fix`
- `amendment_pending`
- `amendment_review`
- `replan`
- `revalidate`
- `finalize`
- `final_review`
- `completed`
- `blocked`

## Preflight

Run:

```bash
node .agents/skills/spec-harness/scripts/validate-feature.mjs \
  --feature .spec/<feature-name> \
  --require-pipeline
```

Then:

1. Load the active contract revision, profile, traceability, TASK order, and dependencies.
2. Confirm there are no unresolved blocking assumptions.
3. Confirm the next TASK is `ready`; do not execute `draft`, `amendment_pending`, `needs_revalidation`, or `superseded` TASKs.
4. If `pipeline-state.json` exists, resume its exact phase. Do not trust a stale TASK definition hash or contract revision.
5. If status is completed, validate it again before reporting completion.

## TASK baseline

Before any TASK write:

- record `base_sha`;
- create or record a reliable `base_tree` that includes earlier completed TASK work;
- compute the active TASK definition hash and traceability hash;
- bind the run to `contract_revision`;
- record implementer run identity when independent Review may be required.

Use the same baseline for scope, report, and evidence. Stop if it cannot be reproduced.

## Human and destructive gates

Before implementation, inspect `requiresHumanConfirmation` and `destructiveActions`.

If confirmation is required:

- stop and request explicit user approval;
- record approver, time, and confirmation text in state and evidence;
- if `skip_human_confirm=true`, add the TASK to blocked and skipped lists and stop.

Never bypass confirmation with a generic exception.

Every destructive TASK requires independent Review. Cutover and source deletion must be different TASKs. Migration cutover/deletion additionally require a fully verified parity manifest.

## Implement

Set `current_phase=implement` and run `spec-harness` `implement-task` for exactly one TASK.

Run scope and project checks:

```bash
git diff --name-only
bash .spec/<feature-name>/scripts/check-task-scope.sh <TASK-ID>
bash .spec/<feature-name>/scripts/agent-check.sh
```

Run every declared required check. The Evidence must include every check ID with actual status. Static/build/registration checks cannot substitute for stronger required kinds.

Validate Evidence. The Pipeline validator will pass the TASK's declared checks to the canonical Evidence validator; do not infer `checks_status` from an Agent-authored summary.

## Typed Review routing

Set `current_phase=review` and run an independent Reviewer when required.

### `pass`

Require:

- `verdict: 通过`;
- all blocking checks passed;
- scope passed;
- no unresolved mandatory coverage;
- active revision and hashes match;
- required human and independent Review evidence exists.

Add the TASK to `completed_tasks`, record the run evidence, reset Review rounds, then enter `reconcile-plan`.

### `implementation_defect`

Set `current_phase=fix`, increment the Review round, and run `fix-check-failures`.

Stop as failed when the configured maximum is exceeded or the same repair fails twice. Do not broaden scope while fixing.

### `contract_gap`

Set:

```text
status=in_progress
current_phase=amendment_pending
```

Create a CR through `spec-harness propose-amendment`. Add it to `open_change_requests`. Do not modify business code or start a later TASK.

After approval and application:

- update `contract_revision`;
- supersede invalid TASK definitions;
- add affected completed TASKs to `needs_revalidation_tasks`;
- enter `replan`, then `revalidate` as required;
- resume only after validators pass and the next TASK is `ready`.

### `requirement_conflict`

Follow the Amendment path with L2 user approval. Preserve both conflicting sources and evidence. Do not choose silently.

### `environment_blocker`

Set `status=blocked`, `current_phase=blocked`, record the blocker and current completion level, and stop. Mandatory live or parity checks cannot be converted to advisory after execution begins.

## Reconcile the rolling plan

After each passing TASK:

1. Recheck downstream assumptions, dependencies, runtime ownership, and acceptance.
2. Compare the actual diff and evidence with the next draft TASKs.
3. Promote only the next one or two valid TASKs to `ready`.
4. Generate a CR instead of silently rewriting a stale task boundary.
5. Revalidate any completed TASK affected by a new contract revision.

## Completion and verification

After all applicable TASKs pass:

1. Set `current_phase=finalize`.
2. Run cumulative feature tests from the feature base tree.
3. Validate traceability with `--require-verified`.
4. For migrations, validate parity with `--require-verified` and run the contract's integration, differential, E2E, cutover, or retirement gates.
5. Set the achieved `completion_level` truthfully.
6. Create a candidate completed state.
7. Run:

```bash
node .agents/skills/harness-pipeline/scripts/validate-pipeline-state.mjs \
  --feature-dir .spec/<feature-name>
```

Keep completed only if validation passes.

Ordinary completed requires:

- every non-superseded TASK completed;
- no failed or blocked TASKs;
- no open CRs or revalidation debt;
- active hashes and revisions match;
- every mandatory requirement verified or user-approved as a behavior delta;
- every blocking check passed;
- completion level meets the contract target;
- migration parity and destructive gates passed when applicable.

## Exceptions

`completed_with_exceptions` requires a scoped, explicit user approval with task IDs, object, approver, time, reason, and category.

Exceptions cannot waive:

- mandatory migration parity;
- an unpinned or changed baseline;
- source deletion safety;
- missing human confirmation;
- open contract gaps that require Amendment.

Use `approved_delta` through an L2 Contract Amendment for intentional behavior changes.

## Pipeline summary

Write `.spec/<feature-name>/reports/pipeline-summary.md` with:

- schema, profile, contract revision, and baseline;
- required and achieved completion level;
- TASK states, Review outcomes, rounds, and evidence links;
- open/applied CRs and revalidation results;
- cumulative checks and skipped advisory checks;
- parity/cutover/retirement status for migrations;
- approved deltas and exceptions;
- unresolved risks and exact next action.

Do not describe code completion as behavior verification, cutover readiness, or retirement.

## Resume

On resume:

1. validate feature and state;
2. verify active revision and hashes;
3. continue the exact `current_phase`;
4. if amendment is open, resume amendment rather than implementation;
5. if revalidation is pending, complete it before new TASKs;
6. if completed, revalidate and report without rerunning implementation.
