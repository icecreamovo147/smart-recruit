---
name: spec-harness
description: Create, review, execute, amend, and verify Specification-Driven feature contracts under the repository .spec directory. Use for SPEC/SDD/TASK planning, one-TASK implementation and review, recoverable scope or contract gaps, and behavior-preserving refactors or migrations that require pinned baselines and parity evidence.
---

# spec-harness

## Authority

Use one canonical control plane:

1. the nearest `AGENTS.md` for durable repository rules;
2. this skill for feature contracts and the single-TASK lifecycle;
3. `harness-pipeline` only for explicitly requested multi-TASK orchestration;
4. `.spec/<feature-name>/` for the executable contract, runtime state, evidence, and reports.

Do not let provider prompts, historical plans, memories, or reports redefine requirements, TASK scope, Review outcomes, or completion.

## Profiles and schema

Use `schemaVersion: 2` for every new feature. Select exactly one profile in `contract.json`:

- `feature_delivery`: create or change product behavior described by the approved contract.
- `behavior_preserving_migration`: restructure, extract, replace, cut over, or retire code while preserving a pinned source behavior baseline.

Read [references/contract-v2.md](references/contract-v2.md) completely when creating, reviewing, amending, or executing a v2 feature. Also read [references/behavior-preserving-migration.md](references/behavior-preserving-migration.md) completely for the migration profile.

Compatibility policy:

- Existing schema v1 non-migration features remain executable without silent rewriting.
- Completed schema v1 features remain readable historical evidence.
- Schema-less features are `legacy-compatible` for audit only.
- Pending migration, cutover, or source-retirement work must use schema v2.

## Modes

| Mode | Purpose |
|---|---|
| `init-feature` | Discover the repository, create SPEC/SDD and a reviewed v2 Harness. |
| `draft-spec-sdd` | Create only SPEC/SDD and discovery evidence. |
| `prepare-harness` | Create the v2 executable contract from approved SPEC/SDD. |
| `review-plan` | Independently review assumptions, coverage, TASK boundaries, and checks. |
| `implement-task` | Implement exactly one `ready` TASK. |
| `self-review` | Review implementation and contract adequacy without modifying files. |
| `fix-check-failures` | Fix only an `implementation_defect`. |
| `propose-amendment` | Record a scope, plan, requirement, or baseline correction as a CR. |
| `review-amendment` | Independently validate CR evidence, impact, and approval level. |
| `apply-amendment` | Apply an approved CR as a new contract revision. |
| `reconcile-plan` | Revalidate assumptions and promote the next draft TASKs to `ready`. |
| `verify-feature` | Run cumulative contract, behavior, and completion gates. |

Required prompt inputs:

```text
Mode: <mode>
Feature: <feature-name>
Task: <TASK-ID, for TASK modes>
Change Request: <CR-ID, for amendment modes>
Feature Prompt: <for init-feature or draft-spec-sdd>
```

## Universal invariants

- Execute and modify business code for only one TASK at a time.
- Treat `readScope`, `writeScope`, and `verificationScope` as different boundaries. Read broadly enough to understand the contract; write only within `allowedFiles`.
- For migrations, read the pinned reference commit read-only. Never let the baseline float with a branch name.
- Do not let an ordinary TASK modify SPEC, SDD, TASKS, `contract.json`, `traceability.json`, `task-scope.json`, acceptance, baseline manifests, or CR files.
- Let amendment modes modify contract-owned files only after the required approval.
- Treat reports, evidence, and `pipeline-state.json` as orchestrator-owned runtime artifacts, not business scope.
- Do not silently invent requirements, remove behavior, weaken acceptance, or redefine missing behavior as expected.
- Do not use `Risk`, `Follow-up`, `debt`, `unsupported`, empty success, or explicit failure to satisfy a mandatory requirement.
- Do not use static, build, registration, or mock-only evidence to claim integration, E2E, or behavioral parity.
- A skipped blocking check is a blocker, not a pass or exception.
- Keep user-visible behavior changes, schema changes, security changes, cutover, and source deletion behind L2 user approval.
- Preserve unrelated dirty changes. Stop if a reliable TASK base tree cannot isolate this TASK.

## Planning and rolling readiness

Before writing SPEC/SDD or TASKS:

1. Inspect the current implementation, callers, interfaces, tests, data, runtime wiring, and operational dependencies.
2. Record assumptions as `verified`, `open`, or `rejected` with evidence. Do not approve a plan with an open blocking assumption.
3. Give stable IDs to mandatory requirements and behavior surfaces.
4. Map every mandatory item through design, TASK, acceptance, required check, and eventual evidence in `traceability.json`.
5. Use an independent plan Reviewer for high-risk, cross-service, migration, cutover, or deletion work.
6. Keep a complete capability backlog, but freeze only the next one or two TASKs as `ready`; leave later TASKs `draft`.
7. Split TASKs by reviewable behavior. A single TASK must not hide a broad cross-system migration merely because it runs serially.

Use vertical user journeys for product behavior. Foundation or architecture-skeleton TASKs may exist, but they cannot claim that a user capability is migrated or verified.

## `init-feature` and `draft-spec-sdd`

Create `.spec/<feature-name>/` only after inspecting relevant repository evidence.

SPEC must define goals, non-goals, user behavior, functional and non-functional requirements, compatibility, errors/fallbacks, security, observability, acceptance, assumptions, and open questions.

SDD must define existing architecture, problem analysis, design, data/API/workflow/configuration changes, compatibility, errors, observability, testing, risks, boundaries, alternatives, assumptions, and open questions.

For `draft-spec-sdd`, stop after SPEC/SDD and discovery evidence. Do not create TASKs or modify business code.

For `init-feature`, continue through `prepare-harness` and `review-plan`; do not mark implementation ready until plan Review passes.

## `prepare-harness`

Create the canonical v2 package described in [references/contract-v2.md](references/contract-v2.md), including:

- `contract.json`, `traceability.json`, TASKS, rules, scope, acceptance, prompts, scripts, reports, `changes/`, `revisions/`, and `reviews/`;
- one stable ID for every required check;
- explicit task lifecycle, dependencies, risk, review policy, destructive actions, and completion gates;
- `baseline/behavior-manifest.json` for migrations.

Run:

```bash
node .agents/skills/spec-harness/scripts/validate-feature.mjs --feature .spec/<feature-name>
```

Stop on missing traceability, unresolved blocking assumptions, invalid profile artifacts, broad contract-directory write scope, or SPEC/SDD conflicts.

## `review-plan`

Use an independent read-only Agent or fresh context when available. Review the repository and pinned baseline, not only the generated documents.

Verify:

- requirement and behavior-surface completeness;
- source references and assumptions;
- vertical closure and real runtime reachability;
- TASK size, dependencies, and write scope;
- acceptance strength and check kind;
- cumulative integration ownership;
- migration parity, cutover, rollback, and deletion separation.

Write a machine-readable plan Review under `reviews/`. Return `pass`, `contract_gap`, or `requirement_conflict`. Only `pass` may set `planningStatus=approved` and promote TASKs to `ready`.

## `implement-task`

Preflight:

1. Validate the feature and confirm the TASK lifecycle is `ready`.
2. Read the active contract revision, traceability, TASK, acceptance, and relevant knowledge.
3. Record `base_sha`, a reliable `base_tree`, `contract_revision`, `task_definition_hash`, and `traceability_hash`.
4. Check human confirmation and independent Review requirements.
5. Inspect the whole read/verification scope before modifying files.

Implement only the TASK. If actual work exceeds scope or the contract is incomplete, do not improvise and do not force a local fix; return `contract_gap` and route to `propose-amendment`.

After implementation run:

```bash
git diff --name-only
bash .spec/<feature-name>/scripts/check-task-scope.sh <TASK-ID>
bash .spec/<feature-name>/scripts/agent-check.sh
```

Run every declared required check. Create matching Markdown report and schema v2 evidence. Each required check ID must appear exactly once with its actual status, command, exit code, timing, coverage, and artifact. Record skipped advisory checks with reasons; blocking checks may not be skipped.

Validate the Evidence against the TASK declaration:

```bash
node .agents/skills/spec-harness/scripts/validate-evidence.mjs \
  --file .spec/<feature-name>/reports/<TASK-ID>-evidence.json \
  --task-scope .spec/<feature-name>/task-scope.json \
  --task <TASK-ID>
```

## `self-review`

Review both implementation correctness and contract adequacy. Inspect current diff, runtime reachability, callers, integration boundaries, and pinned source behavior where applicable.

Return exactly one structured outcome:

- `pass`: implementation and evidence satisfy the active contract;
- `implementation_defect`: contract is adequate; implementation or tests are wrong;
- `contract_gap`: scope, TASK, acceptance, design, or traceability is incomplete;
- `requirement_conflict`: authoritative requirements or baselines conflict;
- `environment_blocker`: mandatory evidence cannot currently be produced.

Use `verdict: 通过` only with `outcome: pass`; all other outcomes use `verdict: 不通过`. A passing Review cannot contain unresolved mandatory coverage, failed checks, skipped blocking checks, reachable stubs, or migration parity gaps.

## `fix-check-failures`

Run only for `implementation_defect`. Fix only reported implementation/test failures within the active scope and contract revision, then rerun the failed checks and Review.

Do not use this mode for `contract_gap`, `requirement_conflict`, or `environment_blocker`.

## Amendment modes

When the plan is wrong, preserve the boundary and revise it explicitly:

1. `propose-amendment`: create `changes/CR-XXXX.json` and `.md` with trigger evidence, base revision, classification, before/after scope, impacted requirements/TASKs, approval level, and revalidation plan.
2. Validate the CR:

```bash
node .agents/skills/spec-harness/scripts/validate-amendment.mjs --file .spec/<feature-name>/changes/CR-XXXX.json
```

3. `review-amendment`: independently confirm that the CR fixes the observed gap without hiding behavior loss.
4. Apply approval policy:
   - L0: local scope correction with no behavior change; independent Reviewer may approve.
   - L1: TASK split, cross-package/service scope, or design/acceptance detail; independent Review, plus user approval when high risk.
   - L2: requirement, baseline, public API, schema, dependency, security, behavior change, cutover, or deletion; user approval required.
5. Stage complete replacement files under `changes/<CR-ID>/`, record their before/after SHA-256 values in the CR, then dry-run and apply atomically:

```bash
node .agents/skills/spec-harness/scripts/apply-amendment.mjs \
  --feature-dir .spec/<feature-name> \
  --cr .spec/<feature-name>/changes/CR-XXXX.json

node .agents/skills/spec-harness/scripts/apply-amendment.mjs \
  --feature-dir .spec/<feature-name> \
  --cr .spec/<feature-name>/changes/CR-XXXX.json \
  --apply
```

6. `apply-amendment` updates all staged contract artifacts, increments `contractRevision`, stores a revision manifest, updates runtime revision state, marks impacted completed TASKs `needs_revalidation`, and rolls back if the resulting feature contract is invalid.
7. Re-run feature, traceability, parity, and pipeline validators before resuming.

Never let a TASK directly edit its own scope. Never use an exception where an Amendment is required.

## `reconcile-plan`

After every passing TASK:

1. Compare actual code and evidence with assumptions and downstream TASK definitions.
2. Check whether new dependencies, behavior surfaces, or integration owners were discovered.
3. If unchanged, promote the next draft TASKs to `ready` within the rolling window.
4. If changed, create a CR and enter `amendment_pending`.
5. Revalidate completed TASKs affected by the new revision before trusting their evidence.

## `verify-feature`

Run cumulative checks from the feature base tree, not only the last working-tree diff. Validate:

```bash
node .agents/skills/spec-harness/scripts/validate-feature.mjs --feature .spec/<feature-name> --require-pipeline
node .agents/skills/spec-harness/scripts/validate-traceability.mjs --file .spec/<feature-name>/traceability.json --require-verified
node .agents/skills/harness-pipeline/scripts/validate-pipeline-state.mjs --feature-dir .spec/<feature-name>
```

For migrations, also run:

```bash
node .agents/skills/spec-harness/scripts/validate-parity.mjs --file .spec/<feature-name>/baseline/behavior-manifest.json --require-verified
```

Do not report completion with open CRs, unresolved blocking assumptions, `needs_revalidation` TASKs, missing evidence, insufficient completion level, or incomplete migration parity.

## Hard Stops

Stop ordinary implementation and classify the reason instead of silently continuing when:

- the TASK is not `ready`, scope is unreliable, or unrelated dirty changes overlap;
- required behavior is absent or conflicting;
- a wider write scope or contract-owned file change is required;
- a public API, schema, shared module, dependency, global configuration, auth/security rule, cutover, or deletion was not approved;
- required evidence cannot be produced truthfully;
- a migration baseline is unpinned or parity inventory is incomplete.

Route contract problems to Amendment. Route environment problems to `environment_blocker`. Ask the user only for L2 or otherwise explicitly human-gated decisions.

## Final output

Report the mode, feature, TASK/CR, active revision, modified files, checks and evidence, Review outcome, completion level, risks, blockers, revalidation impact, and recommended next mode. Do not silently continue into another mode except `init-feature` or explicit pipeline orchestration.
